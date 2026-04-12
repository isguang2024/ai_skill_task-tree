package tasktree

import (
	"context"
	"strings"
)

// upsertSearchIndex inserts or replaces a search index entry.
func (a *App) upsertSearchIndex(ctx context.Context, entityType, entityID, taskID, title, content string) {
	// Delete existing entry first (FTS5 doesn't support UPDATE)
	_, _ = a.execContext(ctx, `DELETE FROM search_index WHERE entity_type = ? AND entity_id = ?`, entityType, entityID)
	text := strings.TrimSpace(content)
	if text == "" && strings.TrimSpace(title) == "" {
		return
	}
	_, _ = a.execContext(ctx, `INSERT INTO search_index(entity_type, entity_id, task_id, title, content) VALUES (?, ?, ?, ?, ?)`,
		entityType, entityID, taskID, strings.TrimSpace(title), text)
}

// indexTask indexes a task's title and goal.
func (a *App) indexTask(ctx context.Context, task jsonMap) {
	taskID := asString(task["id"])
	title := asString(task["title"])
	goal := asString(task["goal"])
	a.upsertSearchIndex(ctx, "task", taskID, taskID, title, goal)
}

// indexNode indexes a node's title and instruction.
func (a *App) indexNode(ctx context.Context, node jsonMap) {
	nodeID := asString(node["id"])
	taskID := asString(node["task_id"])
	title := asString(node["title"])
	instruction := asString(node["instruction"])
	a.upsertSearchIndex(ctx, "node", nodeID, taskID, title, instruction)
}

// indexNodeMemory indexes node memory's summary, execution_log, and structured fields.
func (a *App) indexNodeMemory(ctx context.Context, mem jsonMap) {
	nodeID := asString(mem["node_id"])
	taskID := asString(mem["task_id"])
	parts := []string{
		asString(mem["summary_text"]),
		asString(mem["execution_log"]),
		asString(mem["manual_note_text"]),
	}
	// Flatten JSON array fields
	for _, field := range []string{"conclusions", "decisions", "risks", "blockers", "next_actions", "evidence"} {
		if items := flattenJSONStringArray(mem[field]); items != "" {
			parts = append(parts, items)
		}
	}
	content := strings.Join(parts, "\n")
	a.upsertSearchIndex(ctx, "memory", nodeID, taskID, "memory:"+nodeID, content)
}

// smartSearch performs FTS5 full-text search with BM25 ranking.
func (a *App) smartSearch(ctx context.Context, q, scope, taskID string, limit int) (jsonMap, error) {
	if strings.TrimSpace(q) == "" {
		return jsonMap{"items": []jsonMap{}, "total": 0}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Build FTS5 query: tokenize input words, join with AND for precision
	ftsQuery := buildFTS5Query(q)

	query := `SELECT entity_type, entity_id, task_id, title, snippet(search_index, 4, '>>>', '<<<', '...', 48) AS snippet, bm25(search_index) AS rank FROM search_index WHERE search_index MATCH ?`
	args := []any{ftsQuery}

	if scope != "" && scope != "all" {
		query += ` AND entity_type = ?`
		args = append(args, scope)
	}
	if taskID != "" {
		query += ` AND task_id = ?`
		args = append(args, taskID)
	}

	query += ` ORDER BY rank LIMIT ?`
	args = append(args, limit)

	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		// FTS5 query syntax error → fall back to LIKE search
		return a.search(ctx, q, scope, limit)
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	return jsonMap{"items": items, "total": len(items)}, nil
}

// buildFTS5Query converts user input to FTS5 query syntax.
// "hello world" → "hello AND world"
// Handles Chinese by passing through directly (unicode61 tokenizer handles CJK).
func buildFTS5Query(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	// If user already uses FTS5 syntax (AND, OR, NOT, quotes), pass through
	upper := strings.ToUpper(trimmed)
	if strings.Contains(upper, " AND ") || strings.Contains(upper, " OR ") || strings.Contains(upper, " NOT ") || strings.Contains(trimmed, "\"") {
		return trimmed
	}
	// Split by whitespace, wrap each token with * for prefix matching
	words := strings.Fields(trimmed)
	parts := make([]string, 0, len(words))
	for _, w := range words {
		// Escape quotes in tokens
		w = strings.ReplaceAll(w, "\"", "")
		if w == "" {
			continue
		}
		parts = append(parts, "\""+w+"\"*")
	}
	if len(parts) == 0 {
		return trimmed
	}
	return strings.Join(parts, " AND ")
}

// flattenJSONStringArray extracts strings from a JSON array value (may be []any or []string or raw JSON string).
func flattenJSONStringArray(v any) string {
	switch arr := v.(type) {
	case []any:
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "; ")
	case []string:
		return strings.Join(arr, "; ")
	case string:
		return arr
	default:
		return ""
	}
}

// rebuildSearchIndex rebuilds the entire FTS5 index from existing data.
func (a *App) rebuildSearchIndex(ctx context.Context) error {
	if _, err := a.execContext(ctx, `DELETE FROM search_index`); err != nil {
		return err
	}
	// Index all tasks
	taskRows, err := a.queryContext(ctx, `SELECT * FROM tasks WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	tasks, err := scanRows(taskRows)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		a.indexTask(ctx, task)
	}
	// Index all nodes
	nodeRows, err := a.queryContext(ctx, `SELECT * FROM nodes WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	nodes, err := scanRows(nodeRows)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		a.indexNode(ctx, node)
	}
	// Index all node memories
	memRows, err := a.queryContext(ctx, `SELECT * FROM node_memory_current`)
	if err != nil {
		return err
	}
	mems, err := scanRows(memRows)
	if err != nil {
		return err
	}
	for _, mem := range mems {
		a.indexNodeMemory(ctx, mem)
	}
	return nil
}

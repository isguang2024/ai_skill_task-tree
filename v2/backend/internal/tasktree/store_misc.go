package tasktree

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func (a *App) getRemaining(ctx context.Context, taskID string) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	summary := summarizeLeafNodes(nodes)
	return jsonMap{
		"task_id":            taskID,
		"percent":            task["summary_percent"],
		"remaining_nodes":    summary.RemainingCount,
		"remaining_estimate": summary.RemainingEstimate,
		"blocked_nodes":      summary.BlockedCount,
		"paused_nodes":       summary.PausedCount,
		"canceled_nodes":     summary.CanceledCount,
		"next_ready_nodes":   summary.NextReady,
	}, nil
}

func (a *App) getResumeContext(ctx context.Context, taskID, nodeID string, eventLimit int) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(node["task_id"]) != taskID {
		return nil, &appError{Code: 400, Msg: "node does not belong to task"}
	}
	// Batch-fetch ancestor chain using recursive CTE (replaces N+1 findNode loop)
	ancestorChain, err := a.fetchAncestorChain(ctx, asString(node["parent_node_id"]))
	if err != nil {
		return nil, err
	}
	ancestors := make([]jsonMap, 0, len(ancestorChain))
	for _, parent := range ancestorChain {
		ancestors = append(ancestors, jsonMap{
			"node_id":        parent["id"],
			"parent_node_id": parent["parent_node_id"],
			"path":           parent["path"],
			"title":          parent["title"],
			"status":         parent["status"],
			"kind":           parent["kind"],
			"role":           parent["role"],
			"depth":          parent["depth"],
		})
	}
	rows, err := a.queryContext(ctx, `SELECT type, message, created_at, actor_type, actor_id FROM events WHERE node_id = ? ORDER BY created_at DESC LIMIT ?`, nodeID, eventLimit)
	if err != nil {
		return nil, err
	}
	recentEvents, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	parentMatch := asString(node["parent_node_id"])
	rows, err = a.queryContext(ctx, `SELECT id, parent_node_id, path, title, status, progress FROM nodes WHERE task_id = ? AND id != ? AND deleted_at IS NULL AND COALESCE(parent_node_id, '') = ? ORDER BY sort_order, created_at`, taskID, nodeID, parentMatch)
	if err != nil {
		return nil, err
	}
	siblingsAll, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	siblings := []jsonMap{}
	for _, item := range siblingsAll {
		siblings = append(siblings, jsonMap{
			"node_id":        item["id"],
			"parent_node_id": item["parent_node_id"],
			"path":           item["path"],
			"title":          item["title"],
			"status":         item["status"],
			"progress":       item["progress"],
			"kind":           item["kind"],
			"role":           item["role"],
			"depth":          item["depth"],
		})
	}
	remaining, err := a.getRemaining(ctx, taskID)
	if err != nil {
		return nil, err
	}
	// Fetch node memory without execution_log (structured level)
	memory, _ := a.getNodeMemoryStructured(ctx, nodeID)
	// Fetch recent runs
	runs, _ := a.listNodeRuns(ctx, nodeID, 5)
	latestResultPayload := map[string]any{}
	if len(runs) > 0 {
		if structured := asAnyMap(runs[0]["structured_result"]); structured != nil {
			latestResultPayload = structured
		}
	}

	return jsonMap{
		"task": jsonMap{
			"id":              task["id"],
			"task_id":         task["id"],
			"task_key":        task["task_key"],
			"title":           task["title"],
			"goal":            task["goal"],
			"status":          task["status"],
			"result":          task["result"],
			"project_id":      task["project_id"],
			"summary_percent": task["summary_percent"],
			"version":         task["version"],
		},
		"node": jsonMap{
			"id":                  node["id"],
			"node_id":             node["id"],
			"task_id":             node["task_id"],
			"parent_node_id":      node["parent_node_id"],
			"path":                node["path"],
			"title":               node["title"],
			"instruction":         node["instruction"],
			"acceptance_criteria": node["acceptance_criteria"],
			"depends_on":          stringSliceFromAny(node["depends_on"]),
			"depends_on_count":    len(stringSliceFromAny(node["depends_on"])),
			"status":              node["status"],
			"result":              node["result"],
			"progress":            node["progress"],
			"estimate":            node["estimate"],
			"kind":                node["kind"],
			"version":             node["version"],
		},
		"memory":                memory,
		"recent_runs":           runs,
		"latest_result_payload": latestResultPayload,
		"siblings":              siblings,
		"ancestors":             ancestors,
		"recent_events":         recentEvents,
		"remaining":             remaining,
	}, nil
}

// fetchAncestorChain returns ordered ancestors (root → immediate parent) using a recursive CTE.
// This replaces the N+1 loop of findNode calls for walking the parent chain.
func (a *App) fetchAncestorChain(ctx context.Context, startParentID string) ([]jsonMap, error) {
	if startParentID == "" {
		return []jsonMap{}, nil
	}
	rows, err := a.queryContext(ctx, `WITH RECURSIVE chain(nid) AS (
		VALUES(?)
		UNION ALL
		SELECT n.parent_node_id FROM nodes n JOIN chain c ON n.id = c.nid
		WHERE n.parent_node_id IS NOT NULL AND n.parent_node_id != '' AND n.deleted_at IS NULL
	)
	SELECT n.id, n.parent_node_id, n.path, n.title, n.status, n.depth
	FROM chain c JOIN nodes n ON n.id = c.nid
	WHERE n.deleted_at IS NULL
	ORDER BY n.depth`, startParentID)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func (a *App) resumeTask(ctx context.Context, taskID string) (jsonMap, error) {
	return a.resumeTaskWithOptions(ctx, taskID, nodeListOptions{}, eventListOptions{}, resumeOptions{})
}

func (a *App) resumeTaskWithOptions(ctx context.Context, taskID string, nodeOpts nodeListOptions, eventOpts eventListOptions, opts resumeOptions) (jsonMap, error) {
	return a.buildResumeV2(ctx, taskID, nodeOpts, eventOpts, opts)
}

func limitWithFallback(limit, fallback int) int {
	if limit <= 0 {
		return fallback
	}
	return limit
}

// findNextExecutableFromNodes finds the next executable node from an already-loaded node list.
// This is the same logic as buildFocusNodes but without building the focus tree.
func findNextExecutableFromNodes(nodes []jsonMap, currentStageNodeID string) jsonMap {
	ordered := orderedExecutableLeaves(nodes)
	nodesByID := make(map[string]jsonMap, len(nodes))
	for _, n := range nodes {
		nodesByID[asString(n["id"])] = n
	}
	for _, node := range ordered {
		if currentStageNodeID != "" && asString(node["stage_node_id"]) != currentStageNodeID {
			continue
		}
		if asString(node["status"]) == "running" {
			return node
		}
	}
	for _, node := range ordered {
		if currentStageNodeID != "" && asString(node["stage_node_id"]) != currentStageNodeID {
			continue
		}
		if asString(node["status"]) == "ready" && dependsMet(node, nodesByID) {
			return node
		}
	}
	for _, node := range ordered {
		if (asString(node["status"]) == "running" || asString(node["status"]) == "ready") && dependsMet(node, nodesByID) {
			return node
		}
	}
	return nil
}

func orderedExecutableLeaves(nodes []jsonMap) []jsonMap {
	hasChildren := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		parentID := asString(node["parent_node_id"])
		if parentID != "" {
			hasChildren[parentID] = true
		}
	}
	leaves := make([]jsonMap, 0, len(nodes))
	for _, node := range nodes {
		if asString(node["kind"]) == "group" {
			continue
		}
		if hasChildren[asString(node["id"])] {
			// 兼容历史脏数据：kind=leaf 但已存在子节点，不应再视为可执行叶子。
			continue
		}
		leaves = append(leaves, node)
	}
	sort.Slice(leaves, func(i, j int) bool {
		return naturalPathLess(asString(leaves[i]["path"]), asString(leaves[j]["path"]))
	})
	return leaves
}

// naturalPathLess compares two path strings (e.g. "A/1/10" vs "A/1/2")
// using numeric comparison for each segment, so "2" < "10".
func naturalPathLess(a, b string) bool {
	aParts := strings.Split(a, "/")
	bParts := strings.Split(b, "/")
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		ai, errA := strconv.Atoi(aParts[i])
		bi, errB := strconv.Atoi(bParts[i])
		if errA == nil && errB == nil {
			if ai != bi {
				return ai < bi
			}
			continue
		}
		if aParts[i] != bParts[i] {
			return aParts[i] < bParts[i]
		}
	}
	return len(aParts) < len(bParts)
}

func (a *App) listWorkItems(ctx context.Context, status string, includeClaimed bool, limit int) ([]jsonMap, error) {
	parts := splitCSV(status)
	if len(parts) == 0 {
		parts = []string{"ready"}
	}
	query := `SELECT n.*, t.title AS task_title FROM nodes n JOIN tasks t ON t.id = n.task_id WHERE n.deleted_at IS NULL AND t.deleted_at IS NULL AND n.kind != 'group' AND n.status IN (` + placeholders(len(parts)) + `)`
	args := make([]any, 0, len(parts)+2)
	for _, part := range parts {
		args = append(args, part)
	}
	if !includeClaimed {
		query += ` AND (n.claimed_by_id IS NULL OR n.lease_until IS NULL OR n.lease_until < ?)`
		args = append(args, utcNowISO())
	}
	query += ` ORDER BY n.updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func buildWorkItemSummary(item jsonMap) jsonMap {
	return jsonMap{
		"id":          item["id"],
		"title":       item["title"],
		"status":      item["status"],
		"task_id":     item["task_id"],
		"task_title":  item["task_title"],
		"kind":        item["kind"],
		"instruction": item["instruction"],
		"path":        item["path"],
		"estimate":    item["estimate"],
		"sort_order":  item["sort_order"],
	}
}

func (a *App) search(ctx context.Context, q, kind string, limit int) (jsonMap, error) {
	result := jsonMap{"tasks": []jsonMap{}, "nodes": []jsonMap{}}
	if strings.TrimSpace(q) == "" {
		return result, nil
	}
	searchTasks := kind == "" || kind == "all" || kind == "tasks"
	searchNodes := kind == "" || kind == "all" || kind == "nodes"
	// Determine FTS5 scope
	scope := ""
	if searchTasks && !searchNodes {
		scope = "task"
	} else if searchNodes && !searchTasks {
		scope = "node"
	}
	// Use FTS5 full-text search to find matching entity IDs
	searchResult, err := a.smartSearch(ctx, q, scope, "", limit*2)
	if err != nil {
		// FTS5 unavailable — fall back to LIKE search
		return a.searchLike(ctx, q, kind, limit)
	}
	items := workspaceAsItems(searchResult["items"])
	taskIDs := make([]any, 0, len(items))
	nodeIDs := make([]any, 0, len(items))
	for _, item := range items {
		entityType := asString(item["entity_type"])
		entityID := asString(item["entity_id"])
		if entityType == "task" && searchTasks && len(taskIDs) < limit {
			taskIDs = append(taskIDs, entityID)
		} else if entityType == "node" && searchNodes && len(nodeIDs) < limit {
			nodeIDs = append(nodeIDs, entityID)
		}
	}
	// Batch fetch full task objects
	if len(taskIDs) > 0 {
		rows, err := a.queryContext(ctx, `SELECT * FROM tasks WHERE id IN (`+placeholders(len(taskIDs))+`) AND deleted_at IS NULL ORDER BY updated_at DESC`, taskIDs...)
		if err == nil {
			tasks, _ := scanRows(rows)
			if tasks != nil {
				result["tasks"] = tasks
			}
		}
	}
	// Batch fetch full node objects
	if len(nodeIDs) > 0 {
		rows, err := a.queryContext(ctx, `SELECT n.*, t.title AS task_title FROM nodes n JOIN tasks t ON t.id = n.task_id WHERE n.id IN (`+placeholders(len(nodeIDs))+`) AND n.deleted_at IS NULL AND t.deleted_at IS NULL ORDER BY n.updated_at DESC`, nodeIDs...)
		if err == nil {
			nodes, _ := scanRows(rows)
			if nodes != nil {
				result["nodes"] = nodes
			}
		}
	}
	return result, nil
}

// searchLike is the legacy LIKE-based search fallback when FTS5 is unavailable.
func (a *App) searchLike(ctx context.Context, q, kind string, limit int) (jsonMap, error) {
	result := jsonMap{"tasks": []jsonMap{}, "nodes": []jsonMap{}}
	like := buildLikePattern(q)
	if kind == "" || kind == "all" || kind == "tasks" {
		rows, err := a.queryContext(ctx, `SELECT * FROM tasks WHERE deleted_at IS NULL AND (title LIKE ? ESCAPE '\' OR goal LIKE ? ESCAPE '\') ORDER BY updated_at DESC LIMIT ?`, like, like, limit)
		if err != nil {
			return nil, err
		}
		items, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		result["tasks"] = items
	}
	if kind == "" || kind == "all" || kind == "nodes" {
		rows, err := a.queryContext(ctx, `SELECT n.*, t.title AS task_title FROM nodes n JOIN tasks t ON t.id = n.task_id WHERE n.deleted_at IS NULL AND t.deleted_at IS NULL AND (n.title LIKE ? ESCAPE '\' OR n.instruction LIKE ? ESCAPE '\' OR n.path LIKE ? ESCAPE '\') ORDER BY n.updated_at DESC LIMIT ?`, like, like, like, limit)
		if err != nil {
			return nil, err
		}
		items, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		result["nodes"] = items
	}
	return result, nil
}

func (a *App) listEvents(ctx context.Context, taskID, nodeID, before, after string, limit int) (jsonMap, error) {
	return a.listEventsScoped(ctx, taskID, nodeID, false, before, after, limit, eventListOptions{})
}

func (a *App) listEventsScoped(ctx context.Context, taskID, nodeID string, includeDescendants bool, before, after string, limit int, opts eventListOptions) (jsonMap, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	sortOrder := strings.ToLower(strings.TrimSpace(opts.SortOrder))
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	if strings.TrimSpace(opts.Before) != "" {
		before = strings.TrimSpace(opts.Before)
	}
	if strings.TrimSpace(opts.After) != "" {
		after = strings.TrimSpace(opts.After)
	}
	viewMode := strings.ToLower(strings.TrimSpace(opts.ViewMode))
	if viewMode == "" {
		viewMode = "detail"
	}
	query := `SELECT e.* FROM events e WHERE 1=1`
	if viewMode == "summary" {
		query = `SELECT e.id, e.task_id, e.node_id, e.type, e.message, e.created_at FROM events e WHERE 1=1`
	}
	args := []any{}
	if taskID != "" {
		query += ` AND task_id = ?`
		args = append(args, taskID)
	}
	if nodeID != "" {
		if includeDescendants {
			descendantIDs, err := a.descendantNodeIDs(ctx, taskID, nodeID)
			if err != nil {
				return nil, err
			}
			if len(descendantIDs) == 0 {
				return jsonMap{"items": []jsonMap{}, "next_before": nil, "has_more": false}, nil
			}
			query += ` AND node_id IN (` + placeholders(len(descendantIDs)) + `)`
			for _, id := range descendantIDs {
				args = append(args, id)
			}
		} else {
			query += ` AND node_id = ?`
			args = append(args, nodeID)
		}
	}
	if before != "" {
		query += ` AND created_at < ?`
		args = append(args, before)
	}
	if after != "" {
		query += ` AND created_at > ?`
		args = append(args, after)
	}
	if len(opts.Types) > 0 {
		query += ` AND type IN (` + placeholders(len(opts.Types)) + `)`
		for _, v := range opts.Types {
			args = append(args, v)
		}
	}
	if strings.TrimSpace(opts.Query) != "" {
		like := buildLikePattern(opts.Query)
		query += ` AND (type LIKE ? ESCAPE '\' OR message LIKE ? ESCAPE '\')`
		args = append(args, like, like)
	}
	if opts.Cursor != "" {
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) == 2 {
			cursorCreatedAt := parts[0]
			cursorID := parts[1]
			if sortOrder == "desc" {
				query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
			} else {
				query += ` AND (created_at > ? OR (created_at = ? AND id > ?))`
			}
			args = append(args, cursorCreatedAt, cursorCreatedAt, cursorID)
		}
	}
	query += ` ORDER BY created_at ` + strings.ToUpper(sortOrder) + `, id ` + strings.ToUpper(sortOrder) + ` LIMIT ?`
	args = append(args, limit+1)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	nextCursor := any(nil)
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = asString(last["created_at"]) + "|" + asString(last["id"])
	}
	return jsonMap{"items": items, "next_before": nextCursor, "next_cursor": nextCursor, "has_more": hasMore}, nil
}

func (a *App) descendantNodeIDs(ctx context.Context, taskID, nodeID string) ([]string, error) {
	if nodeID == "" {
		return nil, nil
	}
	selectedPath := ""
	if taskID == "" {
		if err := a.queryRowContext(ctx, `SELECT task_id, path FROM nodes WHERE id = ? AND deleted_at IS NULL`, nodeID).Scan(&taskID, &selectedPath); err != nil {
			return nil, &appError{Code: 404, Msg: fmt.Sprintf("node %s not found", nodeID)}
		}
	} else {
		if err := a.queryRowContext(ctx, `SELECT path FROM nodes WHERE id = ? AND task_id = ? AND deleted_at IS NULL`, nodeID, taskID).Scan(&selectedPath); err != nil {
			return nil, &appError{Code: 404, Msg: fmt.Sprintf("node %s not found in task %s", nodeID, taskID)}
		}
	}
	rows, err := a.queryContext(ctx, `SELECT id FROM nodes WHERE task_id = ? AND deleted_at IS NULL AND (path = ? OR path LIKE ?) ORDER BY path`, taskID, selectedPath, selectedPath+"/%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, 16)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (a *App) listArtifacts(ctx context.Context, taskID string, nodeID *string) ([]jsonMap, error) {
	result, err := a.listArtifactsWithOptions(ctx, taskID, nodeID, artifactListOptions{ViewMode: "detail", Limit: 100})
	if err != nil {
		return nil, err
	}
	return workspaceAsItems(result["items"]), nil
}

func (a *App) listArtifactsWithOptions(ctx context.Context, taskID string, nodeID *string, opts artifactListOptions) (jsonMap, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	viewMode := strings.ToLower(strings.TrimSpace(opts.ViewMode))
	if viewMode == "" {
		viewMode = "summary"
	}
	selectFields := `id, task_id, node_id, run_id, kind, title, uri, created_at`
	if viewMode == "detail" {
		selectFields = `*`
	}
	query := `SELECT ` + selectFields + ` FROM artifacts WHERE task_id = ?`
	args := []any{taskID}
	if nodeID != nil && *nodeID != "" {
		query += ` AND node_id = ?`
		args = append(args, *nodeID)
	}
	if kind := strings.TrimSpace(opts.Kind); kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if opts.Cursor != "" {
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) == 2 {
			query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
			args = append(args, parts[0], parts[0], parts[1])
		}
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, opts.Limit+1)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	hasMore := len(items) > opts.Limit
	if hasMore {
		items = items[:opts.Limit]
	}
	nextCursor := any(nil)
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = asString(last["created_at"]) + "|" + asString(last["id"])
	}
	return jsonMap{"items": items, "has_more": hasMore, "next_cursor": nextCursor}, nil
}

func (a *App) createArtifact(ctx context.Context, taskID string, body artifactCreate) (jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	if body.RunID != nil && *body.RunID != "" {
		run, err := a.findRun(ctx, *body.RunID)
		if err != nil {
			return nil, err
		}
		if asString(run["task_id"]) != taskID {
			return nil, &appError{Code: 400, Msg: "run belongs to another task"}
		}
		if body.NodeID == nil || *body.NodeID == "" {
			nodeID := asString(run["node_id"])
			body.NodeID = &nodeID
		}
	}
	if body.NodeID != nil && *body.NodeID != "" {
		node, err := a.findNode(ctx, *body.NodeID, false)
		if err != nil {
			return nil, err
		}
		if asString(node["task_id"]) != taskID {
			return nil, &appError{Code: 400, Msg: "node belongs to another task"}
		}
	}
	if body.Meta == nil {
		body.Meta = map[string]any{}
	}
	kind := ""
	if body.Kind != nil {
		kind = *body.Kind
	}
	id := newID("art")
	if _, err := a.execContext(ctx, `INSERT INTO artifacts(id, task_id, node_id, run_id, kind, title, uri, meta_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, taskID, body.NodeID, body.RunID, kind, body.Title, body.URI, mustJSON(body.Meta), utcNowISO()); err != nil {
		return nil, err
	}
	msg := body.URI
	if body.Title != nil && *body.Title != "" {
		msg = *body.Title
	}
	if err := a.insertEvent(ctx, taskID, body.NodeID, "artifact", &msg, map[string]any{"artifact_id": id, "kind": kind, "uri": body.URI, "run_id": body.RunID}, nil, nil); err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM artifacts WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (a *App) uploadArtifact(ctx context.Context, taskID string, r *http.Request) (jsonMap, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var nodeID *string
	if raw := r.FormValue("node_id"); raw != "" {
		nodeID = &raw
	}
	kind := r.FormValue("kind")
	if kind == "" {
		kind = "file"
	}
	title := header.Filename
	return a.storeArtifactBytes(ctx, taskID, nodeID, nil, kind, title, header.Filename, content, nil)
}

func (a *App) uploadArtifactBase64(ctx context.Context, taskID string, body artifactUpload) (jsonMap, error) {
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(body.ContentBase))
	if err != nil {
		return nil, &appError{Code: 400, Msg: "invalid base64 content"}
	}
	if strings.TrimSpace(body.Filename) == "" {
		return nil, &appError{Code: 400, Msg: "filename required"}
	}
	kind := "file"
	if body.Kind != nil && *body.Kind != "" {
		kind = *body.Kind
	}
	title := body.Filename
	if body.Title != nil && *body.Title != "" {
		title = *body.Title
	}
	return a.storeArtifactBytes(ctx, taskID, body.NodeID, body.RunID, kind, title, body.Filename, content, body.Meta)
}

func (a *App) storeArtifactFile(ctx context.Context, taskID string, header *multipart.FileHeader, content []byte, nodeID *string, kind string) (jsonMap, error) {
	title := header.Filename
	return a.storeArtifactBytes(ctx, taskID, nodeID, nil, kind, title, header.Filename, content, nil)
}

func (a *App) storeArtifactBytes(ctx context.Context, taskID string, nodeID, runID *string, kind, title, filename string, content []byte, extraMeta map[string]any) (jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	if nodeID != nil && *nodeID != "" {
		node, err := a.findNode(ctx, *nodeID, false)
		if err != nil {
			return nil, err
		}
		if asString(node["task_id"]) != taskID {
			return nil, &appError{Code: 400, Msg: "node belongs to another task"}
		}
	}
	id := newID("art")
	targetDir := filepathJoin(a.artifactRoot, taskID)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	name := sanitizeFilename(filename)
	target := filepathJoin(targetDir, id+"_"+name)
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return nil, err
	}
	meta := map[string]any{
		"filename": filename,
		"size":     len(content),
		"path":     target,
	}
	for key, value := range extraMeta {
		meta[key] = value
	}
	return a.createArtifact(ctx, taskID, artifactCreate{
		NodeID: nodeID,
		RunID:  runID,
		Kind:   &kind,
		Title:  &title,
		URI:    fmt.Sprintf("local://%s/%s", taskID, id),
		Meta:   meta,
	})
}

func (a *App) getArtifactFile(ctx context.Context, artifactID string) ([]byte, string, error) {
	rows, err := a.queryContext(ctx, `SELECT * FROM artifacts WHERE id = ?`, artifactID)
	if err != nil {
		return nil, "", err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, "", err
	}
	if len(items) == 0 {
		return nil, "", &appError{Code: 404, Msg: "artifact not found"}
	}
	meta, _ := items[0]["meta"].(map[string]any)
	path := asString(meta["path"])
	if path == "" {
		return nil, "", &appError{Code: 404, Msg: "artifact has no local file"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", &appError{Code: 404, Msg: "file missing on disk"}
	}
	return data, asString(meta["filename"]), nil
}

func buildLikePattern(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "%"
	}
	return "%" + escapeLike(trimmed) + "%"
}

func escapeLike(input string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(input)
}

func sanitizeFilename(name string) string {
	name = filepathBase(name)
	var out []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-", r) {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "upload.bin"
	}
	if len(out) > 120 {
		out = out[:120]
	}
	return string(out)
}

func filepathBase(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))[strings.LastIndex(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")), "/")+1:]
}

func sortJSONMapsByUpdated(items []jsonMap) {
	sort.Slice(items, func(i, j int) bool {
		return asString(items[i]["updated_at"]) > asString(items[j]["updated_at"])
	})
}

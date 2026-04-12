package tasktree

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

var evidenceWords = []string{"test", "pass", "pytest", "jest", "mocha", "diff", "commit", "sha", "benchmark", "screenshot", "artifact", "cover", "bench", "perf", "验收", "通过", "单测", "测试", "截图", "覆盖率", "性能"}

func (a *App) findNode(ctx context.Context, nodeID string, includeDeleted bool) (jsonMap, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT * FROM nodes WHERE id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("node %s not found", nodeID)}
	}
	if items[0]["deleted_at"] != nil && !includeDeleted {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("node %s deleted", nodeID)}
	}
	return items[0], nil
}

func (a *App) listNodes(ctx context.Context, taskID string) ([]jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT * FROM nodes WHERE task_id = ? AND deleted_at IS NULL ORDER BY depth, sort_order, created_at`, taskID)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func (a *App) listNodesWithOptions(ctx context.Context, taskID string, opts nodeListOptions) (jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	opts = normalizeNodeListOptions(opts)

	selectFields := `n.*, EXISTS(SELECT 1 FROM nodes c WHERE c.parent_node_id = n.id AND c.deleted_at IS NULL) AS has_children`
	if opts.ViewMode == "summary" || opts.ViewMode == "events" {
		selectFields = `n.id, n.task_id, n.parent_node_id, n.path, n.title, n.kind, n.status, n.progress, n.estimate, n.created_at, n.updated_at, EXISTS(SELECT 1 FROM nodes c WHERE c.parent_node_id = n.id AND c.deleted_at IS NULL) AS has_children`
	}
	query := `SELECT ` + selectFields + ` FROM nodes n WHERE n.task_id = ?`
	args := []any{taskID}
	if !opts.IncludeHidden {
		query += ` AND n.deleted_at IS NULL`
	}

	if len(opts.Statuses) > 0 {
		query += ` AND n.status IN (` + placeholders(len(opts.Statuses)) + `)`
		for _, v := range opts.Statuses {
			args = append(args, v)
		}
	}
	if len(opts.Kinds) > 0 {
		query += ` AND n.kind IN (` + placeholders(len(opts.Kinds)) + `)`
		for _, v := range opts.Kinds {
			args = append(args, v)
		}
	}
	if opts.Depth != nil {
		query += ` AND n.depth = ?`
		args = append(args, *opts.Depth)
	}
	if opts.MaxDepth != nil {
		query += ` AND n.depth <= ?`
		args = append(args, *opts.MaxDepth)
	}
	if opts.UpdatedAfter != "" {
		query += ` AND n.updated_at > ?`
		args = append(args, opts.UpdatedAfter)
	}
	if strings.TrimSpace(opts.Query) != "" {
		like := "%" + strings.TrimSpace(opts.Query) + "%"
		query += ` AND (n.title LIKE ? OR n.path LIKE ? OR n.instruction LIKE ?)`
		args = append(args, like, like, like)
	}

	switch opts.FilterMode {
	case "active":
		query += ` AND n.status IN ('ready','running')`
	case "blocked":
		query += ` AND n.status = 'blocked'`
	case "done":
		query += ` AND n.status IN ('done','canceled')`
	}

	if opts.HasChildren != nil {
		if *opts.HasChildren {
			query += ` AND EXISTS(SELECT 1 FROM nodes c2 WHERE c2.parent_node_id = n.id AND c2.deleted_at IS NULL)`
		} else {
			query += ` AND NOT EXISTS(SELECT 1 FROM nodes c2 WHERE c2.parent_node_id = n.id AND c2.deleted_at IS NULL)`
		}
	}

	useCompositePath := false
	sortColumn := "n.sort_order"
	switch opts.SortBy {
	case "path":
		useCompositePath = true
	case "updated_at":
		sortColumn = "n.updated_at"
	case "created_at":
		sortColumn = "n.created_at"
	case "status":
		sortColumn = "n.status"
	case "progress":
		sortColumn = "n.progress"
	}
	if opts.Cursor != "" && !useCompositePath {
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) == 2 {
			cursorValue := parts[0]
			cursorID := parts[1]
			if opts.SortOrder == "desc" {
				query += ` AND (` + sortColumn + ` < ? OR (` + sortColumn + ` = ? AND n.id < ?))`
			} else {
				query += ` AND (` + sortColumn + ` > ? OR (` + sortColumn + ` = ? AND n.id > ?))`
			}
			args = append(args, cursorValue, cursorValue, cursorID)
		}
	}

	if useCompositePath {
		query += ` ORDER BY n.depth ` + strings.ToUpper(opts.SortOrder) + `, n.sort_order ` + strings.ToUpper(opts.SortOrder) + `, n.created_at ` + strings.ToUpper(opts.SortOrder) + ` LIMIT ?`
	} else {
		query += ` ORDER BY ` + sortColumn + ` ` + strings.ToUpper(opts.SortOrder) + `, n.id ` + strings.ToUpper(opts.SortOrder) + ` LIMIT ?`
	}
	args = append(args, opts.Limit+1)

	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	if opts.FilterMode == "focus" {
		focusIDs, err := a.focusNodeIDs(ctx, taskID, opts)
		if err != nil {
			return nil, err
		}
		if len(focusIDs) == 0 {
			return jsonMap{"items": []jsonMap{}, "has_more": false, "next_cursor": nil}, nil
		}
		filtered := make([]jsonMap, 0, len(items))
		for _, item := range items {
			if _, ok := focusIDs[asString(item["id"])]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	hasMore := len(items) > opts.Limit
	if hasMore {
		items = items[:opts.Limit]
	}
	nextCursor := any(nil)
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		cursorValue := asString(last[opts.SortBy])
		if opts.SortBy == "" || opts.SortBy == "path" {
			cursorValue = asString(last["path"])
		}
		nextCursor = cursorValue + "|" + asString(last["id"])
	}

	if opts.ViewMode == "summary" {
		summaryItems := make([]jsonMap, 0, len(items))
		for _, item := range items {
			summaryItems = append(summaryItems, buildNodeSummary(item))
		}
		return jsonMap{"items": summaryItems, "has_more": hasMore, "next_cursor": nextCursor}, nil
	}

	if opts.ViewMode == "events" {
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, asString(item["id"]))
		}
		lastEvents, err := a.listLastEventsForNodes(ctx, ids)
		if err != nil {
			return nil, err
		}
		eventItems := make([]jsonMap, 0, len(items))
		for _, item := range items {
			summary := buildNodeSummary(item)
			if ev, ok := lastEvents[asString(item["id"])]; ok {
				summary["last_event"] = ev
			}
			eventItems = append(eventItems, summary)
		}
		return jsonMap{"items": eventItems, "has_more": hasMore, "next_cursor": nextCursor}, nil
	}

	for _, item := range items {
		item["has_children"] = asFloat(item["has_children"]) > 0
	}
	return jsonMap{"items": items, "has_more": hasMore, "next_cursor": nextCursor}, nil
}

func (a *App) listLastEventsForNodes(ctx context.Context, nodeIDs []string) (map[string]jsonMap, error) {
	out := map[string]jsonMap{}
	if len(nodeIDs) == 0 {
		return out, nil
	}
	query := `SELECT e.node_id, e.type, e.message, e.created_at FROM events e WHERE e.node_id IN (` + placeholders(len(nodeIDs)) + `) ORDER BY e.created_at DESC`
	args := make([]any, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		args = append(args, id)
	}
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var nodeID, eventType, createdAt string
		var message sql.NullString
		if err := rows.Scan(&nodeID, &eventType, &message, &createdAt); err != nil {
			return nil, err
		}
		if _, exists := out[nodeID]; exists {
			continue
		}
		out[nodeID] = jsonMap{
			"type":       eventType,
			"message":    message.String,
			"created_at": createdAt,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) focusNodeIDs(ctx context.Context, taskID string, opts nodeListOptions) (map[string]struct{}, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, parent_node_id, status, kind FROM nodes WHERE task_id = ? AND deleted_at IS NULL ORDER BY path`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	parentByID := map[string]string{}
	statusByID := map[string]string{}
	kindByID := map[string]string{}
	for rows.Next() {
		var id, status, kind string
		var parentID sql.NullString
		if err := rows.Scan(&id, &parentID, &status, &kind); err != nil {
			return nil, err
		}
		parentByID[id] = parentID.String
		statusByID[id] = status
		kindByID[id] = kind
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	targetStatuses := map[string]struct{}{"ready": {}, "running": {}}
	if len(opts.Statuses) > 0 {
		targetStatuses = map[string]struct{}{}
		for _, status := range opts.Statuses {
			targetStatuses[status] = struct{}{}
		}
	}
	focusIDs := map[string]struct{}{}
	for id, status := range statusByID {
		if kindByID[id] == "group" {
			continue
		}
		if _, ok := targetStatuses[status]; !ok {
			continue
		}
		focusIDs[id] = struct{}{}
		parent := parentByID[id]
		for parent != "" {
			focusIDs[parent] = struct{}{}
			parent = parentByID[parent]
		}
	}
	return focusIDs, nil
}

func buildNodeSummary(item jsonMap) jsonMap {
	hasChildren := asFloat(item["has_children"]) > 0
	status := asString(item["status"])
	nextAction := "wait"
	switch status {
	case "ready":
		nextAction = "start"
	case "running":
		nextAction = "continue"
	case "blocked":
		nextAction = "unblock"
	case "paused":
		nextAction = "reopen"
	case "done", "canceled":
		nextAction = "closed"
	}
	if hasChildren && asString(item["kind"]) == "group" {
		nextAction = "drilldown"
	}
	return jsonMap{
		"id":             item["id"],
		"task_id":        item["task_id"],
		"parent_node_id": item["parent_node_id"],
		"path":           item["path"],
		"title":          item["title"],
		"kind":           item["kind"],
		"status":         item["status"],
		"progress":       item["progress"],
		"estimate":       item["estimate"],
		"updated_at":     item["updated_at"],
		"has_children":   hasChildren,
		"next_action":    nextAction,
	}
}

func normalizeNodeListOptions(opts nodeListOptions) nodeListOptions {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Limit > 500 {
		opts.Limit = 500
	}
	switch opts.ViewMode {
	case "summary", "detail", "events":
	default:
		opts.ViewMode = "summary"
	}
	switch opts.FilterMode {
	case "", "all", "focus", "active", "blocked", "done":
	default:
		opts.FilterMode = "all"
	}
	switch opts.SortBy {
	case "", "path", "updated_at", "created_at", "status", "progress":
	default:
		opts.SortBy = "path"
	}
	if opts.SortBy == "" {
		opts.SortBy = "path"
	}
	switch strings.ToLower(opts.SortOrder) {
	case "asc", "desc":
		opts.SortOrder = strings.ToLower(opts.SortOrder)
	default:
		if opts.SortBy == "updated_at" || opts.SortBy == "created_at" {
			opts.SortOrder = "desc"
		} else {
			opts.SortOrder = "asc"
		}
	}
	return opts
}

func (a *App) createNode(ctx context.Context, taskID string, body nodeCreate) (jsonMap, error) {
	if body.Title == "" {
		return nil, &appError{Code: 400, Msg: "title required"}
	}
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	parentPath := ""
	parentDepth := -1
	parentID := ""
	parentWasLeaf := false
	parentShouldCarry := false
	parentCarryTitle := ""
	parentCarryInstruction := ""
	parentCarryAcceptance := []string{}
	parentCarryEstimate := 1.0
	if body.ParentNodeID != nil && *body.ParentNodeID != "" {
		parentID = *body.ParentNodeID
		parent, err := a.findNode(ctx, parentID, false)
		if err != nil {
			return nil, err
		}
		if asString(parent["task_id"]) != taskID {
			return nil, &appError{Code: 400, Msg: "parent node belongs to another task"}
		}
		if asString(parent["kind"]) == "linked_task" {
			return nil, &appError{Code: 400, Msg: "cannot add child to a linked_task node"}
		}
		parentPath = asString(parent["path"])
		parentDepth = int(asFloat(parent["depth"]))
		if asString(parent["kind"]) == "leaf" {
			parentWasLeaf = true
			parentCarryTitle = strings.TrimSpace(asString(parent["title"]))
			parentCarryInstruction = strings.TrimSpace(asString(parent["instruction"]))
			parentCarryAcceptance = stringSliceFromAny(parent["acceptance_criteria"])
			if est := asFloat(parent["estimate"]); est > 0 {
				parentCarryEstimate = est
			}
			parentShouldCarry = parentCarryInstruction != "" ||
				len(parentCarryAcceptance) > 0 ||
				asFloat(parent["progress"]) > 0 ||
				asString(parent["result"]) != ""
			if _, err := a.db.ExecContext(ctx, `UPDATE nodes
				SET kind = 'group',
				    status = 'ready',
				    result = '',
				    progress = 0,
				    instruction = ?,
				    acceptance_criteria_json = ?,
				    claimed_by_type = NULL,
				    claimed_by_id = NULL,
				    lease_until = NULL,
				    updated_at = ?,
				    version = version + 1
				WHERE id = ?`, "已拆解为子节点执行，请查看子节点详情。", mustJSON([]string{}), utcNowISO(), parentID); err != nil {
				return nil, err
			}
			msg := "新增子节点，父节点转为分组节点"
			if err := a.insertEvent(ctx, taskID, &parentID, "node_retyped", &msg, map[string]any{
				"from_kind": "leaf",
				"to_kind":   "group",
				"reason":    "child_added",
			}, nil, nil); err != nil {
				return nil, err
			}
		}
	}
	depth := parentDepth + 1
	nodeKey := ""
	if body.NodeKey != nil && *body.NodeKey != "" {
		nodeKey = *body.NodeKey
	}
	if nodeKey == "" {
		var count int
		if parentID != "" {
			err = a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE parent_node_id = ? AND deleted_at IS NULL`, parentID).Scan(&count)
		} else {
			err = a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE task_id = ? AND parent_node_id IS NULL AND deleted_at IS NULL`, taskID).Scan(&count)
		}
		if err != nil {
			return nil, err
		}
		nodeKey = fmt.Sprintf("%d", count+1)
	}
	path := ""
	if parentPath != "" {
		path = parentPath + "/" + nodeKey
	} else {
		taskKey := asString(task["task_key"])
		if taskKey == "" {
			taskKey = taskID
		}
		path = taskKey + "/" + nodeKey
	}
	kind := body.Kind
	if kind == "" {
		kind = "leaf"
	}
	if kind == "linked_task" {
		return nil, &appError{Code: 400, Msg: "linked_task node type is disabled"}
	}
	status := "ready"
	if body.Status != nil && *body.Status != "" {
		status = *body.Status
	}
	estimate := 1.0
	if body.Estimate != nil {
		estimate = *body.Estimate
	}
	sortOrder := 0
	if body.SortOrder != nil {
		sortOrder = *body.SortOrder
	}
	createdByType := "human"
	if body.CreatedByType != nil && *body.CreatedByType != "" {
		createdByType = *body.CreatedByType
	}
	if body.Metadata == nil {
		body.Metadata = map[string]any{}
	}
	// When a leaf parent is first split, preserve the original work in an auto-created first child.
	if parentWasLeaf && parentShouldCarry {
		carryID := newID("nd")
		carryNow := utcNowISO()
		carryPath := parentPath + "/1"
		carryTitle := "原任务内容"
		if parentCarryTitle != "" {
			carryTitle = parentCarryTitle + "（原任务）"
		}
		if _, err := a.db.ExecContext(ctx, `INSERT INTO nodes(id, task_id, parent_node_id, node_key, path, depth, kind, title, instruction, acceptance_criteria_json, status, estimate, sort_order, metadata_json, created_by_type, created_by_id, creation_reason, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'leaf', ?, ?, ?, 'ready', ?, 0, ?, ?, ?, ?, ?, ?)`,
			carryID, taskID, parentID, "1", carryPath, parentDepth+1, carryTitle, parentCarryInstruction, mustJSON(defaultCriteria(parentCarryAcceptance)), parentCarryEstimate, mustJSON(map[string]any{"generated_from_parent": true}), "system", nil, "split_from_parent", carryNow, carryNow); err != nil {
			return nil, err
		}
		carryMsg := "父节点拆分后自动承接原任务内容"
		if err := a.insertEvent(ctx, taskID, &carryID, "node_created", &carryMsg, map[string]any{
			"path":                carryPath,
			"kind":                "leaf",
			"generated_from":      parentID,
			"generated_from_path": parentPath,
		}, nil, nil); err != nil {
			return nil, err
		}
	}
	// If auto-carry child was inserted, guard against path collisions on user-provided node keys.
	if parentID != "" {
		var pathCount int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE task_id = ? AND path = ? AND deleted_at IS NULL`, taskID, path).Scan(&pathCount); err != nil {
			return nil, err
		}
		if pathCount > 0 {
			var count int
			if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE parent_node_id = ? AND deleted_at IS NULL`, parentID).Scan(&count); err != nil {
				return nil, err
			}
			nodeKey = fmt.Sprintf("%d", count+1)
			path = parentPath + "/" + nodeKey
		}
	}
	nodeID := newID("nd")
	now := utcNowISO()
	if _, err := a.db.ExecContext(ctx, `INSERT INTO nodes(id, task_id, parent_node_id, node_key, path, depth, kind, title, instruction, acceptance_criteria_json, status, estimate, sort_order, metadata_json, created_by_type, created_by_id, creation_reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nodeID, taskID, nullable(parentID), nodeKey, path, depth, kind, body.Title, body.Instruction, mustJSON(defaultCriteria(body.AcceptanceCriteria)), status, estimate, sortOrder, mustJSON(body.Metadata), createdByType, body.CreatedByID, body.CreationReason, now, now); err != nil {
		return nil, err
	}
	if err := a.insertEvent(ctx, taskID, &nodeID, "node_created", &body.Title, map[string]any{"path": path, "kind": kind}, nil, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, taskID); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) retypeNodeToLeaf(ctx context.Context, nodeID string, body retypeBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	kind := asString(node["kind"])
	switch kind {
	case "leaf":
		return node, nil
	case "linked_task":
		return nil, &appError{Code: 400, Msg: "linked_task node cannot be converted to leaf"}
	case "group":
	default:
		return nil, &appError{Code: 400, Msg: fmt.Sprintf("unsupported node kind %q", kind)}
	}
	var childCount int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE parent_node_id = ? AND deleted_at IS NULL`, nodeID).Scan(&childCount); err != nil {
		return nil, err
	}
	if childCount > 0 {
		return nil, &appError{Code: 409, Msg: "group node still has child nodes"}
	}
	taskID := asString(node["task_id"])
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes
		SET kind = 'leaf',
		    status = 'ready',
		    result = '',
		    progress = 0,
		    claimed_by_type = NULL,
		    claimed_by_id = NULL,
		    lease_until = NULL,
		    updated_at = ?,
		    version = version + 1
		WHERE id = ?`, utcNowISO(), nodeID); err != nil {
		return nil, err
	}
	msg := "空分组节点转回执行节点"
	if body.Message != nil && strings.TrimSpace(*body.Message) != "" {
		msg = strings.TrimSpace(*body.Message)
	}
	if err := a.insertEvent(ctx, taskID, &nodeID, "node_retyped", &msg, map[string]any{
		"from_kind": "group",
		"to_kind":   "leaf",
		"reason":    "manual_retype",
	}, body.Actor, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, taskID); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) reportProgress(ctx context.Context, nodeID string, body progressBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	switch asString(node["kind"]) {
	case "group":
		return nil, &appError{Code: 400, Msg: "cannot report progress on group node"}
	case "linked_task":
		return nil, &appError{Code: 400, Msg: "cannot report progress on linked_task node"}
	}
	switch asString(node["status"]) {
	case "blocked":
		return nil, &appError{Code: 409, Msg: "node is blocked, unblock first"}
	case "paused":
		return nil, &appError{Code: 409, Msg: "node is paused, reopen first"}
	case "canceled", "done":
		return nil, &appError{Code: 409, Msg: "node is closed, reopen first"}
	}
	if hasClosedResult(node) {
		return nil, &appError{Code: 409, Msg: "node is closed, reopen first"}
	}
	if body.IdempotencyKey != nil && *body.IdempotencyKey != "" {
		var count int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE idempotency_key = ?`, *body.IdempotencyKey).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return a.findNode(ctx, nodeID, false)
		}
	}
	progress := asFloat(node["progress"])
	if body.Progress != nil {
		progress = *body.Progress
	} else if body.DeltaProgress != nil {
		progress += *body.DeltaProgress
	}
	progress = math.Max(0, math.Min(1, progress))
	status := asString(node["status"])
	result := ""
	clearLease := false
	if progress > 0 && (status == "pending" || status == "ready") {
		status = "running"
	}
	if progress >= 1 {
		status = "done"
		result = "done"
		clearLease = true
	}
	query := `UPDATE nodes SET progress = ?, status = ?, result = ?, updated_at = ?, version = version + 1`
	args := []any{progress, status, result, utcNowISO()}
	if clearLease {
		query += `, claimed_by_type = NULL, claimed_by_id = NULL, lease_until = NULL`
	}
	query += ` WHERE id = ?`
	args = append(args, nodeID)
	if _, err := a.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	payload := map[string]any{"progress": progress}
	warnings := assessMessageQuality(ptr(body.Message), "progress")
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	if err := a.insertEvent(ctx, asString(node["task_id"]), &nodeID, "progress", body.Message, payload, body.Actor, body.IdempotencyKey); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, asString(node["task_id"])); err != nil {
		return nil, err
	}
	updated, err := a.findNode(ctx, nodeID, false)
	if err == nil && len(warnings) > 0 {
		updated["warnings"] = warnings
	}
	return updated, err
}

func (a *App) completeNode(ctx context.Context, nodeID string, body completeBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(node["kind"]) != "leaf" {
		return nil, &appError{Code: 400, Msg: "only leaf node can be completed directly"}
	}
	if body.IdempotencyKey != nil && *body.IdempotencyKey != "" {
		var count int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE idempotency_key = ?`, *body.IdempotencyKey).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return a.findNode(ctx, nodeID, false)
		}
	}
	if isDoneResult(node) {
		return node, nil
	}
	if isCanceledResult(node) {
		return nil, &appError{Code: 409, Msg: "node is canceled, reopen first"}
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes
		SET progress = 1,
		    status = 'done',
		    result = 'done',
		    claimed_by_type = NULL,
		    claimed_by_id = NULL,
		    lease_until = NULL,
		    updated_at = ?,
		    version = version + 1
		WHERE id = ?`, utcNowISO(), nodeID); err != nil {
		return nil, err
	}
	payload := map[string]any{}
	warnings := assessMessageQuality(ptr(body.Message), "complete")
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	if err := a.insertEvent(ctx, asString(node["task_id"]), &nodeID, "complete", body.Message, payload, body.Actor, body.IdempotencyKey); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, asString(node["task_id"])); err != nil {
		return nil, err
	}
	updated, err := a.findNode(ctx, nodeID, false)
	if err == nil && len(warnings) > 0 {
		updated["warnings"] = warnings
	}
	return updated, err
}

func (a *App) blockNode(ctx context.Context, nodeID string, body blockBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(node["kind"]) != "leaf" {
		return nil, &appError{Code: 400, Msg: "only leaf node can be blocked directly"}
	}
	if hasClosedResult(node) {
		return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s", asString(node["status"]))}
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes
		SET status = 'blocked',
		    result = '',
		    claimed_by_type = NULL,
		    claimed_by_id = NULL,
		    lease_until = NULL,
		    updated_at = ?,
		    version = version + 1
		WHERE id = ?`, utcNowISO(), nodeID); err != nil {
		return nil, err
	}
	if err := a.insertEvent(ctx, asString(node["task_id"]), &nodeID, "blocked", &body.Reason, nil, body.Actor, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, asString(node["task_id"])); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) claimNode(ctx context.Context, nodeID string, body claimBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(node["kind"]) == "group" {
		return nil, &appError{Code: 400, Msg: "cannot claim group node"}
	}
	if hasClosedResult(node) {
		return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s", asString(node["status"]))}
	}
	if asString(node["status"]) == "blocked" {
		return nil, &appError{Code: 409, Msg: "node is blocked, unblock first"}
	}
	if asString(node["status"]) == "paused" {
		return nil, &appError{Code: 409, Msg: "node is paused, reopen first"}
	}
	if leaseActive(node) && asString(node["claimed_by_id"]) != strPtrValue(body.Actor.AgentID) {
		return nil, &appError{Code: 409, Msg: fmt.Sprintf("already claimed by %s/%s until %s", asString(node["claimed_by_type"]), asString(node["claimed_by_id"]), asString(node["lease_until"]))}
	}
	leaseSeconds := 900
	if body.LeaseSeconds != nil {
		leaseSeconds = *body.LeaseSeconds
	}
	leaseUntil := time.Now().UTC().Add(time.Duration(leaseSeconds) * time.Second).Format("2006-01-02T15:04:05.000000Z")
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET claimed_by_type = ?, claimed_by_id = ?, lease_until = ?, status = CASE WHEN status IN ('pending','ready') THEN 'running' ELSE status END, result = '', updated_at = ?, version = version + 1 WHERE id = ?`,
		body.Actor.Tool, body.Actor.AgentID, leaseUntil, utcNowISO(), nodeID); err != nil {
		return nil, err
	}
	if err := a.insertEvent(ctx, asString(node["task_id"]), &nodeID, "claim", nil, map[string]any{"lease_until": leaseUntil}, &body.Actor, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, asString(node["task_id"])); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) releaseNode(ctx context.Context, nodeID string) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	status := asString(node["status"])
	if status == "running" && asFloat(node["progress"]) == 0 {
		status = "ready"
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET claimed_by_type = NULL, claimed_by_id = NULL, lease_until = NULL, status = ?, updated_at = ?, version = version + 1 WHERE id = ?`, status, utcNowISO(), nodeID); err != nil {
		return nil, err
	}
	payload := map[string]any{"status": status}
	if err := a.insertEvent(ctx, asString(node["task_id"]), &nodeID, "release", nil, payload, nil, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, asString(node["task_id"])); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) sweepExpiredLeases(ctx context.Context) (int64, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT DISTINCT task_id FROM nodes WHERE lease_until IS NOT NULL AND lease_until < ?`, utcNowISO())
	if err != nil {
		return 0, err
	}
	taskItems, err := scanRows(rows)
	if err != nil {
		return 0, err
	}
	res, err := a.db.ExecContext(ctx, `UPDATE nodes
		SET claimed_by_type = NULL,
		    claimed_by_id = NULL,
		    lease_until = NULL,
		    status = CASE WHEN status = 'running' AND progress <= 0 THEN 'ready' ELSE status END,
		    updated_at = ?,
		    version = version + 1
		WHERE lease_until IS NOT NULL AND lease_until < ?`, utcNowISO(), utcNowISO())
	if err != nil {
		return 0, err
	}
	for _, item := range taskItems {
		if err := a.rollupTask(ctx, asString(item["task_id"])); err != nil {
			return 0, err
		}
	}
	return res.RowsAffected()
}

func (a *App) rollupTask(ctx context.Context, taskID string) error {
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		return err
	}
	groupChildren := map[string][]jsonMap{}
	for _, node := range nodes {
		parent := asString(node["parent_node_id"])
		groupChildren[parent] = append(groupChildren[parent], node)
	}
	sortNodesByDepthDesc(nodes)
	for _, node := range nodes {
		if asString(node["kind"]) != "group" {
			continue
		}
		children := groupChildren[asString(node["id"])]
		if len(children) == 0 {
			continue
		}
		sum := 0.0
		for _, child := range children {
			sum += asFloat(child["progress"])
		}
		progress := sum / float64(len(children))
		status, result := aggregateRollupState(children)
		node["progress"] = progress
		node["status"] = status
		node["result"] = result
		if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET progress = ?, status = ?, result = ?, updated_at = ?, version = version + 1 WHERE id = ?`, progress, status, result, utcNowISO(), node["id"]); err != nil {
			return err
		}
	}
	nodes, err = a.listNodes(ctx, taskID)
	if err != nil {
		return err
	}
	summary := summarizeLeafNodes(nodes)
	percent := 0.0
	status := "ready"
	result := ""
	if summary.Total > 0 {
		percent = 100 * summary.ProgressSum / float64(summary.Total)
	}
	if summary.Total == 0 {
		status = "ready"
	} else {
		status, result = aggregateRollupState(summary.LeafNodes)
	}
	_, err = a.db.ExecContext(ctx, `UPDATE tasks SET summary_percent = ?, summary_done = ?, summary_total = ?, summary_blocked = ?, status = ?, result = ?, updated_at = ? WHERE id = ?`,
		percent, summary.DoneCount, summary.Total, summary.BlockedCount, status, result, utcNowISO(), taskID)
	return err
}

type leafNodeSummary struct {
	LeafNodes         []jsonMap
	Total             int
	DoneCount         int
	BlockedCount      int
	PausedCount       int
	CanceledCount     int
	ProgressSum       float64
	RemainingCount    int
	RemainingEstimate float64
	NextReady         []jsonMap
}

func summarizeLeafNodes(nodes []jsonMap) leafNodeSummary {
	summary := leafNodeSummary{
		LeafNodes: make([]jsonMap, 0, len(nodes)),
		NextReady: []jsonMap{},
	}
	orderedLeaves := orderedExecutableLeaves(nodes)
	for _, node := range orderedLeaves {
		if asString(node["kind"]) == "group" {
			continue
		}
		summary.LeafNodes = append(summary.LeafNodes, node)
		summary.Total++
		summary.ProgressSum += asFloat(node["progress"])
		status := asString(node["status"])
		if isDoneResult(node) {
			summary.DoneCount++
		} else {
			if !isCanceledResult(node) {
				summary.RemainingCount++
				summary.RemainingEstimate += asFloat(node["estimate"])
			}
		}
		switch status {
		case "blocked":
			summary.BlockedCount++
		case "paused":
			summary.PausedCount++
		case "canceled":
			summary.CanceledCount++
		case "ready":
			if len(summary.NextReady) < 5 {
				summary.NextReady = append(summary.NextReady, jsonMap{
					"node_id": node["id"],
					"path":    node["path"],
					"title":   node["title"],
				})
			}
		}
	}
	return summary
}

func (a *App) insertEvent(ctx context.Context, taskID string, nodeID *string, kind string, message *string, payload map[string]any, act *actor, idem *string) error {
	if idem != nil && *idem != "" {
		var count int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE idempotency_key = ?`, *idem).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	var actorTool, actorID *string
	if act != nil {
		actorTool = act.Tool
		actorID = act.AgentID
	}
	_, err := a.db.ExecContext(ctx, `INSERT INTO events(id, task_id, node_id, type, message, payload_json, actor_type, actor_id, idempotency_key, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newID("evt"), taskID, nodeID, kind, message, mustJSON(defaultPayload(payload)), actorTool, actorID, idem, utcNowISO())
	return err
}

func assessMessageQuality(msg *string, kind string) []string {
	if msg == nil || strings.TrimSpace(*msg) == "" {
		return []string{"empty"}
	}
	m := strings.TrimSpace(*msg)
	warnings := []string{}
	if len([]rune(m)) < 20 {
		warnings = append(warnings, "too_short")
	}
	if isASCII(m) && !strings.Contains(m, " ") {
		warnings = append(warnings, "not_a_sentence")
	}
	if kind == "complete" {
		found := false
		lower := strings.ToLower(m)
		for _, word := range evidenceWords {
			if strings.Contains(lower, strings.ToLower(word)) {
				found = true
				break
			}
		}
		if !found {
			warnings = append(warnings, "no_evidence")
		}
		if !strings.Contains(m, "\n") && len([]rune(m)) < 60 {
			warnings = append(warnings, "missing_structure")
		}
	}
	return warnings
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func leaseActive(node jsonMap) bool {
	leaseUntil := asString(node["lease_until"])
	if leaseUntil == "" || asString(node["claimed_by_id"]) == "" {
		return false
	}
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", leaseUntil)
	if err != nil {
		return false
	}
	return t.After(time.Now().UTC())
}

func defaultCriteria(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func stringSliceFromAny(v any) []string {
	switch items := v.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			s := strings.TrimSpace(asString(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return []string{}
	}
}

func defaultPayload(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func strPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func ptr(v *string) *string { return v }

func sortNodesByDepthDesc(nodes []jsonMap) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if asFloat(nodes[i]["depth"]) < asFloat(nodes[j]["depth"]) {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

func aggregateRollupStatus(nodes []jsonMap) string {
	if len(nodes) == 0 {
		return "ready"
	}
	closedCount := 0
	blockedCount := 0
	pausedCount := 0
	runningCount := 0
	readyCount := 0
	closedItems := make([]jsonMap, 0, len(nodes))
	for _, node := range nodes {
		if hasClosedResult(node) {
			closedCount++
			closedItems = append(closedItems, node)
			continue
		}
		switch asString(node["status"]) {
		case "blocked":
			blockedCount++
		case "paused":
			pausedCount++
		case "running":
			runningCount++
		case "ready":
			readyCount++
		}
	}
	switch {
	case closedCount == len(nodes):
		return statusFromResult(rollupClosedResult(closedItems))
	case blockedCount > 0 && blockedCount+closedCount == len(nodes):
		return "blocked"
	case pausedCount > 0 && pausedCount+closedCount == len(nodes):
		return "paused"
	case runningCount > 0:
		return "running"
	case readyCount > 0:
		return "ready"
	case blockedCount > 0:
		return "blocked"
	case pausedCount > 0:
		return "paused"
	default:
		return "ready"
	}
}

func aggregateRollupState(nodes []jsonMap) (string, string) {
	if len(nodes) == 0 {
		return "ready", ""
	}
	result := ""
	if status := aggregateRollupStatus(nodes); status == "done" || status == "canceled" || status == "closed" {
		closedItems := make([]jsonMap, 0, len(nodes))
		for _, node := range nodes {
			if hasClosedResult(node) {
				closedItems = append(closedItems, node)
			}
		}
		result = rollupClosedResult(closedItems)
		return status, result
	}
	return aggregateRollupStatus(nodes), result
}


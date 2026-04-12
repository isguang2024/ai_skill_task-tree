package tasktree

import (
	"context"
	"fmt"
	"strings"
)

var nodeStatuses = map[string]bool{
	"ready":    true,
	"running":  true,
	"blocked":  true,
	"paused":   true,
	"done":     true,
	"closed":   true,
	"canceled": true,
}

func (a *App) updateTask(ctx context.Context, taskID string, body taskUpdate) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	updates := []string{}
	args := []any{}
	fields := []string{}
	if body.TaskKey != nil {
		updates = append(updates, "task_key = ?")
		args = append(args, strings.TrimSpace(*body.TaskKey))
		fields = append(fields, "task_key")
	}
	if body.Title != nil {
		title := strings.TrimSpace(*body.Title)
		if title == "" {
			return nil, &appError{Code: 400, Msg: "title 不能为空"}
		}
		updates = append(updates, "title = ?")
		args = append(args, title)
		fields = append(fields, "title")
	}
	if body.Goal != nil {
		updates = append(updates, "goal = ?")
		args = append(args, strings.TrimSpace(*body.Goal))
		fields = append(fields, "goal")
	}
	if body.ProjectID != nil {
		projectID := strings.TrimSpace(*body.ProjectID)
		if projectID == "" {
			return nil, &appError{Code: 400, Msg: "project_id 不能为空"}
		}
		if _, err := a.getProject(ctx, projectID, false); err != nil {
			return nil, err
		}
		updates = append(updates, "project_id = ?")
		args = append(args, projectID)
		fields = append(fields, "project_id")
	}
	if len(updates) == 0 {
		return task, nil
	}
	updates = append(updates, "updated_at = ?", "version = version + 1")
	args = append(args, utcNowISO(), taskID)
	if _, err := a.db.ExecContext(ctx, `UPDATE tasks SET `+strings.Join(updates, ", ")+` WHERE id = ?`, args...); err != nil {
		return nil, err
	}
	msg := "更新任务信息"
	if body.Title != nil {
		msg = strings.TrimSpace(*body.Title)
	}
	if err := a.insertEvent(ctx, taskID, nil, "task_updated", &msg, map[string]any{"fields": fields}, nil, nil); err != nil {
		return nil, err
	}
	return a.getTask(ctx, taskID, false)
}

func (a *App) updateNode(ctx context.Context, nodeID string, body nodeUpdate) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	updates := []string{}
	args := []any{}
	fields := []string{}
	if body.Title != nil {
		title := strings.TrimSpace(*body.Title)
		if title == "" {
			return nil, &appError{Code: 400, Msg: "title 不能为空"}
		}
		updates = append(updates, "title = ?")
		args = append(args, title)
		fields = append(fields, "title")
	}
	if body.Instruction != nil {
		updates = append(updates, "instruction = ?")
		args = append(args, strings.TrimSpace(*body.Instruction))
		fields = append(fields, "instruction")
	}
	if body.AcceptanceCriteria != nil {
		updates = append(updates, "acceptance_criteria_json = ?")
		args = append(args, mustJSON(defaultCriteria(*body.AcceptanceCriteria)))
		fields = append(fields, "acceptance_criteria")
	}
	if body.Estimate != nil {
		if *body.Estimate < 0 {
			return nil, &appError{Code: 400, Msg: "estimate 不能小于 0"}
		}
		updates = append(updates, "estimate = ?")
		args = append(args, *body.Estimate)
		fields = append(fields, "estimate")
	}
	if body.SortOrder != nil {
		updates = append(updates, "sort_order = ?")
		args = append(args, *body.SortOrder)
		fields = append(fields, "sort_order")
	}
	if len(updates) == 0 {
		return node, nil
	}
	updates = append(updates, "updated_at = ?", "version = version + 1")
	args = append(args, utcNowISO(), nodeID)
	if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET `+strings.Join(updates, ", ")+` WHERE id = ?`, args...); err != nil {
		return nil, err
	}
	msg := "更新节点信息"
	if body.Title != nil {
		msg = strings.TrimSpace(*body.Title)
	}
	taskID := asString(node["task_id"])
	if err := a.insertEvent(ctx, taskID, &nodeID, "node_updated", &msg, map[string]any{"fields": fields}, nil, nil); err != nil {
		return nil, err
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, utcNowISO(), taskID); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

// reorderNodes sets sort_order for sibling nodes according to the given ID order.
// All node_ids must share the same parent_node_id.
func (a *App) reorderNodes(ctx context.Context, nodeIDs []string) ([]jsonMap, error) {
	if len(nodeIDs) == 0 {
		return nil, &appError{Code: 400, Msg: "node_ids 不能为空"}
	}
	// Load all nodes to validate they share the same parent
	var parentID *string
	var taskID string
	for i, nid := range nodeIDs {
		node, err := a.findNode(ctx, nid, false)
		if err != nil {
			return nil, err
		}
		pid := asString(node["parent_node_id"])
		tid := asString(node["task_id"])
		if i == 0 {
			parentID = &pid
			taskID = tid
		} else {
			if pid != *parentID {
				return nil, &appError{Code: 400, Msg: "所有节点必须是同级节点（相同 parent_node_id）"}
			}
			if tid != taskID {
				return nil, &appError{Code: 400, Msg: "所有节点必须属于同一任务"}
			}
		}
	}
	now := utcNowISO()
	for i, nid := range nodeIDs {
		if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET sort_order = ?, updated_at = ? WHERE id = ?`, i, now, nid); err != nil {
			return nil, err
		}
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, now, taskID); err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("重排 %d 个同级节点", len(nodeIDs))
	if err := a.insertEvent(ctx, taskID, nil, "nodes_reordered", &msg, map[string]any{"node_ids": nodeIDs}, nil, nil); err != nil {
		return nil, err
	}
	// Return updated siblings in new order
	result := make([]jsonMap, 0, len(nodeIDs))
	for _, nid := range nodeIDs {
		node, _ := a.findNode(ctx, nid, false)
		if node != nil {
			result = append(result, node)
		}
	}
	return result, nil
}

// moveNode moves a node to a new position among its siblings.
// Specify after_node_id to place it after that sibling, or before_node_id to place it before.
// If neither is specified, the node is moved to the beginning.
func (a *App) moveNode(ctx context.Context, nodeID string, body moveNodeBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	taskID := asString(node["task_id"])
	parentID := asString(node["parent_node_id"])

	// Get all siblings (including self), ordered by current sort_order
	rows, err := a.db.QueryContext(ctx, `SELECT id FROM nodes WHERE task_id = ? AND COALESCE(parent_node_id, '') = ? AND deleted_at IS NULL ORDER BY sort_order, created_at`, taskID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var siblingIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		siblingIDs = append(siblingIDs, id)
	}

	// Remove self from list
	filtered := make([]string, 0, len(siblingIDs))
	for _, id := range siblingIDs {
		if id != nodeID {
			filtered = append(filtered, id)
		}
	}

	// Determine insertion index
	insertIdx := 0 // default: move to beginning
	if body.AfterNodeID != nil {
		found := false
		for i, id := range filtered {
			if id == *body.AfterNodeID {
				insertIdx = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, &appError{Code: 400, Msg: "after_node_id 不在同级节点中"}
		}
	} else if body.BeforeNodeID != nil {
		found := false
		for i, id := range filtered {
			if id == *body.BeforeNodeID {
				insertIdx = i
				found = true
				break
			}
		}
		if !found {
			return nil, &appError{Code: 400, Msg: "before_node_id 不在同级节点中"}
		}
	}

	// Insert self at the target position
	newOrder := make([]string, 0, len(filtered)+1)
	newOrder = append(newOrder, filtered[:insertIdx]...)
	newOrder = append(newOrder, nodeID)
	newOrder = append(newOrder, filtered[insertIdx:]...)

	// Apply new sort_order
	now := utcNowISO()
	for i, id := range newOrder {
		if _, err := a.db.ExecContext(ctx, `UPDATE nodes SET sort_order = ?, updated_at = ? WHERE id = ?`, i, now, id); err != nil {
			return nil, err
		}
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE tasks SET updated_at = ? WHERE id = ?`, now, taskID); err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("移动节点到位置 %d", insertIdx)
	if err := a.insertEvent(ctx, taskID, &nodeID, "node_moved", &msg, map[string]any{"new_position": insertIdx}, nil, nil); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func (a *App) transitionTask(ctx context.Context, taskID string, body transitionBody) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	action := strings.TrimSpace(body.Action)
	if action == "" {
		return nil, &appError{Code: 400, Msg: "action required"}
	}
	var (
		query     string
		args      []any
		eventType string
		eventMsg  string
	)
	switch action {
	case "pause":
		query = `UPDATE nodes
			SET status = 'paused',
			    result = '',
			    claimed_by_type = NULL,
			    claimed_by_id = NULL,
			    lease_until = NULL,
			    updated_at = ?,
			    version = version + 1
			WHERE task_id = ? AND deleted_at IS NULL AND kind = 'leaf' AND COALESCE(result, '') = '' AND status IN ('ready', 'running', 'blocked')`
		args = []any{utcNowISO(), taskID}
		eventType = "task_paused"
		eventMsg = defaultTransitionMessage(body.Message, "任务已暂停")
	case "reopen":
		switch asString(task["status"]) {
		case "paused", "canceled", "closed":
		default:
			return nil, &appError{Code: 409, Msg: fmt.Sprintf("task status is %s, cannot reopen", asString(task["status"]))}
		}
		query = `UPDATE nodes
			SET status = 'ready',
			    result = '',
			    progress = CASE
			      WHEN COALESCE(result, '') = 'done' THEN 0
			      WHEN COALESCE(result, '') = 'canceled' AND progress >= 1 THEN 0
			      ELSE progress
			    END,
			    claimed_by_type = NULL,
			    claimed_by_id = NULL,
			    lease_until = NULL,
			    updated_at = ?,
			    version = version + 1
			WHERE task_id = ? AND deleted_at IS NULL AND kind = 'leaf' AND (status = 'paused' OR COALESCE(result, '') = 'canceled')`
		args = []any{utcNowISO(), taskID}
		eventType = "task_reopened"
		eventMsg = defaultTransitionMessage(body.Message, "任务已恢复")
	case "cancel":
		query = `UPDATE nodes
			SET status = 'canceled',
			    result = 'canceled',
			    claimed_by_type = NULL,
			    claimed_by_id = NULL,
			    lease_until = NULL,
			    updated_at = ?,
			    version = version + 1
			WHERE task_id = ? AND deleted_at IS NULL AND kind = 'leaf' AND COALESCE(result, '') = '' AND status IN ('ready', 'running', 'blocked', 'paused')`
		args = []any{utcNowISO(), taskID}
		eventType = "task_canceled"
		eventMsg = defaultTransitionMessage(body.Message, "任务已取消")
	default:
		return nil, &appError{Code: 400, Msg: fmt.Sprintf("unsupported task action: %s", action)}
	}
	res, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if err := a.insertEvent(ctx, taskID, nil, eventType, &eventMsg, map[string]any{"action": action, "affected_nodes": affected}, body.Actor, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, taskID); err != nil {
		return nil, err
	}
	return a.getTask(ctx, taskID, false)
}

func (a *App) transitionNode(ctx context.Context, nodeID string, body transitionBody) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(node["kind"]) != "leaf" {
		return nil, &appError{Code: 400, Msg: "only leaf node supports transition"}
	}
	action := strings.TrimSpace(body.Action)
	if action == "" {
		return nil, &appError{Code: 400, Msg: "action required"}
	}
	currentStatus := asString(node["status"])
	newStatus := currentStatus
	progress := asFloat(node["progress"])
	currentResult := itemResult(node)
	newResult := currentResult
	clearLease := false
	switch action {
	case "block":
		if hasClosedResult(node) {
			return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s", currentStatus)}
		}
		if currentStatus == "blocked" {
			return node, nil
		}
		newStatus = "blocked"
		newResult = ""
		clearLease = true
	case "pause":
		if currentStatus == "paused" {
			return node, nil
		}
		if hasClosedResult(node) {
			return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s", currentStatus)}
		}
		newStatus = "paused"
		newResult = ""
		clearLease = true
	case "cancel":
		if currentStatus == "canceled" {
			return node, nil
		}
		if currentStatus == "done" || currentStatus == "closed" {
			return nil, &appError{Code: 409, Msg: "done 节点请使用重开"}
		}
		newStatus = "canceled"
		newResult = "canceled"
		clearLease = true
	case "reopen":
		switch currentStatus {
		case "paused":
			newStatus = "ready"
			newResult = ""
		case "done":
			newStatus = "ready"
			progress = 0
			newResult = ""
		case "canceled":
			newStatus = "ready"
			if shouldResetProgressOnReopen(node) {
				progress = 0
			}
			newResult = ""
		default:
			return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s, cannot reopen", currentStatus)}
		}
		clearLease = true
	case "unblock":
		if currentStatus != "blocked" {
			return nil, &appError{Code: 409, Msg: fmt.Sprintf("node status is %s, cannot unblock", currentStatus)}
		}
		newStatus = "ready"
		newResult = ""
		clearLease = true
	default:
		return nil, &appError{Code: 400, Msg: fmt.Sprintf("unsupported node action: %s", action)}
	}
	query := `UPDATE nodes SET status = ?, result = ?, progress = ?, updated_at = ?, version = version + 1`
	args := []any{newStatus, newResult, progress, utcNowISO()}
	if clearLease {
		query += `, claimed_by_type = NULL, claimed_by_id = NULL, lease_until = NULL`
	}
	query += ` WHERE id = ?`
	args = append(args, nodeID)
	if _, err := a.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	taskID := asString(node["task_id"])
	eventType := "node_transition"
	switch action {
	case "block":
		eventType = "blocked"
	case "pause":
		eventType = "paused"
	case "cancel":
		eventType = "canceled"
	case "reopen":
		eventType = "reopened"
	case "unblock":
		eventType = "unblocked"
	}
	msg := defaultTransitionMessage(body.Message, defaultNodeTransitionMessage(action))
	if err := a.insertEvent(ctx, taskID, &nodeID, eventType, &msg, map[string]any{
		"action":      action,
		"from_status": currentStatus,
		"to_status":   newStatus,
		"from_result": currentResult,
		"to_result":   newResult,
	}, body.Actor, nil); err != nil {
		return nil, err
	}
	if err := a.rollupTask(ctx, taskID); err != nil {
		return nil, err
	}
	return a.findNode(ctx, nodeID, false)
}

func defaultTransitionMessage(msg *string, fallback string) string {
	if msg == nil || strings.TrimSpace(*msg) == "" {
		return fallback
	}
	return strings.TrimSpace(*msg)
}

func defaultNodeTransitionMessage(action string) string {
	switch action {
	case "pause":
		return "节点已暂停"
	case "cancel":
		return "节点已取消"
	case "reopen":
		return "节点已重开"
	case "unblock":
		return "节点已解除阻塞"
	default:
		return "节点状态已更新"
	}
}


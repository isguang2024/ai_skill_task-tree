package tasktree

import (
	"context"
	"fmt"
	"strings"
)

type runStart struct {
	Actor            *actor
	TriggerKind      *string
	InputSummary     *string
	OutputPreview    *string
	OutputRef        *string
	StructuredResult map[string]any
}

type runFinish struct {
	Result           *string
	Status           *string
	OutputPreview    *string
	OutputRef        *string
	StructuredResult map[string]any
	ErrorText        *string
}

type runLogCreate struct {
	Kind    string
	Content *string
	Payload map[string]any
}

func (a *App) startRun(ctx context.Context, nodeID string, body runStart) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		node, err := a.findNode(txCtx, nodeID, false)
		if err != nil {
			return err
		}
		if asString(node["kind"]) != "leaf" {
			return &appError{Code: 409, Msg: "only leaf node can start run"}
		}
		if activeRunID := strings.TrimSpace(asString(node["active_run_id"])); activeRunID != "" {
			activeRun, err := a.findRun(txCtx, activeRunID)
			if err == nil && asString(activeRun["status"]) == "running" {
				return &appError{Code: 409, Msg: fmt.Sprintf("node already has active run %s", activeRunID)}
			}
		}
		status := "running"
		triggerKind := "manual"
		if body.TriggerKind != nil && strings.TrimSpace(*body.TriggerKind) != "" {
			triggerKind = strings.TrimSpace(*body.TriggerKind)
		}
		now := utcNowISO()
		runID := newID("run")
		var actorType, actorID *string
		if body.Actor != nil {
			actorType = body.Actor.Tool
			actorID = body.Actor.AgentID
		}
		if body.StructuredResult == nil {
			body.StructuredResult = map[string]any{}
		}
		if _, err := a.execContext(txCtx, `INSERT INTO node_runs(
			id, task_id, node_id, stage_node_id, actor_type, actor_id, trigger_kind, status,
			input_summary, output_preview, output_ref, structured_result_json,
			started_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID,
			asString(node["task_id"]),
			nodeID,
			nullable(asString(node["stage_node_id"])),
			actorType,
			actorID,
			triggerKind,
			status,
			body.InputSummary,
			body.OutputPreview,
			body.OutputRef,
			mustJSON(body.StructuredResult),
			now,
			now,
			now,
		); err != nil {
			return err
		}
		nodeStatus := asString(node["status"])
		if nodeStatus == "" || nodeStatus == "ready" || nodeStatus == "paused" || nodeStatus == "blocked" {
			nodeStatus = "running"
		}
		if _, err := a.execContext(txCtx, `UPDATE nodes
			SET active_run_id = ?, last_run_at = ?, run_count = run_count + 1,
			    status = ?, updated_at = ?, version = version + 1
			WHERE id = ?`,
			runID, now, nodeStatus, now, nodeID,
		); err != nil {
			return err
		}
		if err := a.insertEvent(txCtx, asString(node["task_id"]), &nodeID, "run_started", body.InputSummary, map[string]any{
			"run_id":       runID,
			"trigger_kind": triggerKind,
		}, body.Actor, nil); err != nil {
			return err
		}
		if err := a.rollupTask(txCtx, asString(node["task_id"])); err != nil {
			return err
		}
		out, err = a.getRun(txCtx, runID)
		return err
	})
	return out, err
}

func (a *App) finishRun(ctx context.Context, runID string, body runFinish) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		run, err := a.findRun(txCtx, runID)
		if err != nil {
			return err
		}
		if asString(run["status"]) != "running" {
			return &appError{Code: 409, Msg: fmt.Sprintf("run status is %s", asString(run["status"]))}
		}
		status := "finished"
		if body.Status != nil && strings.TrimSpace(*body.Status) != "" {
			status = strings.TrimSpace(*body.Status)
		}
		result := ""
		if body.Result != nil && strings.TrimSpace(*body.Result) != "" {
			result = strings.TrimSpace(*body.Result)
		}
		if body.StructuredResult == nil {
			body.StructuredResult = asAnyMap(run["structured_result"])
			if body.StructuredResult == nil {
				body.StructuredResult = map[string]any{}
			}
		}
		now := utcNowISO()
		if _, err := a.execContext(txCtx, `UPDATE node_runs
			SET status = ?, result = ?, output_preview = COALESCE(?, output_preview),
			    output_ref = COALESCE(?, output_ref), structured_result_json = ?, error_text = ?,
			    finished_at = ?, updated_at = ?
			WHERE id = ?`,
			status, nullable(result), body.OutputPreview, body.OutputRef, mustJSON(body.StructuredResult), body.ErrorText, now, now, runID,
		); err != nil {
			return err
		}
		nodeID := asString(run["node_id"])
		node, err := a.findNode(txCtx, nodeID, false)
		if err != nil {
			return err
		}
		nodeStatus := asString(node["status"])
		nodeResult := asString(node["result"])
		switch result {
		case "done":
			nodeStatus = "done"
			nodeResult = "done"
		case "canceled":
			nodeStatus = "canceled"
			nodeResult = "canceled"
		default:
			if nodeStatus == "running" {
				nodeStatus = "ready"
			}
			if nodeResult != "done" && nodeResult != "canceled" {
				nodeResult = ""
			}
		}
		if _, err := a.execContext(txCtx, `UPDATE nodes
			SET active_run_id = CASE WHEN active_run_id = ? THEN NULL ELSE active_run_id END,
			    status = ?, result = ?, progress = CASE WHEN ? = 'done' THEN 1 ELSE progress END,
			    claimed_by_type = CASE WHEN active_run_id = ? THEN NULL ELSE claimed_by_type END,
			    claimed_by_id = CASE WHEN active_run_id = ? THEN NULL ELSE claimed_by_id END,
			    lease_until = CASE WHEN active_run_id = ? THEN NULL ELSE lease_until END,
			    updated_at = ?, version = version + 1
			WHERE id = ?`,
			runID, nodeStatus, nodeResult, result, runID, runID, runID, now, nodeID,
		); err != nil {
			return err
		}
		message := body.OutputPreview
		if message == nil || strings.TrimSpace(*message) == "" {
			message = body.ErrorText
		}
		if err := a.insertEvent(txCtx, asString(run["task_id"]), &nodeID, "run_finished", message, map[string]any{
			"run_id": runID,
			"status": status,
			"result": result,
		}, nil, nil); err != nil {
			return err
		}
		if err := a.refreshMemoryChainForNode(txCtx, nodeID); err != nil {
			return err
		}
		if err := a.rollupTask(txCtx, asString(run["task_id"])); err != nil {
			return err
		}
		stageNodeID := asString(node["stage_node_id"])
		if stageNodeID != "" {
			stage, err := a.findNode(txCtx, stageNodeID, false)
			if err == nil {
				if err := a.maybeSnapshotForNode(txCtx, stage); err != nil {
					return err
				}
			}
		}
		task, err := a.getTask(txCtx, asString(run["task_id"]), false)
		if err != nil {
			return err
		}
		if err := a.maybeSnapshotForTask(txCtx, task); err != nil {
			return err
		}
		out, err = a.getRun(txCtx, runID)
		return err
	})
	return out, err
}

func (a *App) addRunLog(ctx context.Context, runID string, body runLogCreate) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		run, err := a.findRun(txCtx, runID)
		if err != nil {
			return err
		}
		kind := strings.TrimSpace(body.Kind)
		if kind == "" {
			return &appError{Code: 400, Msg: "kind required"}
		}
		if body.Payload == nil {
			body.Payload = map[string]any{}
		}
		var seq int
		if err := a.queryRowContext(txCtx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM run_logs WHERE run_id = ?`, runID).Scan(&seq); err != nil {
			return err
		}
		now := utcNowISO()
		logID := newID("rlog")
		if _, err := a.execContext(txCtx, `INSERT INTO run_logs(id, run_id, seq, kind, content, payload_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			logID, runID, seq, kind, body.Content, mustJSON(body.Payload), now,
		); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE node_runs SET updated_at = ? WHERE id = ?`, now, runID); err != nil {
			return err
		}
		out = jsonMap{
			"id":         logID,
			"log_id":     logID,
			"run_id":     runID,
			"seq":        seq,
			"kind":       kind,
			"content":    nullableString(body.Content),
			"payload":    body.Payload,
			"created_at": now,
			"task_id":    asString(run["task_id"]),
			"node_id":    asString(run["node_id"]),
		}
		return nil
	})
	return out, err
}

func (a *App) getRun(ctx context.Context, runID string) (jsonMap, error) {
	return a.getRunWithOptions(ctx, runID, runListOptions{IncludeLogs: true})
}

func (a *App) getRunWithOptions(ctx context.Context, runID string, opts runListOptions) (jsonMap, error) {
	run, err := a.findRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if opts.IncludeLogs {
		logs, err := a.listRunLogs(ctx, runID)
		if err != nil {
			return nil, err
		}
		run["logs"] = logs
	}
	run["run_id"] = asString(run["id"])
	if structured := asAnyMap(run["structured_result"]); structured != nil {
		run["latest_result_payload"] = structured
	} else {
		run["latest_result_payload"] = map[string]any{}
	}
	return run, nil
}

func (a *App) listNodeRuns(ctx context.Context, nodeID string, limit int) ([]jsonMap, error) {
	result, err := a.listNodeRunsWithOptions(ctx, nodeID, runListOptions{Limit: limit, ViewMode: "detail"})
	if err != nil {
		return nil, err
	}
	return workspaceAsItems(result["items"]), nil
}

func (a *App) listNodeRunsWithOptions(ctx context.Context, nodeID string, opts runListOptions) (jsonMap, error) {
	if _, err := a.findNode(ctx, nodeID, false); err != nil {
		return nil, err
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	viewMode := strings.ToLower(strings.TrimSpace(opts.ViewMode))
	if viewMode == "" {
		viewMode = "summary"
	}
	selectFields := `id, task_id, node_id, stage_node_id, actor_type, actor_id, trigger_kind, status, output_preview, output_ref, result, structured_result_json, started_at, finished_at, created_at, updated_at`
	if viewMode == "detail" {
		selectFields = `*`
	}
	query := `SELECT ` + selectFields + ` FROM node_runs WHERE node_id = ?`
	args := []any{nodeID}
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
	for _, item := range items {
		item["run_id"] = asString(item["id"])
		if structured := asAnyMap(item["structured_result"]); structured != nil {
			item["latest_result_payload"] = structured
		} else {
			item["latest_result_payload"] = map[string]any{}
		}
	}
	nextCursor := any(nil)
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = asString(last["created_at"]) + "|" + asString(last["id"])
	}
	return jsonMap{"items": items, "has_more": hasMore, "next_cursor": nextCursor}, nil
}

func (a *App) listTaskRuns(ctx context.Context, taskID string, limit int) ([]jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := a.queryContext(ctx, `SELECT id, task_id, node_id, stage_node_id, actor_type, actor_id, trigger_kind, status, result, output_preview, output_ref, structured_result_json, started_at, finished_at, created_at, updated_at FROM node_runs WHERE task_id = ? ORDER BY updated_at DESC, id DESC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item["run_id"] = asString(item["id"])
		if structured := asAnyMap(item["structured_result"]); structured != nil {
			item["latest_result_payload"] = structured
		} else {
			item["latest_result_payload"] = map[string]any{}
		}
	}
	return items, nil
}

func (a *App) findRun(ctx context.Context, runID string) (jsonMap, error) {
	rows, err := a.queryContext(ctx, `SELECT * FROM node_runs WHERE id = ?`, runID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("run %s not found", runID)}
	}
	return items[0], nil
}

func (a *App) listRunLogs(ctx context.Context, runID string) ([]jsonMap, error) {
	rows, err := a.queryContext(ctx, `SELECT id, run_id, seq, kind, content, payload_json, created_at FROM run_logs WHERE run_id = ? ORDER BY seq ASC, id ASC`, runID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item["log_id"] = asString(item["id"])
	}
	return items, nil
}

func (a *App) ensureSyntheticRun(ctx context.Context, node jsonMap, triggerKind string, message *string, act *actor) (jsonMap, error) {
	if activeRunID := strings.TrimSpace(asString(node["active_run_id"])); activeRunID != "" {
		run, err := a.findRun(ctx, activeRunID)
		if err == nil && asString(run["status"]) == "running" {
			return run, nil
		}
	}
	trigger := triggerKind
	return a.startRun(ctx, asString(node["id"]), runStart{
		Actor:        act,
		TriggerKind:  &trigger,
		InputSummary: message,
	})
}

// aggregateNodeRunLogs queries all run_logs for a node (across all runs),
// formats them into a human-readable execution_log string.
// This replaces manual AI-written execution_log with auto-captured data.
func (a *App) aggregateNodeRunLogs(ctx context.Context, nodeID string) (string, error) {
	rows, err := a.queryContext(ctx, `
		SELECT rl.kind, rl.content, rl.payload_json, rl.created_at,
		       nr.trigger_kind, nr.actor_type, nr.actor_id
		FROM run_logs rl
		JOIN node_runs nr ON rl.run_id = nr.id
		WHERE nr.node_id = ?
		ORDER BY rl.created_at ASC, rl.seq ASC`, nodeID)
	if err != nil {
		return "", err
	}
	items, err := scanRows(rows)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", nil
	}
	var sb strings.Builder
	for _, item := range items {
		ts := asString(item["created_at"])
		if len(ts) >= 19 {
			ts = ts[:10] + " " + ts[11:19] // "2026-04-14 06:30:00"
		}
		kind := asString(item["kind"])
		content := asString(item["content"])
		progress := ""
		if payload := asAnyMap(item["payload"]); payload != nil {
			if p, ok := payload["progress"]; ok {
				progress = fmt.Sprintf(" [%.0f%%]", asFloat(p)*100)
			}
		}
		if content == "" {
			sb.WriteString(fmt.Sprintf("[%s] (%s)%s\n", ts, kind, progress))
		} else {
			sb.WriteString(fmt.Sprintf("[%s] (%s)%s %s\n", ts, kind, progress, content))
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return s
}

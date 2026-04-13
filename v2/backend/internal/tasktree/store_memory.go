package tasktree

import (
	"context"
	"fmt"
	"strings"
)

func (a *App) getNodeMemory(ctx context.Context, nodeID string) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM node_memory_current WHERE node_id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	return a.initNodeMemory(ctx, node)
}

func (a *App) getStageMemory(ctx context.Context, stageNodeID string) (jsonMap, error) {
	stage, err := a.findNode(ctx, stageNodeID, false)
	if err != nil {
		return nil, err
	}
	if asString(stage["role"]) != "stage" {
		return nil, &appError{Code: 400, Msg: "node is not a stage"}
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM stage_memory_current WHERE stage_node_id = ?`, stageNodeID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	return a.initStageMemory(ctx, stage)
}

func (a *App) getTaskMemory(ctx context.Context, taskID string) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM task_memory_current WHERE task_id = ?`, taskID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	return a.initTaskMemory(ctx, task)
}

func (a *App) patchNodeMemoryManualNote(ctx context.Context, nodeID, note string, expectedVersion *int) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		mem, err := a.getNodeMemory(txCtx, nodeID)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(mem, expectedVersion, "node_memory"); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE node_memory_current SET manual_note_text = ?, updated_at = ?, version = version + 1 WHERE node_id = ?`,
			strings.TrimSpace(note), utcNowISO(), nodeID); err != nil {
			return err
		}
		out, err = a.getNodeMemory(txCtx, nodeID)
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil && out != nil {
		a.indexNodeMemory(ctx, out)
	}
	return out, err
}

func (a *App) patchNodeMemoryFull(ctx context.Context, nodeID string, body memoryFullPatchBody) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		mem, err := a.getNodeMemory(txCtx, nodeID)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(mem, body.ExpectedVersion, "node_memory"); err != nil {
			return err
		}

		setClauses := []string{"updated_at = ?", "version = version + 1"}
		args := []any{utcNowISO()}

		if body.SummaryText != nil {
			setClauses = append(setClauses, "summary_text = ?")
			args = append(args, strings.TrimSpace(*body.SummaryText))
		}
		if body.Conclusions != nil {
			setClauses = append(setClauses, "conclusions_json = ?")
			args = append(args, mustJSON(body.Conclusions))
		}
		if body.Decisions != nil {
			setClauses = append(setClauses, "decisions_json = ?")
			args = append(args, mustJSON(body.Decisions))
		}
		if body.Risks != nil {
			setClauses = append(setClauses, "risks_json = ?")
			args = append(args, mustJSON(body.Risks))
		}
		if body.Blockers != nil {
			setClauses = append(setClauses, "blockers_json = ?")
			args = append(args, mustJSON(body.Blockers))
		}
		if body.NextActions != nil {
			setClauses = append(setClauses, "next_actions_json = ?")
			args = append(args, mustJSON(body.NextActions))
		}
		if body.Evidence != nil {
			setClauses = append(setClauses, "evidence_json = ?")
			args = append(args, mustJSON(body.Evidence))
		}
		if body.ExecutionLog != nil {
			setClauses = append(setClauses, "execution_log = ?")
			args = append(args, *body.ExecutionLog)
		} else if body.AppendExecutionLog != nil {
			appendText := strings.TrimSpace(*body.AppendExecutionLog)
			if appendText != "" {
				existing, _ := mem["execution_log"].(string)
				if existing != "" {
					setClauses = append(setClauses, "execution_log = ?")
					args = append(args, existing+"\n"+appendText)
				} else {
					setClauses = append(setClauses, "execution_log = ?")
					args = append(args, appendText)
				}
			}
		}
		if body.ManualNoteText != nil {
			setClauses = append(setClauses, "manual_note_text = ?")
			args = append(args, strings.TrimSpace(*body.ManualNoteText))
		}

		args = append(args, nodeID)
		query := "UPDATE node_memory_current SET " + strings.Join(setClauses, ", ") + " WHERE node_id = ?"
		if _, err := a.execContext(txCtx, query, args...); err != nil {
			return err
		}
		out, err = a.getNodeMemory(txCtx, nodeID)
		return err
	})
	if err == nil && out != nil {
		a.indexNodeMemory(ctx, out)
	}
	return out, err
}

func (a *App) patchStageMemoryManualNote(ctx context.Context, stageNodeID, note string, expectedVersion *int) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		mem, err := a.getStageMemory(txCtx, stageNodeID)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(mem, expectedVersion, "stage_memory"); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE stage_memory_current SET manual_note_text = ?, updated_at = ?, version = version + 1 WHERE stage_node_id = ?`,
			strings.TrimSpace(note), utcNowISO(), stageNodeID); err != nil {
			return err
		}
		out, err = a.getStageMemory(txCtx, stageNodeID)
		if err != nil {
			return err
		}
		return nil
	})
	return out, err
}

func (a *App) patchTaskMemoryManualNote(ctx context.Context, taskID, note string, expectedVersion *int) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		mem, err := a.getTaskMemory(txCtx, taskID)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(mem, expectedVersion, "task_memory"); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE task_memory_current SET manual_note_text = ?, updated_at = ?, version = version + 1 WHERE task_id = ?`,
			strings.TrimSpace(note), utcNowISO(), taskID); err != nil {
			return err
		}
		out, err = a.getTaskMemory(txCtx, taskID)
		if err != nil {
			return err
		}
		return nil
	})
	return out, err
}

func (a *App) patchTaskContext(ctx context.Context, taskID string, body taskContextPatchBody) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		mem, err := a.getTaskMemory(txCtx, taskID)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(mem, body.ExpectedVersion, "task_memory"); err != nil {
			return err
		}
		setClauses := []string{"updated_at = ?", "version = version + 1"}
		args := []any{utcNowISO()}
		if body.ArchitectureDecisions != nil {
			setClauses = append(setClauses, "architecture_decisions_json = ?")
			args = append(args, mustJSON(uniqueStrings(*body.ArchitectureDecisions)))
		}
		if body.ReferenceFiles != nil {
			setClauses = append(setClauses, "reference_files_json = ?")
			args = append(args, mustJSON(uniqueStrings(*body.ReferenceFiles)))
		}
		if body.ContextDocText != nil {
			setClauses = append(setClauses, "context_doc_text = ?")
			args = append(args, strings.TrimSpace(*body.ContextDocText))
		}
		args = append(args, taskID)
		query := "UPDATE task_memory_current SET " + strings.Join(setClauses, ", ") + " WHERE task_id = ?"
		if _, err := a.execContext(txCtx, query, args...); err != nil {
			return err
		}
		out, err = a.getTaskMemory(txCtx, taskID)
		return err
	})
	return out, err
}

func (a *App) initNodeMemory(ctx context.Context, node jsonMap) (jsonMap, error) {
	now := utcNowISO()
	payload := jsonMap{
		"node_id":         asString(node["id"]),
		"task_id":         asString(node["task_id"]),
		"stage_node_id":   nullable(asString(node["stage_node_id"])),
		"summary_text":    defaultNodeMemorySummary(node),
		"conclusions":     []string{},
		"decisions":       []any{},
		"risks":           []any{},
		"blockers":        []any{},
		"next_actions":    []any{},
		"evidence":        []string{},
		"execution_log":    "",
		"manual_note_text": "",
		"source_run_id":   nil,
		"created_at":      now,
		"updated_at":      now,
		"version":         1,
	}
	if _, err := a.execContext(ctx, `INSERT INTO node_memory_current(
			node_id, task_id, stage_node_id, summary_text, conclusions_json, decisions_json, risks_json,
			blockers_json, next_actions_json, evidence_json, execution_log, manual_note_text, source_run_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		payload["node_id"], payload["task_id"], payload["stage_node_id"], payload["summary_text"],
		mustJSON(payload["conclusions"]), mustJSON(payload["decisions"]), mustJSON(payload["risks"]),
		mustJSON(payload["blockers"]), mustJSON(payload["next_actions"]), mustJSON(payload["evidence"]),
		payload["execution_log"], payload["manual_note_text"], payload["source_run_id"], payload["created_at"], payload["updated_at"],
	); err != nil {
		return nil, err
	}
	mem, err := a.getNodeMemory(ctx, asString(node["id"]))
	if err == nil && mem != nil {
		a.indexNodeMemory(ctx, mem)
	}
	return mem, err
}

func (a *App) initStageMemory(ctx context.Context, stage jsonMap) (jsonMap, error) {
	now := utcNowISO()
	if _, err := a.execContext(ctx, `INSERT INTO stage_memory_current(
			stage_node_id, task_id, summary_text, conclusions_json, decisions_json, risks_json,
			blockers_json, next_actions_json, evidence_json, manual_note_text, source_snapshot_ref, created_at, updated_at
		) VALUES (?, ?, ?, '[]', '[]', '[]', '[]', '[]', '[]', '', NULL, ?, ?)`,
		stage["id"], stage["task_id"], defaultStageMemorySummary(stage), now, now,
	); err != nil {
		return nil, err
	}
	return a.getStageMemory(ctx, asString(stage["id"]))
}

func (a *App) initTaskMemory(ctx context.Context, task jsonMap) (jsonMap, error) {
	now := utcNowISO()
	remaining, _ := a.getRemaining(ctx, asString(task["id"]))
	if _, err := a.execContext(ctx, `INSERT INTO task_memory_current(
			task_id, current_stage_node_id, summary_text, conclusions_json, decisions_json, risks_json,
			blockers_json, next_actions_json, evidence_json, manual_note_text, source_snapshot_ref, architecture_decisions_json, reference_files_json, context_doc_text, created_at, updated_at
		) VALUES (?, ?, ?, '[]', '[]', '[]', '[]', '[]', '[]', '', NULL, '[]', '[]', '', ?, ?)`,
		task["id"], nullable(asString(task["current_stage_node_id"])), defaultTaskMemorySummary(task, nil, remaining), now, now,
	); err != nil {
		return nil, err
	}
	return a.getTaskMemory(ctx, asString(task["id"]))
}

func (a *App) refreshNodeMemory(ctx context.Context, nodeID string) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		node, err := a.findNode(txCtx, nodeID, false)
		if err != nil {
			return err
		}
		current, err := a.getNodeMemory(txCtx, nodeID)
		if err != nil {
			return err
		}
		runs, err := a.listNodeRuns(txCtx, nodeID, 100)
		if err != nil {
			return err
		}
		payload := mergeNodeMemoryRuns(node, runs)
		payload["manual_note_text"] = asString(current["manual_note_text"])
		if _, err := a.execContext(txCtx, `UPDATE node_memory_current
			SET summary_text = ?, conclusions_json = ?, decisions_json = ?, risks_json = ?, blockers_json = ?,
			    next_actions_json = ?, evidence_json = ?, manual_note_text = ?, source_run_id = ?, stage_node_id = ?,
			    updated_at = ?, version = version + 1
			WHERE node_id = ?`,
			payload["summary_text"], mustJSON(payload["conclusions"]), mustJSON(payload["decisions"]), mustJSON(payload["risks"]),
			mustJSON(payload["blockers"]), mustJSON(payload["next_actions"]), mustJSON(payload["evidence"]),
			payload["manual_note_text"], payload["source_run_id"], nullable(asString(node["stage_node_id"])),
			utcNowISO(), nodeID,
		); err != nil {
			return err
		}
		out, err = a.getNodeMemory(txCtx, nodeID)
		return err
	})
	return out, err
}

func (a *App) refreshStageMemory(ctx context.Context, stageNodeID string) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		stage, err := a.findNode(txCtx, stageNodeID, false)
		if err != nil {
			return err
		}
		current, err := a.getStageMemory(txCtx, stageNodeID)
		if err != nil {
			return err
		}
		rows, err := a.queryContext(txCtx, `SELECT * FROM node_memory_current WHERE stage_node_id = ? ORDER BY updated_at DESC`, stageNodeID)
		if err != nil {
			return err
		}
		nodeMemories, err := scanRows(rows)
		if err != nil {
			return err
		}
		payload := mergeStageMemory(stage, nodeMemories)
		payload["manual_note_text"] = asString(current["manual_note_text"])
		if _, err := a.execContext(txCtx, `UPDATE stage_memory_current
			SET summary_text = ?, conclusions_json = ?, decisions_json = ?, risks_json = ?, blockers_json = ?,
			    next_actions_json = ?, evidence_json = ?, manual_note_text = ?, updated_at = ?, version = version + 1
			WHERE stage_node_id = ?`,
			payload["summary_text"], mustJSON(payload["conclusions"]), mustJSON(payload["decisions"]), mustJSON(payload["risks"]),
			mustJSON(payload["blockers"]), mustJSON(payload["next_actions"]), mustJSON(payload["evidence"]),
			payload["manual_note_text"], utcNowISO(), stageNodeID,
		); err != nil {
			return err
		}
		out, err = a.getStageMemory(txCtx, stageNodeID)
		return err
	})
	return out, err
}

func (a *App) refreshTaskMemory(ctx context.Context, taskID string) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		task, err := a.getTask(txCtx, taskID, false)
		if err != nil {
			return err
		}
		current, err := a.getTaskMemory(txCtx, taskID)
		if err != nil {
			return err
		}
		stageNodeID := asString(task["current_stage_node_id"])
		var currentStage jsonMap
		if stageNodeID != "" {
			currentStage, _ = a.findNode(txCtx, stageNodeID, false)
		}
		rows, err := a.queryContext(txCtx, `SELECT * FROM stage_memory_current WHERE task_id = ? ORDER BY updated_at DESC`, taskID)
		if err != nil {
			return err
		}
		stageMemories, err := scanRows(rows)
		if err != nil {
			return err
		}
		remaining, err := a.getRemaining(txCtx, taskID)
		if err != nil {
			return err
		}
		payload := mergeTaskMemory(task, currentStage, stageMemories, remaining)
		payload["manual_note_text"] = asString(current["manual_note_text"])
		payload["architecture_decisions"] = stringSliceFromAny(current["architecture_decisions"])
		payload["reference_files"] = stringSliceFromAny(current["reference_files"])
		payload["context_doc_text"] = asString(current["context_doc_text"])
		if _, err := a.execContext(txCtx, `UPDATE task_memory_current
			SET current_stage_node_id = ?, summary_text = ?, conclusions_json = ?, decisions_json = ?, risks_json = ?,
			    blockers_json = ?, next_actions_json = ?, evidence_json = ?, manual_note_text = ?, architecture_decisions_json = ?, reference_files_json = ?, context_doc_text = ?, updated_at = ?, version = version + 1
			WHERE task_id = ?`,
			nullable(stageNodeID), payload["summary_text"], mustJSON(payload["conclusions"]), mustJSON(payload["decisions"]),
			mustJSON(payload["risks"]), mustJSON(payload["blockers"]), mustJSON(payload["next_actions"]), mustJSON(payload["evidence"]),
			payload["manual_note_text"], mustJSON(payload["architecture_decisions"]), mustJSON(payload["reference_files"]), payload["context_doc_text"], utcNowISO(), taskID,
		); err != nil {
			return err
		}
		out, err = a.getTaskMemory(txCtx, taskID)
		return err
	})
	return out, err
}

func (a *App) refreshMemoryChainForNode(ctx context.Context, nodeID string) error {
	return a.withTx(ctx, func(txCtx context.Context) error {
		node, err := a.findNode(txCtx, nodeID, false)
		if err != nil {
			return err
		}
		if _, err := a.refreshNodeMemory(txCtx, nodeID); err != nil {
			return err
		}
		stageNodeID := asString(node["stage_node_id"])
		if stageNodeID != "" {
			if _, err := a.refreshStageMemory(txCtx, stageNodeID); err != nil {
				return err
			}
		}
		if _, err := a.refreshTaskMemory(txCtx, asString(node["task_id"])); err != nil {
			return err
		}
		return nil
	})
}

func (a *App) maybeSnapshotForNode(ctx context.Context, node jsonMap) error {
	if asString(node["role"]) == "stage" && (isDoneResult(node) || isCanceledResult(node)) {
		stageMemory, err := a.getStageMemory(ctx, asString(node["id"]))
		if err != nil {
			return err
		}
		_, err = a.createMemorySnapshot(ctx, "stage", asString(node["task_id"]), asString(node["id"]), "", stageMemory, "stage_closed")
		return err
	}
	return nil
}

func (a *App) maybeSnapshotForTask(ctx context.Context, task jsonMap) error {
	switch asString(task["status"]) {
	case "paused", "done", "canceled", "closed":
		taskMemory, err := a.getTaskMemory(ctx, asString(task["id"]))
		if err != nil {
			return err
		}
		_, err = a.createMemorySnapshot(ctx, "task", asString(task["id"]), asString(task["current_stage_node_id"]), "", taskMemory, "task_"+asString(task["status"]))
		return err
	default:
		return nil
	}
}

func (a *App) createMemorySnapshot(ctx context.Context, scopeKind, taskID, stageNodeID, nodeID string, memory jsonMap, reason string) (jsonMap, error) {
	id := newID("msnap")
	now := utcNowISO()
	if _, err := a.execContext(ctx, `INSERT INTO memory_snapshots(id, scope_kind, task_id, stage_node_id, node_id, summary_text, payload_json, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, scopeKind, taskID, nullable(stageNodeID), nullable(nodeID), asString(memory["summary_text"]), mustJSON(memory), nullable(reason), now,
	); err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM memory_snapshots WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("memory snapshot %s not found after insert", id)
	}
	return items[0], nil
}

func (a *App) snapshotMemoryManually(ctx context.Context, scopeKind, refID string) (jsonMap, error) {
	switch scopeKind {
	case "task":
		memory, err := a.getTaskMemory(ctx, refID)
		if err != nil {
			return nil, err
		}
		task, err := a.getTask(ctx, refID, false)
		if err != nil {
			return nil, err
		}
		return a.createMemorySnapshot(ctx, "task", refID, asString(task["current_stage_node_id"]), "", memory, "manual")
	case "stage":
		stage, err := a.findNode(ctx, refID, false)
		if err != nil {
			return nil, err
		}
		memory, err := a.getStageMemory(ctx, refID)
		if err != nil {
			return nil, err
		}
		return a.createMemorySnapshot(ctx, "stage", asString(stage["task_id"]), refID, "", memory, "manual")
	case "node":
		node, err := a.findNode(ctx, refID, false)
		if err != nil {
			return nil, err
		}
		memory, err := a.getNodeMemory(ctx, refID)
		if err != nil {
			return nil, err
		}
		return a.createMemorySnapshot(ctx, "node", asString(node["task_id"]), asString(node["stage_node_id"]), refID, memory, "manual")
	default:
		return nil, &appError{Code: 400, Msg: "unsupported snapshot scope"}
	}
}

func (a *App) initializeTaskMemories(ctx context.Context, taskID string) error {
	return a.withTx(ctx, func(txCtx context.Context) error {
		task, err := a.getTask(txCtx, taskID, false)
		if err != nil {
			return err
		}
		if _, err := a.getTaskMemory(txCtx, taskID); err != nil {
			return err
		}
		nodes, err := a.listNodes(txCtx, taskID)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if _, err := a.getNodeMemory(txCtx, asString(node["id"])); err != nil {
				return err
			}
			if asString(node["role"]) == "stage" {
				if _, err := a.getStageMemory(txCtx, asString(node["id"])); err != nil {
					return err
				}
			}
		}
		if _, err := a.refreshTaskMemory(txCtx, taskID); err != nil {
			return err
		}
		return a.maybeSnapshotForTask(txCtx, task)
	})
}

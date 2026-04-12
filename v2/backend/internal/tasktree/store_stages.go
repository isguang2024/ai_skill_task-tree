package tasktree

import (
	"context"
	"strings"
)

func normalizeNodeRole(kind string, role *string) string {
	if role != nil && strings.TrimSpace(*role) != "" {
		return strings.TrimSpace(*role)
	}
	if kind == "group" {
		return "container"
	}
	return "step"
}

func (a *App) taskHasStages(ctx context.Context, taskID string) (bool, error) {
	var count int
	if err := a.queryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE task_id = ? AND deleted_at IS NULL AND role = 'stage'`, taskID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *App) resolveActiveStageNodeID(ctx context.Context, taskID string) (string, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return "", err
	}
	stageNodeID := strings.TrimSpace(asString(task["current_stage_node_id"]))
	if stageNodeID != "" {
		stage, err := a.findNode(ctx, stageNodeID, false)
		if err == nil && asString(stage["role"]) == "stage" {
			return stageNodeID, nil
		}
	}
	rows := a.queryRowContext(ctx, `SELECT id FROM nodes WHERE task_id = ? AND deleted_at IS NULL AND role = 'stage' ORDER BY sort_order, created_at LIMIT 1`, taskID)
	if err := rows.Scan(&stageNodeID); err != nil {
		return "", err
	}
	return stageNodeID, nil
}

func (a *App) listStages(ctx context.Context, taskID string) ([]jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM nodes WHERE task_id = ? AND deleted_at IS NULL AND role = 'stage' ORDER BY sort_order, created_at`, taskID)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func (a *App) createStage(ctx context.Context, taskID string, body stageCreate) (jsonMap, error) {
	if strings.TrimSpace(body.Title) == "" {
		return nil, &appError{Code: 400, Msg: "title required"}
	}
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	if err := ensureExpectedVersion(task, body.ExpectedVersion, "task"); err != nil {
		return nil, err
	}
	stage, err := a.createNode(ctx, taskID, nodeCreate{
		NodeKey:            body.NodeKey,
		Kind:               "group",
		Role:               strPtr("stage"),
		Title:              body.Title,
		Instruction:        body.Instruction,
		AcceptanceCriteria: body.AcceptanceCriteria,
		Estimate:           body.Estimate,
		SortOrder:          body.SortOrder,
		Metadata:           body.Metadata,
	})
	if err != nil {
		return nil, err
	}
	activate := body.Activate != nil && *body.Activate
	if !activate && strings.TrimSpace(asString(task["current_stage_node_id"])) == "" {
		activate = true
	}
	if activate {
		if _, err := a.activateStage(ctx, taskID, asString(stage["id"]), stageActivate{}); err != nil {
			return nil, err
		}
		return a.findNode(ctx, asString(stage["id"]), false)
	}
	return stage, nil
}

func (a *App) activateStage(ctx context.Context, taskID, stageNodeID string, body stageActivate) (jsonMap, error) {
	var out jsonMap
	err := a.withTx(ctx, func(txCtx context.Context) error {
		task, err := a.getTask(txCtx, taskID, false)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(task, body.ExpectedVersion, "task"); err != nil {
			return err
		}
		stage, err := a.findNode(txCtx, stageNodeID, false)
		if err != nil {
			return err
		}
		if asString(stage["task_id"]) != taskID {
			return &appError{Code: 400, Msg: "stage node belongs to another task"}
		}
		if asString(stage["role"]) != "stage" {
			return &appError{Code: 400, Msg: "node is not a stage"}
		}
		if asString(stage["parent_node_id"]) != "" {
			return &appError{Code: 400, Msg: "stage must stay at root level"}
		}
		if _, err := a.execContext(txCtx, `UPDATE tasks SET current_stage_node_id = ?, updated_at = ?, version = version + 1 WHERE id = ?`, stageNodeID, utcNowISO(), taskID); err != nil {
			return err
		}
		msg := "阶段已切换"
		if body.Message != nil && strings.TrimSpace(*body.Message) != "" {
			msg = strings.TrimSpace(*body.Message)
		}
		if err := a.insertEvent(txCtx, taskID, &stageNodeID, "stage_activated", &msg, map[string]any{"stage_node_id": stageNodeID}, body.Actor, nil); err != nil {
			return err
		}
		out, err = a.getTask(txCtx, taskID, false)
		return err
	})
	return out, err
}

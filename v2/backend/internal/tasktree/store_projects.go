package tasktree

import (
	"context"
	"fmt"
	"strings"
)

func (a *App) ensureDefaultProject(ctx context.Context) error {
	rows, err := a.queryContext(ctx, `SELECT * FROM projects WHERE is_default = 1 AND deleted_at IS NULL LIMIT 1`)
	if err != nil {
		return err
	}
	items, err := scanRows(rows)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return nil
	}
	now := utcNowISO()
	id := newID("prj")
	if _, err := a.execContext(ctx, `INSERT INTO projects(id, project_key, name, description, is_default, metadata_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?)`, id, "DEFAULT", "默认项目", "未指定项目时自动归档到这里。", mustJSON(map[string]any{}), now, now); err != nil {
		return err
	}
	return nil
}

func (a *App) listProjects(ctx context.Context, q string, includeDeleted bool, limit int) ([]jsonMap, error) {
	query := `SELECT * FROM projects WHERE 1=1`
	args := []any{}
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	if text := strings.TrimSpace(q); text != "" {
		like := "%" + text + "%"
		query += ` AND (name LIKE ? OR project_key LIKE ? OR description LIKE ?)`
		args = append(args, like, like, like)
	}
	query += ` ORDER BY is_default DESC, updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func (a *App) getProject(ctx context.Context, projectID string, includeDeleted bool) (jsonMap, error) {
	rows, err := a.queryContext(ctx, `SELECT * FROM projects WHERE id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("project %s not found", projectID)}
	}
	if items[0]["deleted_at"] != nil && !includeDeleted {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("project %s deleted", projectID)}
	}
	return items[0], nil
}

func (a *App) findProjectByKey(ctx context.Context, projectKey string) (jsonMap, error) {
	rows, err := a.queryContext(ctx, `SELECT * FROM projects WHERE project_key = ? AND deleted_at IS NULL LIMIT 1`, strings.TrimSpace(projectKey))
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("project key %s not found", projectKey)}
	}
	return items[0], nil
}

func (a *App) defaultProject(ctx context.Context) (jsonMap, error) {
	if err := a.ensureDefaultProject(ctx); err != nil {
		return nil, err
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM projects WHERE is_default = 1 AND deleted_at IS NULL LIMIT 1`)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 500, Msg: "default project missing"}
	}
	return items[0], nil
}

func (a *App) createProject(ctx context.Context, body projectCreate) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return &appError{Code: 400, Msg: "name required"}
		}
		if body.Metadata == nil {
			body.Metadata = map[string]any{}
		}
		now := utcNowISO()
		id := newID("prj")
		projectKey := ""
		if body.ProjectKey != nil {
			projectKey = strings.TrimSpace(*body.ProjectKey)
		}
		description := ""
		if body.Description != nil {
			description = strings.TrimSpace(*body.Description)
		}
		isDefault := body.IsDefault != nil && *body.IsDefault
		if isDefault {
			if _, err := a.execContext(txCtx, `UPDATE projects SET is_default = 0, updated_at = ? WHERE is_default = 1 AND deleted_at IS NULL`, now); err != nil {
				return err
			}
		}
		if _, err := a.execContext(txCtx, `INSERT INTO projects(id, project_key, name, description, is_default, metadata_json, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, nullable(projectKey), name, nullable(description), boolToInt(isDefault), mustJSON(body.Metadata), now, now); err != nil {
			return err
		}
		out, err = a.getProject(txCtx, id, false)
		return err
	})
	return out, err
}

func (a *App) updateProject(ctx context.Context, projectID string, body projectUpdate) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		if _, err := a.getProject(txCtx, projectID, false); err != nil {
			return err
		}
		project, err := a.getProject(txCtx, projectID, false)
		if err != nil {
			return err
		}
		if err := ensureExpectedVersion(project, body.ExpectedVersion, "project"); err != nil {
			return err
		}
		updates := []string{}
		args := []any{}
		if body.ProjectKey != nil {
			key := strings.TrimSpace(*body.ProjectKey)
			updates = append(updates, "project_key = ?")
			args = append(args, nullable(key))
		}
		if body.Name != nil {
			name := strings.TrimSpace(*body.Name)
			if name == "" {
				return &appError{Code: 400, Msg: "name 不能为空"}
			}
			updates = append(updates, "name = ?")
			args = append(args, name)
		}
		if body.Description != nil {
			description := strings.TrimSpace(*body.Description)
			updates = append(updates, "description = ?")
			args = append(args, nullable(description))
		}
		if body.Metadata != nil {
			updates = append(updates, "metadata_json = ?")
			args = append(args, mustJSON(body.Metadata))
		}
		if body.IsDefault != nil {
			if *body.IsDefault {
				if _, err := a.execContext(txCtx, `UPDATE projects SET is_default = 0, updated_at = ? WHERE is_default = 1 AND deleted_at IS NULL`, utcNowISO()); err != nil {
					return err
				}
			}
			updates = append(updates, "is_default = ?")
			args = append(args, boolToInt(*body.IsDefault))
		}
		if len(updates) == 0 {
			out, err = a.getProject(txCtx, projectID, false)
			return err
		}
		updates = append(updates, "updated_at = ?")
		args = append(args, utcNowISO(), projectID)
		if _, err := a.execContext(txCtx, `UPDATE projects SET `+strings.Join(updates, ", ")+` WHERE id = ?`, args...); err != nil {
			return err
		}
		out, err = a.getProject(txCtx, projectID, false)
		return err
	})
	return out, err
}

func (a *App) projectOverview(ctx context.Context, projectID string, includeDeleted bool, limit int) (jsonMap, error) {
	project, err := a.getProject(ctx, projectID, includeDeleted)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM tasks WHERE project_id = ?`
	args := []any{projectID}
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	tasks, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	summary := map[string]int{
		"total":    len(tasks),
		"ready":    0,
		"running":  0,
		"blocked":  0,
		"paused":   0,
		"done":     0,
		"canceled": 0,
		"closed":   0,
	}
	for _, task := range tasks {
		state := asString(task["status"])
		if _, ok := summary[state]; ok {
			summary[state]++
		}
		if remaining, err := a.getRemaining(ctx, asString(task["id"])); err == nil {
			task["remaining_nodes"] = remaining["remaining_nodes"]
			task["blocked_nodes"] = remaining["blocked_nodes"]
			task["paused_nodes"] = remaining["paused_nodes"]
		}
		if currentStageID := asString(task["current_stage_node_id"]); currentStageID != "" {
			if stage, err := a.findNode(ctx, currentStageID, false); err == nil {
				task["current_stage"] = jsonMap{
					"id":    stage["id"],
					"path":  stage["path"],
					"title": stage["title"],
				}
			}
		}
		if mem, err := a.getTaskMemory(ctx, asString(task["id"])); err == nil {
			task["memory"] = mem
		}
	}
	return jsonMap{
		"project": project,
		"summary": summary,
		"tasks":   tasks,
	}, nil
}

func (a *App) resolveTaskProjectID(ctx context.Context, projectID, projectKey *string) (string, error) {
	if projectID != nil && strings.TrimSpace(*projectID) != "" {
		project, err := a.getProject(ctx, strings.TrimSpace(*projectID), false)
		if err != nil {
			return "", err
		}
		return asString(project["id"]), nil
	}
	if projectKey != nil && strings.TrimSpace(*projectKey) != "" {
		project, err := a.findProjectByKey(ctx, strings.TrimSpace(*projectKey))
		if err != nil {
			return "", err
		}
		return asString(project["id"]), nil
	}
	project, err := a.defaultProject(ctx)
	if err != nil {
		return "", err
	}
	return asString(project["id"]), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (a *App) deleteProject(ctx context.Context, projectID string) error {
	return a.withTx(ctx, func(txCtx context.Context) error {
		proj, err := a.getProject(txCtx, projectID, false)
		if err != nil {
			return err
		}
		if asString(proj["is_default"]) == "1" {
			return &appError{Code: 400, Msg: "默认项目不能删除"}
		}
		now := utcNowISO()
		rows, err := a.queryContext(txCtx, `SELECT id FROM tasks WHERE project_id = ? AND deleted_at IS NULL`, projectID)
		if err != nil {
			return err
		}
		items, err := scanRows(rows)
		if err != nil {
			return err
		}
		for _, item := range items {
			if _, err := a.softDeleteTask(txCtx, asString(item["id"])); err != nil {
				return err
			}
		}
		if _, err := a.execContext(txCtx,
			`UPDATE projects SET deleted_at = ?, updated_at = ? WHERE id = ?`,
			now, now, projectID); err != nil {
			return err
		}
		return nil
	})
}

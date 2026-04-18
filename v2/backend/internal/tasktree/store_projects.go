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

func (a *App) listProjectsWithStats(ctx context.Context, q string, includeDeleted bool, limit int) ([]jsonMap, error) {
	projects, err := a.listProjects(ctx, q, includeDeleted, limit)
	if err != nil {
		return nil, err
	}
	query := `SELECT project_id,
		COUNT(*) AS total,
		SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) AS running,
		SUM(CASE WHEN status = 'blocked' THEN 1 ELSE 0 END) AS blocked,
		SUM(CASE WHEN status = 'paused' THEN 1 ELSE 0 END) AS paused,
		SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) AS done
		FROM tasks WHERE project_id IS NOT NULL`
	args := []any{}
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` GROUP BY project_id`
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	stats, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	byProject := map[string]jsonMap{}
	for _, stat := range stats {
		byProject[asString(stat["project_id"])] = jsonMap{
			"total":   int(asFloat(stat["total"])),
			"running": int(asFloat(stat["running"])),
			"blocked": int(asFloat(stat["blocked"])),
			"paused":  int(asFloat(stat["paused"])),
			"done":    int(asFloat(stat["done"])),
		}
	}
	for _, project := range projects {
		project["_summary"] = byProject[asString(project["id"])]
	}
	return projects, nil
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
	query := `SELECT id, task_key, title, status, result, usage_tokens, summary_percent, summary_done, summary_total, summary_blocked, current_stage_node_id, project_id, updated_at, created_at FROM tasks WHERE project_id = ?`
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
	taskIDs := make([]string, 0, len(tasks))
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
		taskIDs = append(taskIDs, asString(task["id"]))
		state := asString(task["status"])
		if _, ok := summary[state]; ok {
			summary[state]++
		}
	}
	if len(taskIDs) > 0 {
		placeholdersText := placeholders(len(taskIDs))
		args := make([]any, 0, len(taskIDs))
		for _, taskID := range taskIDs {
			args = append(args, taskID)
		}
		remainingRows, err := a.queryContext(ctx, `SELECT task_id,
			SUM(CASE WHEN kind != 'group' AND COALESCE(result, '') != 'done' AND COALESCE(result, '') != 'canceled' THEN 1 ELSE 0 END) AS remaining_nodes,
			SUM(CASE WHEN kind != 'group' AND status = 'blocked' THEN 1 ELSE 0 END) AS blocked_nodes,
			SUM(CASE WHEN kind != 'group' AND status = 'paused' THEN 1 ELSE 0 END) AS paused_nodes
			FROM nodes WHERE deleted_at IS NULL AND task_id IN (`+placeholdersText+`) GROUP BY task_id`, args...)
		if err != nil {
			return nil, err
		}
		remainingItems, err := scanRows(remainingRows)
		if err != nil {
			return nil, err
		}
		remainingByTask := map[string]jsonMap{}
		for _, item := range remainingItems {
			remainingByTask[asString(item["task_id"])] = item
		}

		stageIDs := make([]string, 0, len(tasks))
		for _, task := range tasks {
			if currentStageID := asString(task["current_stage_node_id"]); currentStageID != "" {
				stageIDs = append(stageIDs, currentStageID)
			}
		}
		stageByID := map[string]jsonMap{}
		if len(stageIDs) > 0 {
			stageArgs := make([]any, 0, len(stageIDs))
			for _, stageID := range stageIDs {
				stageArgs = append(stageArgs, stageID)
			}
			stageRows, err := a.queryContext(ctx, `SELECT id, path, title FROM nodes WHERE deleted_at IS NULL AND id IN (`+placeholders(len(stageIDs))+`)`, stageArgs...)
			if err != nil {
				return nil, err
			}
			stageItems, err := scanRows(stageRows)
			if err != nil {
				return nil, err
			}
			for _, item := range stageItems {
				stageByID[asString(item["id"])] = item
			}
		}

		memoryRows, err := a.queryContext(ctx, `SELECT task_id, summary_text, next_actions_json, risks_json, blockers_json, decisions_json, version, updated_at FROM task_memory_current WHERE task_id IN (`+placeholdersText+`)`, args...)
		if err != nil {
			return nil, err
		}
		memoryItems, err := scanRows(memoryRows)
		if err != nil {
			return nil, err
		}
		memoryByTask := map[string]jsonMap{}
		for _, item := range memoryItems {
			memoryByTask[asString(item["task_id"])] = jsonMap{
				"task_id":      item["task_id"],
				"summary":      item["summary_text"],
				"summary_text": item["summary_text"],
				"next_actions": item["next_actions"],
				"risks":        item["risks"],
				"blockers":     item["blockers"],
				"decisions":    item["decisions"],
				"version":      item["version"],
				"updated_at":   item["updated_at"],
			}
		}

		for _, task := range tasks {
			taskID := asString(task["id"])
			if remaining := remainingByTask[taskID]; remaining != nil {
				task["remaining_nodes"] = remaining["remaining_nodes"]
				task["blocked_nodes"] = remaining["blocked_nodes"]
				task["paused_nodes"] = remaining["paused_nodes"]
			} else {
				task["remaining_nodes"] = 0
				task["blocked_nodes"] = 0
				task["paused_nodes"] = 0
			}
			if stage := stageByID[asString(task["current_stage_node_id"])]; stage != nil {
				task["current_stage"] = jsonMap{
					"id":    stage["id"],
					"path":  stage["path"],
					"title": stage["title"],
				}
			}
			if mem := memoryByTask[taskID]; mem != nil {
				task["memory"] = mem
			} else {
				task["memory"] = jsonMap{
					"task_id":      task["id"],
					"summary":      "",
					"summary_text": "",
					"next_actions": []any{},
					"risks":        []any{},
					"blockers":     []any{},
					"decisions":    []any{},
				}
			}
		}
	}
	overviewTasks := make([]jsonMap, 0, len(tasks))
	for _, task := range tasks {
		remainingSummary := jsonMap{
			"nodes":   task["remaining_nodes"],
			"blocked": task["blocked_nodes"],
			"paused":  task["paused_nodes"],
		}
		if remainingSummary["nodes"] == nil {
			remainingSummary["nodes"] = 0
		}
		if remainingSummary["blocked"] == nil {
			remainingSummary["blocked"] = 0
		}
		if remainingSummary["paused"] == nil {
			remainingSummary["paused"] = 0
		}
		memorySummary := asAnyMap(task["memory"])
		if memorySummary == nil {
			memorySummary = jsonMap{
				"summary":      "",
				"summary_text": "",
				"next_actions": []any{},
			}
		}
		overviewTasks = append(overviewTasks, jsonMap{
			"id":              task["id"],
			"task_key":        task["task_key"],
			"title":           task["title"],
			"status":          task["status"],
			"result":          task["result"],
			"usage_tokens":    task["usage_tokens"],
			"summary_percent": task["summary_percent"],
			"updated_at":      task["updated_at"],
			"current_stage":   task["current_stage"],
			"remaining":       remainingSummary,
			"memory": jsonMap{
				"summary":      memorySummary["summary"],
				"summary_text": memorySummary["summary_text"],
				"next_actions": memorySummary["next_actions"],
			},
		})
	}
	return jsonMap{
		"project": project,
		"summary": summary,
		"tasks":   overviewTasks,
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

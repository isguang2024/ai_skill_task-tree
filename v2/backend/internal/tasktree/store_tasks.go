package tasktree

import (
	"context"
	"fmt"
	"strings"
)

func (a *App) createTask(ctx context.Context, body taskCreate) (jsonMap, error) {
	if body.DryRun != nil && *body.DryRun {
		return a.previewCreateTask(ctx, body)
	}
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		if body.Title == "" {
			return &appError{Code: 400, Msg: "title required"}
		}
		projectID, err := a.resolveTaskProjectID(txCtx, body.ProjectID, body.ProjectKey)
		if err != nil {
			return err
		}
		taskID := newID("tsk")
		now := utcNowISO()
		if body.Tags == nil {
			body.Tags = []string{}
		}
		if body.Metadata == nil {
			body.Metadata = map[string]any{}
		}
		createdByType := "human"
		if body.CreatedByType != nil && *body.CreatedByType != "" {
			createdByType = *body.CreatedByType
		}
		if _, err := a.execContext(txCtx, `INSERT INTO tasks(id, task_key, title, goal, status, source_tool, source_session_id, created_by_type, created_by_id, creation_mode, creation_reason, tags_json, metadata_json, project_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'ready', ?, ?, ?, ?, 'auto', ?, ?, ?, ?, ?, ?)`,
			taskID, body.TaskKey, body.Title, body.Goal, body.SourceTool, body.SourceSessionID, createdByType, body.CreatedByID, body.CreationReason, mustJSON(body.Tags), mustJSON(body.Metadata), projectID, now, now); err != nil {
			return err
		}
		if err := a.insertEvent(txCtx, taskID, nil, "task_created", &body.Title, nil, &actor{Tool: body.SourceTool, AgentID: body.CreatedByID}, nil); err != nil {
			return err
		}
		// 创建阶段并建立 key → stage_node_id 映射
		stageKeyMap := make(map[string]string, len(body.Stages))
		for _, stage := range body.Stages {
			created, err := a.createStage(txCtx, taskID, stage)
			if err != nil {
				return err
			}
			if stage.Key != nil && *stage.Key != "" {
				stageKeyMap[*stage.Key] = asString(created["id"])
			}
		}
		for _, seed := range body.Nodes {
			if _, err := a.createTaskSeedNode(txCtx, taskID, nil, seed, stageKeyMap); err != nil {
				return err
			}
		}
		out, err = a.getTask(txCtx, taskID, false)
		return err
	})
	if err == nil && out != nil {
		a.indexTask(ctx, out)
	}
	return out, err
}

func (a *App) createTaskSeedNode(ctx context.Context, taskID string, parentNodeID *string, seed taskNodeSeed, stageKeyMap map[string]string) (jsonMap, error) {
	kind := seed.Kind
	if len(seed.Children) > 0 {
		kind = "group"
	} else if kind != "group" && kind != "leaf" {
		kind = "leaf" // normalize: "task", "", or any other value → "leaf"
	}
	// resolve stage_key → stage_node_id
	var stageNodeID *string
	if seed.StageKey != nil && *seed.StageKey != "" {
		if sid, ok := stageKeyMap[*seed.StageKey]; ok {
			stageNodeID = &sid
		}
	}
	created, err := a.createNode(ctx, taskID, nodeCreate{
		ParentNodeID:       parentNodeID,
		StageNodeID:        stageNodeID,
		NodeKey:            seed.NodeKey,
		Kind:               kind,
		Title:              seed.Title,
		Instruction:        seed.Instruction,
		AcceptanceCriteria: seed.AcceptanceCriteria,
		DependsOn:          seed.DependsOn,
		DependsOnKeys:      seed.DependsOnKeys,
		Estimate:           seed.Estimate,
		Status:             seed.Status,
		SortOrder:          seed.SortOrder,
		Metadata:           seed.Metadata,
		CreatedByType:      seed.CreatedByType,
		CreatedByID:        seed.CreatedByID,
		CreationReason:     seed.CreationReason,
	})
	if err != nil {
		return nil, err
	}
	nodeID := asString(created["id"])
	if nodeID == "" {
		return nil, fmt.Errorf("seed node created without id")
	}
	for _, child := range seed.Children {
		if _, err := a.createTaskSeedNode(ctx, taskID, &nodeID, child, stageKeyMap); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func (a *App) previewCreateTask(_ context.Context, body taskCreate) (jsonMap, error) {
	if strings.TrimSpace(body.Title) == "" {
		return nil, &appError{Code: 400, Msg: "title required"}
	}
	taskKey := ""
	if body.TaskKey != nil {
		taskKey = strings.TrimSpace(*body.TaskKey)
	}
	if taskKey == "" {
		taskKey = "DRYRUN"
	}
	type previewNode struct {
		ID          string
		Path        string
		Title       string
		NodeKey     string
		DependsOn   []string
		DependsKeys []string
	}
	nodes := make([]previewNode, 0, len(body.Nodes)+len(body.Stages))
	keyToNodes := map[string][]previewNode{}
	type stageInfo struct {
		id   string
		path string
	}
	stageKey := ""
	activeStageID := ""
	activeStageKey := ""
	activeStagePath := ""
	stageKeyToInfo := map[string]stageInfo{}
	for idx, stage := range body.Stages {
		nk := strings.TrimSpace(strPtrValue(stage.NodeKey))
		if nk == "" {
			nk = fmt.Sprintf("%d", idx+1)
		}
		id := fmt.Sprintf("dry_stage_%d", idx+1)
		path := taskKey + "/" + nk
		p := previewNode{ID: id, Path: path, Title: stage.Title, NodeKey: nk}
		nodes = append(nodes, p)
		keyToNodes[nk] = append(keyToNodes[nk], p)
		if stage.Key != nil && *stage.Key != "" {
			stageKeyToInfo[*stage.Key] = stageInfo{id: id, path: path}
		}
		if idx == 0 {
			activeStageID = id
			activeStageKey = nk
			activeStagePath = path
		}
		if stage.Activate != nil && *stage.Activate {
			activeStageID = id
			activeStageKey = nk
			activeStagePath = path
		}
	}
	if activeStageID != "" {
		stageKey = activeStageKey
	}
	var walk func(parentPath string, parentID string, siblingCounter map[string]int, seed taskNodeSeed)
	walk = func(parentPath string, parentID string, siblingCounter map[string]int, seed taskNodeSeed) {
		nk := strings.TrimSpace(strPtrValue(seed.NodeKey))
		counterKey := parentPath
		if counterKey == "" {
			counterKey = "__root__"
		}
		if nk == "" {
			siblingCounter[counterKey]++
			nk = fmt.Sprintf("%d", siblingCounter[counterKey])
		}
		path := taskKey + "/" + nk
		if parentPath != "" {
			path = parentPath + "/" + nk
		}
		id := fmt.Sprintf("dry_%d", len(nodes)+1)
		p := previewNode{
			ID:          id,
			Path:        path,
			Title:       seed.Title,
			NodeKey:     nk,
			DependsOn:   uniqueStrings(seed.DependsOn),
			DependsKeys: uniqueStrings(seed.DependsOnKeys),
		}
		_ = parentID
		nodes = append(nodes, p)
		keyToNodes[nk] = append(keyToNodes[nk], p)
		childCounter := map[string]int{}
		for _, child := range seed.Children {
			walk(path, id, childCounter, child)
		}
	}
	rootCounter := map[string]int{}
	for _, seed := range body.Nodes {
		rootParentPath := ""
		rootParentID := ""
		// resolve stage_key → specific stage; fallback to active stage
		if seed.StageKey != nil && *seed.StageKey != "" {
			if info, ok := stageKeyToInfo[*seed.StageKey]; ok {
				rootParentPath = info.path
				rootParentID = info.id
			}
		} else if stageKey != "" {
			rootParentPath = activeStagePath
			rootParentID = activeStageID
		}
		walk(rootParentPath, rootParentID, rootCounter, seed)
	}

	resolved := []jsonMap{}
	errors := []string{}
	for _, node := range nodes {
		if len(node.DependsKeys) == 0 {
			continue
		}
		depIDs := append([]string{}, node.DependsOn...)
		for _, key := range node.DependsKeys {
			matches := keyToNodes[key]
			if len(matches) == 0 {
				errors = append(errors, fmt.Sprintf("节点 %s 的 depends_on_keys=%q 未匹配到节点", node.Path, key))
				continue
			}
			if len(matches) > 1 {
				paths := make([]string, 0, len(matches))
				for _, match := range matches {
					paths = append(paths, match.Path)
				}
				errors = append(errors, fmt.Sprintf("节点 %s 的 depends_on_keys=%q 匹配到多个节点：%s", node.Path, key, strings.Join(paths, ", ")))
				continue
			}
			depIDs = append(depIDs, matches[0].ID)
		}
		resolved = append(resolved, jsonMap{
			"path":              node.Path,
			"title":             node.Title,
			"depends_on_keys":   node.DependsKeys,
			"depends_on_ids":    uniqueStrings(depIDs),
			"depends_on_origin": node.DependsOn,
		})
	}

	previewTree := make([]jsonMap, 0, len(nodes))
	for _, node := range nodes {
		previewTree = append(previewTree, jsonMap{
			"id":       node.ID,
			"path":     node.Path,
			"title":    node.Title,
			"node_key": node.NodeKey,
		})
	}
	return jsonMap{
		"dry_run":               true,
		"validated":             len(errors) == 0,
		"task_key":              taskKey,
		"active_stage_node_id":  activeStageID,
		"preview_tree":          previewTree,
		"resolved_dependencies": resolved,
		"errors":                errors,
		"warnings":              []string{},
	}, nil
}

func (a *App) getTask(ctx context.Context, taskID string, includeDeleted bool) (jsonMap, error) {
	rows, err := a.queryContext(ctx, `SELECT * FROM tasks WHERE id = ?`, taskID)
	if err != nil {
		return nil, err
	}
	items, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("task %s not found", taskID)}
	}
	if items[0]["deleted_at"] != nil && !includeDeleted {
		return nil, &appError{Code: 404, Msg: fmt.Sprintf("task %s deleted", taskID)}
	}
	return items[0], nil
}

func (a *App) listTasks(ctx context.Context, status, q string, includeDeleted, deletedOnly bool, limit int) ([]jsonMap, error) {
	return a.listTasksByProject(ctx, status, q, "", includeDeleted, deletedOnly, limit)
}

func (a *App) listTasksByProject(ctx context.Context, status, q, projectID string, includeDeleted, deletedOnly bool, limit int) ([]jsonMap, error) {
	query := `SELECT * FROM tasks WHERE 1=1`
	args := []any{}
	if deletedOnly {
		query += ` AND deleted_at IS NOT NULL`
	} else if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	if parts := splitCSV(status); len(parts) > 0 {
		query += ` AND status IN (` + placeholders(len(parts)) + `)`
		for _, part := range parts {
			args = append(args, part)
		}
	}
	if q != "" {
		query += ` AND (title LIKE ? ESCAPE '\' OR goal LIKE ? ESCAPE '\')`
		like := buildLikePattern(q)
		args = append(args, like, like)
	}
	if strings.TrimSpace(projectID) != "" {
		query += ` AND project_id = ?`
		args = append(args, strings.TrimSpace(projectID))
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := a.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func buildTaskSummary(item jsonMap) jsonMap {
	return jsonMap{
		"id":              item["id"],
		"task_key":        item["task_key"],
		"title":           item["title"],
		"status":          item["status"],
		"project_id":      item["project_id"],
		"summary_percent": item["summary_percent"],
		"summary_done":    item["summary_done"],
		"summary_total":   item["summary_total"],
		"summary_blocked": item["summary_blocked"],
		"updated_at":      item["updated_at"],
		"created_at":      item["created_at"],
		"version":         item["version"],
	}
}

func (a *App) softDeleteTask(ctx context.Context, taskID string) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		if _, err := a.getTask(txCtx, taskID, false); err != nil {
			return err
		}
		var refs int
		if err := a.queryRowContext(txCtx, `SELECT COUNT(*) FROM nodes WHERE linked_task_id = ? AND deleted_at IS NULL`, taskID).Scan(&refs); err != nil {
			return err
		}
		if refs > 0 {
			if _, err := a.execContext(txCtx, `UPDATE nodes
				SET linked_task_id = NULL,
				    kind = CASE WHEN kind = 'linked_task' THEN 'leaf' ELSE kind END,
				    updated_at = ?,
				    version = version + 1
				WHERE linked_task_id = ? AND deleted_at IS NULL`, utcNowISO(), taskID); err != nil {
				return err
			}
		}
		now := utcNowISO()
		if _, err := a.execContext(txCtx, `UPDATE tasks SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, taskID); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE nodes SET deleted_at = ?, updated_at = ? WHERE task_id = ? AND deleted_at IS NULL`, now, now, taskID); err != nil {
			return err
		}
		if err := a.insertEvent(txCtx, taskID, nil, "task_deleted", nil, nil, nil, nil); err != nil {
			return err
		}
		out = jsonMap{"ok": true, "unlinked_references": refs}
		return nil
	})
	return out, err
}

func (a *App) restoreTask(ctx context.Context, taskID string) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		task, err := a.getTask(txCtx, taskID, true)
		if err != nil {
			return err
		}
		if task["deleted_at"] == nil {
			out = task
			return nil
		}
		now := utcNowISO()
		projectID := strings.TrimSpace(asString(task["project_id"]))
		targetProjectID := projectID
		restorePayload := map[string]any{}
		switch {
		case projectID == "":
			defaultProject, err := a.defaultProject(txCtx)
			if err != nil {
				return err
			}
			targetProjectID = asString(defaultProject["id"])
			restorePayload["restored_to_default_project"] = true
			restorePayload["new_project_id"] = targetProjectID
		default:
			if _, err := a.getProject(txCtx, projectID, false); err != nil {
				appErr, ok := err.(*appError)
				if !ok || appErr.Code != 404 {
					return err
				}
				defaultProject, err := a.defaultProject(txCtx)
				if err != nil {
					return err
				}
				targetProjectID = asString(defaultProject["id"])
				restorePayload["restored_to_default_project"] = true
				restorePayload["previous_project_id"] = projectID
				restorePayload["new_project_id"] = targetProjectID
			}
		}
		if _, err := a.execContext(txCtx, `UPDATE tasks SET deleted_at = NULL, project_id = ?, updated_at = ? WHERE id = ?`, targetProjectID, now, taskID); err != nil {
			return err
		}
		if _, err := a.execContext(txCtx, `UPDATE nodes SET deleted_at = NULL, updated_at = ? WHERE task_id = ? AND deleted_at IS NOT NULL`, now, taskID); err != nil {
			return err
		}
		if len(restorePayload) == 0 {
			restorePayload = nil
		}
		if err := a.insertEvent(txCtx, taskID, nil, "task_restored", nil, restorePayload, nil, nil); err != nil {
			return err
		}
		if err := a.rollupTask(txCtx, taskID); err != nil {
			return err
		}
		out, err = a.getTask(txCtx, taskID, false)
		return err
	})
	return out, err
}

func (a *App) hardDeleteTask(ctx context.Context, taskID string) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		task, err := a.getTask(txCtx, taskID, true)
		if err != nil {
			return err
		}
		if task["deleted_at"] == nil {
			return &appError{Code: 409, Msg: "task must be soft-deleted first"}
		}
		var refs int
		if err := a.queryRowContext(txCtx, `SELECT COUNT(*) FROM nodes WHERE linked_task_id = ?`, taskID).Scan(&refs); err != nil {
			return err
		}
		if refs > 0 {
			return &appError{Code: 409, Msg: fmt.Sprintf("task is still referenced by %d linked_task node(s)", refs)}
		}
		rows, err := a.queryContext(txCtx, `SELECT id FROM artifacts WHERE task_id = ?`, taskID)
		if err != nil {
			return err
		}
		arts, err := scanRows(rows)
		if err != nil {
			return err
		}
		_ = osRemoveAll(filepathJoin(a.artifactRoot, taskID))
		for _, query := range []string{
			`DELETE FROM artifacts WHERE task_id = ?`,
			`DELETE FROM events WHERE task_id = ?`,
			`DELETE FROM nodes WHERE task_id = ?`,
			`DELETE FROM tasks WHERE id = ?`,
		} {
			if _, err := a.execContext(txCtx, query, taskID); err != nil {
				return err
			}
		}
		out = jsonMap{"ok": true, "artifacts_removed": len(arts)}
		return nil
	})
	return out, err
}

func (a *App) emptyTrash(ctx context.Context) (jsonMap, error) {
	var (
		out jsonMap
		err error
	)
	err = a.withTx(ctx, func(txCtx context.Context) error {
		rows, err := a.queryContext(txCtx, `SELECT id FROM tasks WHERE deleted_at IS NOT NULL`)
		if err != nil {
			return err
		}
		items, err := scanRows(rows)
		if err != nil {
			return err
		}
		deleted := 0
		skipped := 0
		for _, item := range items {
			if _, err := a.hardDeleteTask(txCtx, asString(item["id"])); err != nil {
				skipped++
			} else {
				deleted++
			}
		}
		out = jsonMap{"deleted": deleted, "skipped": skipped}
		return nil
	})
	return out, err
}

func (a *App) wrapupTask(ctx context.Context, taskID string, summary string) (jsonMap, error) {
	if _, err := a.getTask(ctx, taskID, false); err != nil {
		return nil, err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil, &appError{Code: 400, Msg: "summary 不能为空"}
	}
	if _, err := a.execContext(ctx, `UPDATE tasks SET wrapup_summary = ?, updated_at = ?, version = version + 1 WHERE id = ?`, summary, utcNowISO(), taskID); err != nil {
		return nil, err
	}
	return jsonMap{"task_id": taskID, "wrapup_summary": summary, "status": "saved"}, nil
}

func (a *App) getWrapup(ctx context.Context, taskID string) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	summary, _ := task["wrapup_summary"].(string)
	return jsonMap{"task_id": taskID, "title": task["title"], "wrapup_summary": summary}, nil
}

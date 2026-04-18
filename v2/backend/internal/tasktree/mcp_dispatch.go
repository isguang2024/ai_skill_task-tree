package tasktree

import (
	"context"
)

// mcpToolHandlers maps MCP tool names to their handlers. Entries are grouped by
// domain to make additions and audits cheaper than a monolithic switch.
var mcpToolHandlers = map[string]mcpToolHandler{}

// wrapListResult wraps a non-paginated slice into the unified list envelope so
// every MCP list_* tool returns {items, has_more, next_cursor}. Callers that
// own pagination should return their own envelope instead of calling this.
func wrapListResult(items []jsonMap) jsonMap {
	if items == nil {
		items = []jsonMap{}
	}
	return jsonMap{"items": items, "has_more": false, "next_cursor": ""}
}

// normalizeListEnvelope guarantees the {items, has_more, next_cursor} contract
// on any map that looks like a list envelope. Paginated stores may emit
// next_cursor=nil or omit has_more entirely — this keeps the MCP surface stable
// for AI consumers that expect all three keys.
func normalizeListEnvelope(m jsonMap) {
	if _, ok := m["items"]; !ok {
		return
	}
	if v, ok := m["has_more"]; !ok || v == nil {
		m["has_more"] = false
	}
	if v, ok := m["next_cursor"]; !ok || v == nil {
		m["next_cursor"] = ""
	}
}

func init() {
	registerProjectToolHandlers()
	registerTaskToolHandlers()
	registerNodeToolHandlers()
	registerStageToolHandlers()
	registerRunToolHandlers()
	registerArtifactToolHandlers()
	registerMemoryToolHandlers()
	registerMiscToolHandlers()
}

func registerProjectToolHandlers() {
	mcpToolHandlers["task_tree_list_projects"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		items, err := s.app.listProjects(ctx, stringArg(args, "q"), boolArg(args, "include_deleted"), intArg(args, "limit", 100))
		if err != nil {
			return nil, err
		}
		return wrapListResult(items), nil
	}
	mcpToolHandlers["task_tree_get_project"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getProject(ctx, stringArg(args, "project_id"), boolArg(args, "include_deleted"))
	}
	mcpToolHandlers["task_tree_create_project"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.createProject(ctx, projectCreate{
			ProjectKey:  optStringArg(args, "project_key"),
			Name:        stringArg(args, "name"),
			Description: optStringArg(args, "description"),
			IsDefault:   optBoolArg(args, "is_default"),
			Metadata:    mapArg(args, "metadata"),
		})
	}
	mcpToolHandlers["task_tree_update_project"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.updateProject(ctx, stringArg(args, "project_id"), projectUpdate{
			ProjectKey:      optStringArg(args, "project_key"),
			Name:            optStringArg(args, "name"),
			Description:     optStringArg(args, "description"),
			IsDefault:       optBoolArg(args, "is_default"),
			Metadata:        mapArg(args, "metadata"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_delete_project"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		if err := s.app.deleteProject(ctx, stringArg(args, "project_id")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	}
	mcpToolHandlers["task_tree_project_overview"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.projectOverview(ctx, stringArg(args, "project_id"), boolArg(args, "include_deleted"), intArg(args, "limit", 100))
	}
}

func registerTaskToolHandlers() {
	mcpToolHandlers["task_tree_resume"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.resumeTaskWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:          stringSliceArg(args, "status"),
			Kinds:             stringSliceArg(args, "kind"),
			Depth:             optIntArg(args, "depth"),
			MaxDepth:          optIntArg(args, "max_depth"),
			ParentNodeID:      stringArg(args, "parent_node_id"),
			SubtreeRootNodeID: stringArg(args, "subtree_root_node_id"),
			MaxRelativeDepth:  optIntArg(args, "max_relative_depth"),
			HasChildren:       optBoolArg(args, "has_children"),
			Query:             stringArg(args, "q"),
			Limit:             intArg(args, "limit", 100),
			Cursor:            stringArg(args, "cursor"),
			SortBy:            stringArg(args, "sort_by"),
			SortOrder:         stringArg(args, "sort_order"),
			ViewMode:          stringArgDefault(args, "view_mode", "summary"),
			FilterMode:        stringArg(args, "filter_mode"),
			IncludeFullTree:   boolArg(args, "include_full_tree"),
		}, eventListOptions{
			Types:     stringSliceArg(args, "event_type"),
			Query:     stringArg(args, "event_q"),
			ViewMode:  stringArg(args, "event_view_mode"),
			SortOrder: stringArg(args, "event_sort_order"),
			Limit:     intArg(args, "event_limit", 15),
			Cursor:    stringArg(args, "event_cursor"),
		}, resumeOptions{
			IncludeEvents:          hasStringArg(args, "include", "events"),
			IncludeRuns:            hasStringArg(args, "include", "runs"),
			IncludeArtifacts:       hasStringArg(args, "include", "artifacts"),
			IncludeNextNodeContext: hasStringArg(args, "include", "next_node_context"),
			IncludeTaskMemory:      hasStringArg(args, "include", "task_memory"),
			IncludeStageMemory:     hasStringArg(args, "include", "stage_memory"),
		})
	}
	mcpToolHandlers["task_tree_list_tasks"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		items, err := s.app.listTasksByProject(ctx, stringArg(args, "status"), stringArg(args, "q"), stringArg(args, "project_id"), boolArg(args, "include_deleted"), false, intArg(args, "limit", 100))
		if err != nil {
			return nil, err
		}
		summaries := make([]jsonMap, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, buildTaskSummary(item))
		}
		return wrapListResult(summaries), nil
	}
	mcpToolHandlers["task_tree_get_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getTask(ctx, stringArg(args, "task_id"), boolArg(args, "include_deleted"))
	}
	mcpToolHandlers["task_tree_create_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		stages, err := stageCreatesArg(args, "stages")
		if err != nil {
			return nil, err
		}
		seeds, err := taskNodeSeedsArg(args, "nodes")
		if err != nil {
			return nil, err
		}
		return s.app.createTask(ctx, taskCreate{
			TaskKey:         optStringArg(args, "task_key"),
			Title:           stringArg(args, "title"),
			Goal:            optStringArg(args, "goal"),
			ProjectID:       optStringArg(args, "project_id"),
			ProjectKey:      optStringArg(args, "project_key"),
			SourceTool:      optStringArg(args, "source_tool"),
			SourceSessionID: optStringArg(args, "source_session_id"),
			Tags:            stringSliceArg(args, "tags"),
			Nodes:           seeds,
			Stages:          stages,
			DryRun:          optBoolArg(args, "dry_run"),
			Metadata:        mapArg(args, "metadata"),
			CreatedByType:   optStringArg(args, "created_by_type"),
			CreatedByID:     optStringArg(args, "created_by_id"),
			CreationReason:  optStringArg(args, "creation_reason"),
		})
	}
	mcpToolHandlers["task_tree_update_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.updateTask(ctx, stringArg(args, "task_id"), taskUpdate{
			TaskKey:         optStringArg(args, "task_key"),
			Title:           optStringArg(args, "title"),
			Goal:            optStringArg(args, "goal"),
			ProjectID:       optStringArg(args, "project_id"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_delete_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.softDeleteTask(ctx, stringArg(args, "task_id"))
	}
	mcpToolHandlers["task_tree_restore_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.restoreTask(ctx, stringArg(args, "task_id"))
	}
	mcpToolHandlers["task_tree_hard_delete_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.hardDeleteTask(ctx, stringArg(args, "task_id"))
	}
	mcpToolHandlers["task_tree_empty_trash"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.emptyTrash(ctx)
	}
	mcpToolHandlers["task_tree_transition_task"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.transitionTask(ctx, stringArg(args, "task_id"), transitionBody{
			Action:          stringArg(args, "action"),
			Message:         optStringArg(args, "message"),
			Actor:           actorArg(args, "actor"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_tree_view"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.treeView(ctx, stringArg(args, "task_id"), optStringArg(args, "stage_node_id"), boolArg(args, "only_executable"))
	}
	mcpToolHandlers["task_tree_import_plan"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.importPlan(ctx, importPlanBody{
			Format: stringArg(args, "format"),
			Data:   stringArg(args, "data"),
			Apply:  optBoolArg(args, "apply"),
		})
	}
	mcpToolHandlers["task_tree_wrapup"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.wrapupTask(ctx, stringArg(args, "task_id"), stringArg(args, "summary"))
	}
	mcpToolHandlers["task_tree_get_wrapup"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getWrapup(ctx, stringArg(args, "task_id"))
	}
	mcpToolHandlers["task_tree_get_remaining"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getRemaining(ctx, stringArg(args, "task_id"))
	}
	mcpToolHandlers["task_tree_get_resume_context"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getResumeContext(ctx, stringArg(args, "task_id"), stringArg(args, "node_id"), intArg(args, "event_limit", 5))
	}
	mcpToolHandlers["task_tree_next_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.findNextNode(ctx, stringArg(args, "task_id"))
	}
}

func registerNodeToolHandlers() {
	mcpToolHandlers["task_tree_create_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.createNode(ctx, stringArg(args, "task_id"), nodeCreate{
			ParentNodeID:       optStringArg(args, "parent_node_id"),
			StageNodeID:        optStringArg(args, "stage_node_id"),
			NodeKey:            optStringArg(args, "node_key"),
			Kind:               stringArgDefault(args, "kind", "leaf"),
			Role:               optStringArg(args, "role"),
			Title:              stringArg(args, "title"),
			Instruction:        optStringArg(args, "instruction"),
			AcceptanceCriteria: stringSliceArg(args, "acceptance_criteria"),
			DependsOn:          stringSliceArg(args, "depends_on"),
			DependsOnKeys:      stringSliceArg(args, "depends_on_keys"),
			Estimate:           optFloatArg(args, "estimate"),
			Status:             optStringArg(args, "status"),
			SortOrder:          optIntArg(args, "sort_order"),
			Metadata:           mapArg(args, "metadata"),
			CreatedByType:      optStringArg(args, "created_by_type"),
			CreatedByID:        optStringArg(args, "created_by_id"),
			CreationReason:     optStringArg(args, "creation_reason"),
		})
	}
	mcpToolHandlers["task_tree_list_nodes"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodesWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:          stringSliceArg(args, "status"),
			Kinds:             stringSliceArg(args, "kind"),
			Depth:             optIntArg(args, "depth"),
			MaxDepth:          optIntArg(args, "max_depth"),
			ParentNodeID:      stringArg(args, "parent_node_id"),
			SubtreeRootNodeID: stringArg(args, "subtree_root_node_id"),
			MaxRelativeDepth:  optIntArg(args, "max_relative_depth"),
			UpdatedAfter:      stringArg(args, "updated_after"),
			HasChildren:       optBoolArg(args, "has_children"),
			Query:             stringArg(args, "q"),
			Limit:             intArg(args, "limit", 100),
			Cursor:            stringArg(args, "cursor"),
			SortBy:            stringArg(args, "sort_by"),
			SortOrder:         stringArg(args, "sort_order"),
			ViewMode:          stringArgDefault(args, "view_mode", "summary"),
			FilterMode:        stringArg(args, "filter_mode"),
			IncludeHidden:     boolArg(args, "include_deleted"),
		})
	}
	mcpToolHandlers["task_tree_list_nodes_summary"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodesWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:          stringSliceArg(args, "status"),
			Kinds:             stringSliceArg(args, "kind"),
			Depth:             optIntArg(args, "depth"),
			MaxDepth:          optIntArg(args, "max_depth"),
			ParentNodeID:      stringArg(args, "parent_node_id"),
			SubtreeRootNodeID: stringArg(args, "subtree_root_node_id"),
			MaxRelativeDepth:  optIntArg(args, "max_relative_depth"),
			Query:             stringArg(args, "q"),
			Limit:             intArg(args, "limit", 100),
			Cursor:            stringArg(args, "cursor"),
			SortBy:            stringArg(args, "sort_by"),
			SortOrder:         stringArg(args, "sort_order"),
			ViewMode:          "summary",
			FilterMode:        stringArg(args, "filter_mode"),
		})
	}
	mcpToolHandlers["task_tree_list_children"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodesWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:     stringSliceArg(args, "status"),
			ParentNodeID: stringArg(args, "node_id"),
			Limit:        intArg(args, "limit", 100),
			Cursor:       stringArg(args, "cursor"),
			SortBy:       stringArg(args, "sort_by"),
			SortOrder:    stringArg(args, "sort_order"),
			ViewMode:     "summary",
		})
	}
	mcpToolHandlers["task_tree_list_subtree_summary"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodesWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:          stringSliceArg(args, "status"),
			SubtreeRootNodeID: stringArg(args, "root_node_id"),
			MaxRelativeDepth:  optIntArg(args, "max_relative_depth"),
			Limit:             intArg(args, "limit", 100),
			Cursor:            stringArg(args, "cursor"),
			SortBy:            stringArg(args, "sort_by"),
			SortOrder:         stringArg(args, "sort_order"),
			ViewMode:          "summary",
		})
	}
	mcpToolHandlers["task_tree_focus_nodes"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodesWithOptions(ctx, stringArg(args, "task_id"), nodeListOptions{
			Statuses:   stringSliceArg(args, "status"),
			Limit:      intArg(args, "limit", 100),
			Cursor:     stringArg(args, "cursor"),
			SortBy:     stringArgDefault(args, "sort_by", "path"),
			SortOrder:  stringArgDefault(args, "sort_order", "asc"),
			ViewMode:   stringArgDefault(args, "view_mode", "summary"),
			FilterMode: "focus",
		})
	}
	mcpToolHandlers["task_tree_update_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.updateNode(ctx, stringArg(args, "node_id"), nodeUpdate{
			Title:              optStringArg(args, "title"),
			Instruction:        optStringArg(args, "instruction"),
			AcceptanceCriteria: optStringSliceArg(args, "acceptance_criteria"),
			DependsOn:          optStringSliceArg(args, "depends_on"),
			DependsOnKeys:      optStringSliceArg(args, "depends_on_keys"),
			Estimate:           optFloatArg(args, "estimate"),
			SortOrder:          optIntArg(args, "sort_order"),
			ExpectedVersion:    optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_reorder_nodes"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		ids := optStringSliceArg(args, "node_ids")
		if ids == nil {
			return nil, &appError{Code: 400, Msg: "node_ids 不能为空"}
		}
		return s.app.reorderNodes(ctx, *ids)
	}
	mcpToolHandlers["task_tree_move_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.moveNode(ctx, stringArg(args, "node_id"), moveNodeBody{
			AfterNodeID:  optStringArg(args, "after_node_id"),
			BeforeNodeID: optStringArg(args, "before_node_id"),
		})
	}
	mcpToolHandlers["task_tree_get_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		nodeID := stringArg(args, "node_id")
		if boolArg(args, "include_context") {
			return s.app.buildNodeContextWithOptions(ctx, nodeID, nodeContextOptions{
				Preset: stringArg(args, "preset"),
			})
		}
		return s.app.findNode(ctx, nodeID, boolArg(args, "include_deleted"))
	}
	mcpToolHandlers["task_tree_get_node_context"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.buildNodeContextWithOptions(ctx, stringArg(args, "node_id"), nodeContextOptions{
			Preset: stringArg(args, "preset"),
		})
	}
	mcpToolHandlers["task_tree_progress"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.reportProgress(ctx, stringArg(args, "node_id"), progressBody{
			DeltaProgress:   optFloatArg(args, "delta_progress"),
			Progress:        optFloatArg(args, "progress"),
			Message:         optStringArg(args, "message"),
			LogContent:      optStringArg(args, "log_content"),
			Actor:           actorArg(args, "actor"),
			IdempotencyKey:  optStringArg(args, "idempotency_key"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_complete"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		var inlineMemory *memoryFullPatchBody
		if memArg := mapArg(args, "memory"); memArg != nil {
			inlineMemory = &memoryFullPatchBody{
				SummaryText: optStringArg(memArg, "summary_text"),
				Conclusions: stringSliceArg(memArg, "conclusions"),
				Decisions:   stringSliceArg(memArg, "decisions"),
				Risks:       stringSliceArg(memArg, "risks"),
				Blockers:    stringSliceArg(memArg, "blockers"),
				NextActions: stringSliceArg(memArg, "next_actions"),
				Evidence:    stringSliceArg(memArg, "evidence"),
			}
		}
		return s.app.completeNode(ctx, stringArg(args, "node_id"), completeBody{
			Message:         optStringArg(args, "message"),
			Actor:           actorArg(args, "actor"),
			IdempotencyKey:  optStringArg(args, "idempotency_key"),
			ExpectedVersion: optIntArg(args, "expected_version"),
			UsageTokens:     optIntArg(args, "usage_tokens"),
			Memory:          inlineMemory,
			ResultPayload:   mapArg(args, "result_payload"),
		})
	}
	mcpToolHandlers["task_tree_block_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.blockNode(ctx, stringArg(args, "node_id"), blockBody{
			Reason:          stringArg(args, "reason"),
			Actor:           actorArg(args, "actor"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_claim"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.claimNode(ctx, stringArg(args, "node_id"), claimBody{
			Actor:           mustActorArg(args, "actor"),
			LeaseSeconds:    optIntArg(args, "lease_seconds"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_release"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.releaseNode(ctx, stringArg(args, "node_id"))
	}
	mcpToolHandlers["task_tree_retype_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.retypeNodeToLeaf(ctx, stringArg(args, "node_id"), retypeBody{
			Message:         optStringArg(args, "message"),
			Actor:           actorArg(args, "actor"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_transition_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.transitionNode(ctx, stringArg(args, "node_id"), transitionBody{
			Action:          stringArg(args, "action"),
			Message:         optStringArg(args, "message"),
			Actor:           actorArg(args, "actor"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_delete_node"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.deleteNode(ctx, stringArg(args, "node_id"))
	}
	mcpToolHandlers["task_tree_batch_create_nodes"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		nodesRaw, provided, parseErr := anyArrayArg(args, "nodes")
		if parseErr != nil {
			return nil, &appError{Code: 400, Msg: parseErr.Error()}
		}
		if !provided {
			return nil, &appError{Code: 400, Msg: "nodes required"}
		}
		bodies := make([]nodeCreate, 0, len(nodesRaw))
		for _, raw := range nodesRaw {
			m, ok := raw.(map[string]any)
			if !ok {
				return nil, &appError{Code: 400, Msg: "nodes 必须是对象数组"}
			}
			bodies = append(bodies, nodeCreate{
				ParentNodeID:       optStringArg(m, "parent_node_id"),
				StageNodeID:        optStringArg(m, "stage_node_id"),
				NodeKey:            optStringArg(m, "node_key"),
				Kind:               stringArgDefault(m, "kind", "leaf"),
				Role:               optStringArg(m, "role"),
				Title:              stringArg(m, "title"),
				Instruction:        optStringArg(m, "instruction"),
				AcceptanceCriteria: stringSliceArg(m, "acceptance_criteria"),
				DependsOn:          stringSliceArg(m, "depends_on"),
				DependsOnKeys:      stringSliceArg(m, "depends_on_keys"),
				Estimate:           optFloatArg(m, "estimate"),
				Status:             optStringArg(m, "status"),
				SortOrder:          optIntArg(m, "sort_order"),
				Metadata:           mapArg(m, "metadata"),
				CreatedByType:      optStringArg(m, "created_by_type"),
				CreatedByID:        optStringArg(m, "created_by_id"),
				CreationReason:     optStringArg(m, "creation_reason"),
			})
		}
		items, err := s.app.batchCreateNodes(ctx, stringArg(args, "task_id"), bodies)
		if err != nil {
			return nil, err
		}
		return jsonMap{"created": items, "count": len(items)}, nil
	}
}

func registerStageToolHandlers() {
	mcpToolHandlers["task_tree_list_stages"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		items, err := s.app.listStages(ctx, stringArg(args, "task_id"))
		if err != nil {
			return nil, err
		}
		summaries := make([]jsonMap, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, buildStageSummary(item))
		}
		return wrapListResult(summaries), nil
	}
	mcpToolHandlers["task_tree_create_stage"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.createStage(ctx, stringArg(args, "task_id"), stageCreate{
			NodeKey:            optStringArg(args, "node_key"),
			Title:              stringArg(args, "title"),
			Instruction:        optStringArg(args, "instruction"),
			AcceptanceCriteria: stringSliceArg(args, "acceptance_criteria"),
			Estimate:           optFloatArg(args, "estimate"),
			SortOrder:          optIntArg(args, "sort_order"),
			Metadata:           mapArg(args, "metadata"),
			Activate:           optBoolArg(args, "activate"),
			ExpectedVersion:    optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_batch_create_stages"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		stages, err := stageCreatesArg(args, "stages")
		if err != nil {
			return nil, err
		}
		items, err := s.app.batchCreateStages(ctx, stringArg(args, "task_id"), stages)
		if err != nil {
			return nil, err
		}
		return jsonMap{"created": items, "count": len(items)}, nil
	}
	mcpToolHandlers["task_tree_activate_stage"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.activateStage(ctx, stringArg(args, "task_id"), stringArg(args, "stage_node_id"), stageActivate{
			ExpectedVersion: optIntArg(args, "expected_version"),
			Message:         optStringArg(args, "message"),
			Actor:           actorArg(args, "actor"),
		})
	}
}

func registerRunToolHandlers() {
	mcpToolHandlers["task_tree_start_run"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.startRun(ctx, stringArg(args, "node_id"), runStart{
			Actor:            actorArg(args, "actor"),
			TriggerKind:      optStringArg(args, "trigger_kind"),
			InputSummary:     optStringArg(args, "input_summary"),
			OutputPreview:    optStringArg(args, "output_preview"),
			OutputRef:        optStringArg(args, "output_ref"),
			StructuredResult: mapArg(args, "structured_result"),
		})
	}
	mcpToolHandlers["task_tree_finish_run"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.finishRun(ctx, stringArg(args, "run_id"), runFinish{
			Result:           optStringArg(args, "result"),
			Status:           optStringArg(args, "status"),
			OutputPreview:    optStringArg(args, "output_preview"),
			OutputRef:        optStringArg(args, "output_ref"),
			UsageTokens:      optIntArg(args, "usage_tokens"),
			StructuredResult: mapArg(args, "structured_result"),
			ErrorText:        optStringArg(args, "error_text"),
		})
	}
	mcpToolHandlers["task_tree_get_run"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getRunWithOptions(ctx, stringArg(args, "run_id"), runListOptions{
			IncludeLogs: boolArg(args, "include_logs"),
		})
	}
	mcpToolHandlers["task_tree_list_node_runs"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listNodeRunsWithOptions(ctx, stringArg(args, "node_id"), runListOptions{
			Limit:    intArg(args, "limit", 20),
			Cursor:   stringArg(args, "cursor"),
			ViewMode: stringArg(args, "view_mode"),
		})
	}
	mcpToolHandlers["task_tree_append_run_log"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.addRunLog(ctx, stringArg(args, "run_id"), runLogCreate{
			Kind:    stringArg(args, "kind"),
			Content: optStringArg(args, "content"),
			Payload: mapArg(args, "payload"),
		})
	}
	mcpToolHandlers["task_tree_claim_and_start_run"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.claimAndStartRun(ctx, stringArg(args, "node_id"), claimStartBody{
			Actor:        mustActorArg(args, "actor"),
			LeaseSeconds: optIntArg(args, "lease_seconds"),
			InputSummary: optStringArg(args, "input_summary"),
			TriggerKind:  optStringArg(args, "trigger_kind"),
			Metadata:     mapArg(args, "metadata"),
		})
	}
}

func registerArtifactToolHandlers() {
	mcpToolHandlers["task_tree_list_artifacts"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.listArtifactsWithOptions(ctx, stringArg(args, "task_id"), optStringArg(args, "node_id"), artifactListOptions{
			Limit:    intArg(args, "limit", 100),
			Cursor:   stringArg(args, "cursor"),
			ViewMode: stringArg(args, "view_mode"),
			Kind:     stringArg(args, "kind"),
		})
	}
	mcpToolHandlers["task_tree_create_artifact"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.createArtifact(ctx, stringArg(args, "task_id"), artifactCreate{
			NodeID: optStringArg(args, "node_id"),
			Kind:   optStringArg(args, "kind"),
			Title:  optStringArg(args, "title"),
			URI:    stringArg(args, "uri"),
			Meta:   mapArg(args, "meta"),
		})
	}
	mcpToolHandlers["task_tree_upload_artifact"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.uploadArtifactBase64(ctx, stringArg(args, "task_id"), artifactUpload{
			NodeID:      optStringArg(args, "node_id"),
			Kind:        optStringArg(args, "kind"),
			Title:       optStringArg(args, "title"),
			Filename:    stringArg(args, "filename"),
			ContentBase: stringArg(args, "content_base64"),
			Meta:        mapArg(args, "meta"),
		})
	}
}

func registerMemoryToolHandlers() {
	mcpToolHandlers["task_tree_patch_node_memory"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.patchNodeMemoryFull(ctx, stringArg(args, "node_id"), memoryFullPatchBody{
			SummaryText:     optStringArg(args, "summary_text"),
			Conclusions:     stringSliceArg(args, "conclusions"),
			Decisions:       stringSliceArg(args, "decisions"),
			Risks:           stringSliceArg(args, "risks"),
			Blockers:        stringSliceArg(args, "blockers"),
			NextActions:     stringSliceArg(args, "next_actions"),
			Evidence:        stringSliceArg(args, "evidence"),
			ManualNoteText:  optStringArg(args, "manual_note_text"),
			ExpectedVersion: optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_patch_task_context"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.patchTaskContext(ctx, stringArg(args, "task_id"), taskContextPatchBody{
			ArchitectureDecisions: optStringSliceArg(args, "architecture_decisions"),
			ReferenceFiles:        optStringSliceArg(args, "reference_files"),
			ContextDocText:        optStringArg(args, "context_doc_text"),
			ExpectedVersion:       optIntArg(args, "expected_version"),
		})
	}
	mcpToolHandlers["task_tree_get_task_context"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.getTaskMemory(ctx, stringArg(args, "task_id"))
	}
}

func registerMiscToolHandlers() {
	mcpToolHandlers["task_tree_list_events"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		evLimit := intArg(args, "limit", 20)
		return s.app.listEventsScoped(ctx, stringArg(args, "task_id"), stringArg(args, "node_id"), boolArg(args, "include_descendants"), stringArg(args, "before"), stringArg(args, "after"), evLimit, eventListOptions{
			Types:     stringSliceArg(args, "type"),
			Query:     stringArg(args, "q"),
			ViewMode:  stringArg(args, "view_mode"),
			SortOrder: stringArg(args, "sort_order"),
			Limit:     evLimit,
			Cursor:    stringArg(args, "cursor"),
		})
	}
	mcpToolHandlers["task_tree_sweep_leases"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		cleared, err := s.app.sweepExpiredLeases(ctx)
		if err != nil {
			return nil, err
		}
		return jsonMap{"cleared": cleared}, nil
	}
	mcpToolHandlers["task_tree_search"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.smartSearch(ctx, stringArg(args, "q"), stringArg(args, "kind"), "", intArg(args, "limit", 30))
	}
	mcpToolHandlers["task_tree_smart_search"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		return s.app.smartSearch(ctx, stringArg(args, "q"), stringArg(args, "scope"), stringArg(args, "task_id"), intArg(args, "limit", 20))
	}
	mcpToolHandlers["task_tree_work_items"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		items, err := s.app.listWorkItems(ctx, stringArgDefault(args, "status", "ready"), boolArg(args, "include_claimed"), intArg(args, "limit", 50))
		if err != nil {
			return nil, err
		}
		summaries := make([]jsonMap, 0, len(items))
		for _, item := range items {
			summaries = append(summaries, buildWorkItemSummary(item))
		}
		return wrapListResult(summaries), nil
	}
	mcpToolHandlers["task_tree_rebuild_index"] = func(ctx context.Context, s *mcpServer, args map[string]any) (any, error) {
		if err := s.app.rebuildSearchIndex(ctx); err != nil {
			return nil, err
		}
		return jsonMap{"status": "ok", "message": "索引重建完成"}, nil
	}
}

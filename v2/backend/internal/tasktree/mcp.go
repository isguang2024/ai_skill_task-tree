package tasktree

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const mcpProtocolVersionLatest = "2025-11-25"

var mcpSupportedVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-11-25": true,
}

type mcpServer struct {
	app *App
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func runMCP(app *App, in io.Reader, out io.Writer) error {
	server := &mcpServer{app: app}
	reader := bufio.NewReader(in)
	for {
		msg, err := readRPCMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		resp := server.handle(msg)
		if resp == nil {
			continue
		}
		if err := writeRPCMessage(out, resp); err != nil {
			return err
		}
	}
}

func (s *mcpServer) handle(raw []byte) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: err.Error()}}
	}
	if req.Method == "" {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32600, Message: "missing method"}}
	}
	switch req.Method {
	case "initialize":
		negotiated := mcpProtocolVersionLatest
		var initParams struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(req.Params, &initParams); err == nil && initParams.ProtocolVersion != "" {
			if mcpSupportedVersions[initParams.ProtocolVersion] {
				negotiated = initParams.ProtocolVersion
			}
		}
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": negotiated,
				"serverInfo": map[string]any{
					"name":    "task-tree-service",
					"version": "0.1.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}
	case "notifications/initialized":
		return nil
	case "ping":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": mcpTools()}}
	case "tools/call":
		result, err := s.callTool(req.Params)
		if err != nil {
			return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"content": []map[string]any{{"type": "text", "text": err.Error()}},
				"isError": true,
			}}
		}
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	default:
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}}
	}
}

func (s *mcpServer) callTool(params json.RawMessage) (map[string]any, error) {
	var payload struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil, err
	}
	ctx := context.Background()
	var result any
	var err error
	switch payload.Name {
	case "task_tree_resume":
		result, err = s.app.resumeTaskWithOptions(ctx, stringArg(payload.Arguments, "task_id"), nodeListOptions{
			Statuses:        stringSliceArg(payload.Arguments, "status"),
			Kinds:           stringSliceArg(payload.Arguments, "kind"),
			Depth:           optIntArg(payload.Arguments, "depth"),
			MaxDepth:        optIntArg(payload.Arguments, "max_depth"),
			Query:           stringArg(payload.Arguments, "q"),
			Limit:           intArg(payload.Arguments, "limit", 100),
			Cursor:          stringArg(payload.Arguments, "cursor"),
			SortBy:          stringArg(payload.Arguments, "sort_by"),
			SortOrder:       stringArg(payload.Arguments, "sort_order"),
			ViewMode:        stringArgDefault(payload.Arguments, "view_mode", "summary"),
			FilterMode:      stringArg(payload.Arguments, "filter_mode"),
			IncludeFullTree: boolArg(payload.Arguments, "include_full_tree"),
		}, eventListOptions{
			Types:     stringSliceArg(payload.Arguments, "event_type"),
			Query:     stringArg(payload.Arguments, "event_q"),
			ViewMode:  stringArg(payload.Arguments, "event_view_mode"),
			SortOrder: stringArg(payload.Arguments, "event_sort_order"),
			Limit:     intArg(payload.Arguments, "event_limit", 15),
			Cursor:    stringArg(payload.Arguments, "event_cursor"),
		})
	case "task_tree_list_tasks":
		result, err = s.app.listTasksByProject(ctx, stringArg(payload.Arguments, "status"), stringArg(payload.Arguments, "q"), stringArg(payload.Arguments, "project_id"), boolArg(payload.Arguments, "include_deleted"), false, intArg(payload.Arguments, "limit", 100))
	case "task_tree_list_projects":
		result, err = s.app.listProjects(ctx, stringArg(payload.Arguments, "q"), boolArg(payload.Arguments, "include_deleted"), intArg(payload.Arguments, "limit", 100))
	case "task_tree_get_project":
		result, err = s.app.getProject(ctx, stringArg(payload.Arguments, "project_id"), boolArg(payload.Arguments, "include_deleted"))
	case "task_tree_create_project":
		result, err = s.app.createProject(ctx, projectCreate{
			ProjectKey:  optStringArg(payload.Arguments, "project_key"),
			Name:        stringArg(payload.Arguments, "name"),
			Description: optStringArg(payload.Arguments, "description"),
			IsDefault:   optBoolArg(payload.Arguments, "is_default"),
			Metadata:    mapArg(payload.Arguments, "metadata"),
		})
	case "task_tree_update_project":
		result, err = s.app.updateProject(ctx, stringArg(payload.Arguments, "project_id"), projectUpdate{
			ProjectKey:      optStringArg(payload.Arguments, "project_key"),
			Name:            optStringArg(payload.Arguments, "name"),
			Description:     optStringArg(payload.Arguments, "description"),
			IsDefault:       optBoolArg(payload.Arguments, "is_default"),
			Metadata:        mapArg(payload.Arguments, "metadata"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_delete_project":
		err = s.app.deleteProject(ctx, stringArg(payload.Arguments, "project_id"))
		if err == nil {
			result = map[string]any{"ok": true}
		}
	case "task_tree_project_overview":
		result, err = s.app.projectOverview(ctx, stringArg(payload.Arguments, "project_id"), boolArg(payload.Arguments, "include_deleted"), intArg(payload.Arguments, "limit", 100))
	case "task_tree_get_task":
		result, err = s.app.getTask(ctx, stringArg(payload.Arguments, "task_id"), boolArg(payload.Arguments, "include_deleted"))
	case "task_tree_create_task":
		result, err = s.app.createTask(ctx, taskCreate{
			TaskKey:         optStringArg(payload.Arguments, "task_key"),
			Title:           stringArg(payload.Arguments, "title"),
			Goal:            optStringArg(payload.Arguments, "goal"),
			ProjectID:       optStringArg(payload.Arguments, "project_id"),
			ProjectKey:      optStringArg(payload.Arguments, "project_key"),
			SourceTool:      optStringArg(payload.Arguments, "source_tool"),
			SourceSessionID: optStringArg(payload.Arguments, "source_session_id"),
			Tags:            stringSliceArg(payload.Arguments, "tags"),
			Nodes:           taskNodeSeedsArg(payload.Arguments, "nodes"),
			Metadata:        mapArg(payload.Arguments, "metadata"),
			CreatedByType:   optStringArg(payload.Arguments, "created_by_type"),
			CreatedByID:     optStringArg(payload.Arguments, "created_by_id"),
			CreationReason:  optStringArg(payload.Arguments, "creation_reason"),
		})
	case "task_tree_update_task":
		result, err = s.app.updateTask(ctx, stringArg(payload.Arguments, "task_id"), taskUpdate{
			TaskKey:         optStringArg(payload.Arguments, "task_key"),
			Title:           optStringArg(payload.Arguments, "title"),
			Goal:            optStringArg(payload.Arguments, "goal"),
			ProjectID:       optStringArg(payload.Arguments, "project_id"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_create_node":
		result, err = s.app.createNode(ctx, stringArg(payload.Arguments, "task_id"), nodeCreate{
			ParentNodeID:       optStringArg(payload.Arguments, "parent_node_id"),
			StageNodeID:        optStringArg(payload.Arguments, "stage_node_id"),
			NodeKey:            optStringArg(payload.Arguments, "node_key"),
			Kind:               stringArgDefault(payload.Arguments, "kind", "leaf"),
			Role:               optStringArg(payload.Arguments, "role"),
			Title:              stringArg(payload.Arguments, "title"),
			Instruction:        optStringArg(payload.Arguments, "instruction"),
			AcceptanceCriteria: stringSliceArg(payload.Arguments, "acceptance_criteria"),
			Estimate:           optFloatArg(payload.Arguments, "estimate"),
			Status:             optStringArg(payload.Arguments, "status"),
			SortOrder:          optIntArg(payload.Arguments, "sort_order"),
			Metadata:           mapArg(payload.Arguments, "metadata"),
			CreatedByType:      optStringArg(payload.Arguments, "created_by_type"),
			CreatedByID:        optStringArg(payload.Arguments, "created_by_id"),
			CreationReason:     optStringArg(payload.Arguments, "creation_reason"),
		})
	case "task_tree_list_stages":
		result, err = s.app.listStages(ctx, stringArg(payload.Arguments, "task_id"))
	case "task_tree_create_stage":
		result, err = s.app.createStage(ctx, stringArg(payload.Arguments, "task_id"), stageCreate{
			NodeKey:            optStringArg(payload.Arguments, "node_key"),
			Title:              stringArg(payload.Arguments, "title"),
			Instruction:        optStringArg(payload.Arguments, "instruction"),
			AcceptanceCriteria: stringSliceArg(payload.Arguments, "acceptance_criteria"),
			Estimate:           optFloatArg(payload.Arguments, "estimate"),
			SortOrder:          optIntArg(payload.Arguments, "sort_order"),
			Metadata:           mapArg(payload.Arguments, "metadata"),
			Activate:           optBoolArg(payload.Arguments, "activate"),
			ExpectedVersion:    optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_activate_stage":
		result, err = s.app.activateStage(ctx, stringArg(payload.Arguments, "task_id"), stringArg(payload.Arguments, "stage_node_id"), stageActivate{
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
		})
	case "task_tree_start_run":
		result, err = s.app.startRun(ctx, stringArg(payload.Arguments, "node_id"), runStart{
			Actor:            actorArg(payload.Arguments, "actor"),
			TriggerKind:      optStringArg(payload.Arguments, "trigger_kind"),
			InputSummary:     optStringArg(payload.Arguments, "input_summary"),
			OutputPreview:    optStringArg(payload.Arguments, "output_preview"),
			OutputRef:        optStringArg(payload.Arguments, "output_ref"),
			StructuredResult: mapArg(payload.Arguments, "structured_result"),
		})
	case "task_tree_finish_run":
		result, err = s.app.finishRun(ctx, stringArg(payload.Arguments, "run_id"), runFinish{
			Result:           optStringArg(payload.Arguments, "result"),
			Status:           optStringArg(payload.Arguments, "status"),
			OutputPreview:    optStringArg(payload.Arguments, "output_preview"),
			OutputRef:        optStringArg(payload.Arguments, "output_ref"),
			StructuredResult: mapArg(payload.Arguments, "structured_result"),
			ErrorText:        optStringArg(payload.Arguments, "error_text"),
		})
	case "task_tree_get_run":
		result, err = s.app.getRun(ctx, stringArg(payload.Arguments, "run_id"))
	case "task_tree_list_node_runs":
		result, err = s.app.listNodeRuns(ctx, stringArg(payload.Arguments, "node_id"), intArg(payload.Arguments, "limit", 20))
	case "task_tree_append_run_log":
		result, err = s.app.addRunLog(ctx, stringArg(payload.Arguments, "run_id"), runLogCreate{
			Kind:    stringArg(payload.Arguments, "kind"),
			Content: optStringArg(payload.Arguments, "content"),
			Payload: mapArg(payload.Arguments, "payload"),
		})
	case "task_tree_get_node_context":
		result, err = s.app.buildNodeContext(ctx, stringArg(payload.Arguments, "node_id"))
	case "task_tree_list_nodes":
		result, err = s.app.listNodesWithOptions(ctx, stringArg(payload.Arguments, "task_id"), nodeListOptions{
			Statuses:      stringSliceArg(payload.Arguments, "status"),
			Kinds:         stringSliceArg(payload.Arguments, "kind"),
			Depth:         optIntArg(payload.Arguments, "depth"),
			MaxDepth:      optIntArg(payload.Arguments, "max_depth"),
			UpdatedAfter:  stringArg(payload.Arguments, "updated_after"),
			HasChildren:   optBoolArg(payload.Arguments, "has_children"),
			Query:         stringArg(payload.Arguments, "q"),
			Limit:         intArg(payload.Arguments, "limit", 100),
			Cursor:        stringArg(payload.Arguments, "cursor"),
			SortBy:        stringArg(payload.Arguments, "sort_by"),
			SortOrder:     stringArg(payload.Arguments, "sort_order"),
			ViewMode:      stringArgDefault(payload.Arguments, "view_mode", "detail"),
			FilterMode:    stringArg(payload.Arguments, "filter_mode"),
			IncludeHidden: boolArg(payload.Arguments, "include_deleted"),
		})
	case "task_tree_list_nodes_summary":
		result, err = s.app.listNodesWithOptions(ctx, stringArg(payload.Arguments, "task_id"), nodeListOptions{
			Statuses:   stringSliceArg(payload.Arguments, "status"),
			Kinds:      stringSliceArg(payload.Arguments, "kind"),
			Depth:      optIntArg(payload.Arguments, "depth"),
			MaxDepth:   optIntArg(payload.Arguments, "max_depth"),
			Query:      stringArg(payload.Arguments, "q"),
			Limit:      intArg(payload.Arguments, "limit", 100),
			Cursor:     stringArg(payload.Arguments, "cursor"),
			SortBy:     stringArg(payload.Arguments, "sort_by"),
			SortOrder:  stringArg(payload.Arguments, "sort_order"),
			ViewMode:   "summary",
			FilterMode: stringArg(payload.Arguments, "filter_mode"),
		})
	case "task_tree_focus_nodes":
		result, err = s.app.listNodesWithOptions(ctx, stringArg(payload.Arguments, "task_id"), nodeListOptions{
			Statuses:   stringSliceArg(payload.Arguments, "status"),
			Limit:      intArg(payload.Arguments, "limit", 100),
			Cursor:     stringArg(payload.Arguments, "cursor"),
			SortBy:     stringArgDefault(payload.Arguments, "sort_by", "path"),
			SortOrder:  stringArgDefault(payload.Arguments, "sort_order", "asc"),
			ViewMode:   stringArgDefault(payload.Arguments, "view_mode", "summary"),
			FilterMode: "focus",
		})
	case "task_tree_update_node":
		criteria := optStringSliceArg(payload.Arguments, "acceptance_criteria")
		result, err = s.app.updateNode(ctx, stringArg(payload.Arguments, "node_id"), nodeUpdate{
			Title:              optStringArg(payload.Arguments, "title"),
			Instruction:        optStringArg(payload.Arguments, "instruction"),
			AcceptanceCriteria: criteria,
			Estimate:           optFloatArg(payload.Arguments, "estimate"),
			SortOrder:          optIntArg(payload.Arguments, "sort_order"),
			ExpectedVersion:    optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_reorder_nodes":
		ids := optStringSliceArg(payload.Arguments, "node_ids")
		if ids == nil {
			err = &appError{Code: 400, Msg: "node_ids 不能为空"}
		} else {
			var items []jsonMap
			items, err = s.app.reorderNodes(ctx, *ids)
			result = items
		}
	case "task_tree_move_node":
		result, err = s.app.moveNode(ctx, stringArg(payload.Arguments, "node_id"), moveNodeBody{
			AfterNodeID:  optStringArg(payload.Arguments, "after_node_id"),
			BeforeNodeID: optStringArg(payload.Arguments, "before_node_id"),
		})
	case "task_tree_get_node":
		result, err = s.app.findNode(ctx, stringArg(payload.Arguments, "node_id"), boolArg(payload.Arguments, "include_deleted"))
	case "task_tree_progress":
		result, err = s.app.reportProgress(ctx, stringArg(payload.Arguments, "node_id"), progressBody{
			DeltaProgress:   optFloatArg(payload.Arguments, "delta_progress"),
			Progress:        optFloatArg(payload.Arguments, "progress"),
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
			IdempotencyKey:  optStringArg(payload.Arguments, "idempotency_key"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_complete":
		result, err = s.app.completeNode(ctx, stringArg(payload.Arguments, "node_id"), completeBody{
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
			IdempotencyKey:  optStringArg(payload.Arguments, "idempotency_key"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_block_node":
		result, err = s.app.blockNode(ctx, stringArg(payload.Arguments, "node_id"), blockBody{
			Reason:          stringArg(payload.Arguments, "reason"),
			Actor:           actorArg(payload.Arguments, "actor"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_claim":
		result, err = s.app.claimNode(ctx, stringArg(payload.Arguments, "node_id"), claimBody{
			Actor:           mustActorArg(payload.Arguments, "actor"),
			LeaseSeconds:    optIntArg(payload.Arguments, "lease_seconds"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_release":
		result, err = s.app.releaseNode(ctx, stringArg(payload.Arguments, "node_id"))
	case "task_tree_retype_node":
		result, err = s.app.retypeNodeToLeaf(ctx, stringArg(payload.Arguments, "node_id"), retypeBody{
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_transition_task":
		result, err = s.app.transitionTask(ctx, stringArg(payload.Arguments, "task_id"), transitionBody{
			Action:          stringArg(payload.Arguments, "action"),
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_transition_node":
		result, err = s.app.transitionNode(ctx, stringArg(payload.Arguments, "node_id"), transitionBody{
			Action:          stringArg(payload.Arguments, "action"),
			Message:         optStringArg(payload.Arguments, "message"),
			Actor:           actorArg(payload.Arguments, "actor"),
			ExpectedVersion: optIntArg(payload.Arguments, "expected_version"),
		})
	case "task_tree_get_remaining":
		result, err = s.app.getRemaining(ctx, stringArg(payload.Arguments, "task_id"))
	case "task_tree_get_resume_context":
		result, err = s.app.getResumeContext(ctx, stringArg(payload.Arguments, "task_id"), stringArg(payload.Arguments, "node_id"), intArg(payload.Arguments, "event_limit", 10))
	case "task_tree_list_events":
		result, err = s.app.listEventsScoped(ctx, stringArg(payload.Arguments, "task_id"), stringArg(payload.Arguments, "node_id"), boolArg(payload.Arguments, "include_descendants"), stringArg(payload.Arguments, "before"), stringArg(payload.Arguments, "after"), intArg(payload.Arguments, "limit", 100), eventListOptions{
			Types:     stringSliceArg(payload.Arguments, "type"),
			Query:     stringArg(payload.Arguments, "q"),
			ViewMode:  stringArg(payload.Arguments, "view_mode"),
			SortOrder: stringArg(payload.Arguments, "sort_order"),
			Limit:     intArg(payload.Arguments, "limit", 100),
			Cursor:    stringArg(payload.Arguments, "cursor"),
		})
	case "task_tree_list_artifacts":
		nodeID := optStringArg(payload.Arguments, "node_id")
		result, err = s.app.listArtifacts(ctx, stringArg(payload.Arguments, "task_id"), nodeID)
	case "task_tree_create_artifact":
		result, err = s.app.createArtifact(ctx, stringArg(payload.Arguments, "task_id"), artifactCreate{
			NodeID: optStringArg(payload.Arguments, "node_id"),
			Kind:   optStringArg(payload.Arguments, "kind"),
			Title:  optStringArg(payload.Arguments, "title"),
			URI:    stringArg(payload.Arguments, "uri"),
			Meta:   mapArg(payload.Arguments, "meta"),
		})
	case "task_tree_upload_artifact":
		result, err = s.app.uploadArtifactBase64(ctx, stringArg(payload.Arguments, "task_id"), artifactUpload{
			NodeID:      optStringArg(payload.Arguments, "node_id"),
			Kind:        optStringArg(payload.Arguments, "kind"),
			Title:       optStringArg(payload.Arguments, "title"),
			Filename:    stringArg(payload.Arguments, "filename"),
			ContentBase: stringArg(payload.Arguments, "content_base64"),
			Meta:        mapArg(payload.Arguments, "meta"),
		})
	case "task_tree_delete_task":
		result, err = s.app.softDeleteTask(ctx, stringArg(payload.Arguments, "task_id"))
	case "task_tree_restore_task":
		result, err = s.app.restoreTask(ctx, stringArg(payload.Arguments, "task_id"))
	case "task_tree_hard_delete_task":
		result, err = s.app.hardDeleteTask(ctx, stringArg(payload.Arguments, "task_id"))
	case "task_tree_empty_trash":
		result, err = s.app.emptyTrash(ctx)
	case "task_tree_sweep_leases":
		var cleared int64
		cleared, err = s.app.sweepExpiredLeases(ctx)
		result = jsonMap{"cleared": cleared}
	case "task_tree_search":
		result, err = s.app.search(ctx, stringArg(payload.Arguments, "q"), stringArg(payload.Arguments, "kind"), intArg(payload.Arguments, "limit", 30))
	case "task_tree_work_items":
		result, err = s.app.listWorkItems(ctx, stringArgDefault(payload.Arguments, "status", "ready"), boolArg(payload.Arguments, "include_claimed"), intArg(payload.Arguments, "limit", 50))
	default:
		return nil, fmt.Errorf("unknown tool: %s", payload.Name)
	}
	if err != nil {
		return nil, err
	}
	text, _ := json.MarshalIndent(result, "", "  ")
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(text)},
		},
	}, nil
}

func mcpTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "task_tree_resume",
			Description: "根据 task_id 返回完整 resume 包与下一个可执行节点。",
			InputSchema: objectSchema(map[string]any{
				"task_id":           stringSchema("任务 ID"),
				"status":            arrayStringSchema("节点状态过滤"),
				"kind":              arrayStringSchema("节点类型过滤"),
				"depth":             intSchema("仅返回指定深度节点"),
				"max_depth":         intSchema("仅返回不超过该深度节点"),
				"q":                 stringSchema("节点关键字搜索"),
				"filter_mode":       stringSchema("all/focus/active/blocked/done"),
				"view_mode":         stringSchema("summary/detail/events"),
				"sort_by":           stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":        stringSchema("asc/desc"),
				"cursor":            stringSchema("tree 分页游标"),
				"limit":             intSchema("tree 返回数量"),
				"event_type":        arrayStringSchema("事件类型过滤"),
				"event_q":           stringSchema("事件关键字搜索"),
				"event_view_mode":   stringSchema("事件视图 summary/detail"),
				"event_sort_order":  stringSchema("事件排序 asc/desc"),
				"event_cursor":      stringSchema("事件分页游标"),
				"event_limit":       intSchema("事件返回数量"),
				"include_full_tree": boolSchema("是否返回 full_tree（默认 false，节省内存）"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_list_tasks",
			Description: "列出任务，可按状态或关键词过滤。",
			InputSchema: objectSchema(map[string]any{
				"status":          stringSchema("可选，逗号分隔状态"),
				"q":               stringSchema("可选，搜索关键词"),
				"project_id":      stringSchema("可选，按项目过滤"),
				"include_deleted": boolSchema("是否包含已删除任务"),
				"limit":           intSchema("返回数量上限"),
			}, nil),
		},
		{
			Name:        "task_tree_list_projects",
			Description: "列出项目，可按关键词过滤。",
			InputSchema: objectSchema(map[string]any{
				"q":               stringSchema("可选，搜索关键词"),
				"include_deleted": boolSchema("是否包含已删除项目"),
				"limit":           intSchema("返回数量上限"),
			}, nil),
		},
		{
			Name:        "task_tree_get_project",
			Description: "读取单个项目详情。",
			InputSchema: objectSchema(map[string]any{
				"project_id":      stringSchema("项目 ID"),
				"include_deleted": boolSchema("是否允许读取已删除项目"),
			}, []string{"project_id"}),
		},
		{
			Name:        "task_tree_create_project",
			Description: "创建项目，可指定是否为默认项目。",
			InputSchema: objectSchema(map[string]any{
				"name":        stringSchema("项目名称"),
				"project_key": stringSchema("项目短 key"),
				"description": stringSchema("项目描述"),
				"is_default":  boolSchema("是否设为默认项目"),
				"metadata":    mapSchema("扩展元数据"),
			}, []string{"name"}),
		},
		{
			Name:        "task_tree_update_project",
			Description: "更新项目名称、key、描述或默认项状态。",
			InputSchema: objectSchema(map[string]any{
				"project_id":  stringSchema("项目 ID"),
				"project_key": stringSchema("项目短 key"),
				"name":        stringSchema("项目名称"),
				"description": stringSchema("项目描述"),
				"is_default":  boolSchema("是否设为默认项目"),
				"metadata":    mapSchema("扩展元数据"),
			}, []string{"project_id"}),
		},
		{
			Name:        "task_tree_delete_project",
			Description: "删除项目。",
			InputSchema: objectSchema(map[string]any{
				"project_id": stringSchema("项目 ID"),
			}, []string{"project_id"}),
		},
		{
			Name:        "task_tree_project_overview",
			Description: "读取项目摘要与项目下任务列表。",
			InputSchema: objectSchema(map[string]any{
				"project_id":      stringSchema("项目 ID"),
				"include_deleted": boolSchema("是否包含已删除任务"),
				"limit":           intSchema("返回任务上限"),
			}, []string{"project_id"}),
		},
		{
			Name:        "task_tree_get_task",
			Description: "获取单个任务详情。",
			InputSchema: objectSchema(map[string]any{
				"task_id":         stringSchema("任务 ID"),
				"include_deleted": boolSchema("是否允许读取已删除任务"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_create_task",
			Description: "创建任务；可选一次性附带初始节点树，保留后续逐个补节点的方式。",
			InputSchema: taskCreateSchema(),
		},
		{
			Name:        "task_tree_update_task",
			Description: "更新任务标题、task_key 或 goal。",
			InputSchema: objectSchema(map[string]any{
				"task_id":    stringSchema("任务 ID"),
				"task_key":   stringSchema("任务短 key"),
				"title":      stringSchema("任务标题"),
				"goal":       stringSchema("任务目标"),
				"project_id": stringSchema("所属项目 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_create_node",
			Description: "在指定任务下创建节点。",
			InputSchema: objectSchema(map[string]any{
				"task_id":             stringSchema("任务 ID"),
				"title":               stringSchema("节点标题"),
				"parent_node_id":      stringSchema("父节点 ID"),
				"stage_node_id":       stringSchema("所属阶段节点 ID"),
				"node_key":            stringSchema("节点 key"),
				"kind":                stringSchema("leaf/group"),
				"role":                stringSchema("step/container/stage"),
				"instruction":         stringSchema("执行说明"),
				"acceptance_criteria": arrayStringSchema("验收标准"),
				"estimate":            numberSchema("预计工时"),
				"status":              stringSchema("节点状态"),
				"sort_order":          intSchema("排序"),
				"metadata":            mapSchema("扩展元数据"),
				"created_by_type":     stringSchema("创建者类型"),
				"created_by_id":       stringSchema("创建者 ID"),
				"creation_reason":     stringSchema("创建原因"),
			}, []string{"task_id", "title"}),
		},
		{
			Name:        "task_tree_list_stages",
			Description: "列出任务下的所有阶段节点。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_create_stage",
			Description: "在任务下创建阶段节点。",
			InputSchema: objectSchema(map[string]any{
				"task_id":             stringSchema("任务 ID"),
				"title":               stringSchema("阶段标题"),
				"node_key":            stringSchema("阶段 key"),
				"instruction":         stringSchema("执行说明"),
				"acceptance_criteria": arrayStringSchema("验收标准"),
				"estimate":            numberSchema("预计工时"),
				"sort_order":          intSchema("排序序号"),
				"metadata":            mapSchema("扩展元数据"),
				"activate":            boolSchema("是否创建后立即激活"),
				"expected_version":    intSchema("预期版本"),
			}, []string{"task_id", "title"}),
		},
		{
			Name:        "task_tree_activate_stage",
			Description: "切换任务当前激活阶段。",
			InputSchema: objectSchema(map[string]any{
				"task_id":          stringSchema("任务 ID"),
				"stage_node_id":    stringSchema("阶段节点 ID"),
				"expected_version": intSchema("预期版本"),
				"message":          stringSchema("切换说明"),
				"actor":            actorSchema(),
			}, []string{"task_id", "stage_node_id"}),
		},
		{
			Name:        "task_tree_start_run",
			Description: "为节点创建一次新的执行 Run。",
			InputSchema: objectSchema(map[string]any{
				"node_id":           stringSchema("节点 ID"),
				"actor":             actorSchema(),
				"trigger_kind":      stringSchema("触发类型"),
				"input_summary":     stringSchema("输入摘要"),
				"output_preview":    stringSchema("输出摘要"),
				"output_ref":        stringSchema("输出引用"),
				"structured_result": mapSchema("结构化结果"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_finish_run",
			Description: "结束一次运行中的 Run。",
			InputSchema: objectSchema(map[string]any{
				"run_id":            stringSchema("Run ID"),
				"result":            stringSchema("结果"),
				"status":            stringSchema("Run 状态"),
				"output_preview":    stringSchema("输出摘要"),
				"output_ref":        stringSchema("输出引用"),
				"structured_result": mapSchema("结构化结果"),
				"error_text":        stringSchema("错误信息"),
			}, []string{"run_id"}),
		},
		{
			Name:        "task_tree_get_run",
			Description: "读取单个 Run 详情及其日志。",
			InputSchema: objectSchema(map[string]any{
				"run_id": stringSchema("Run ID"),
			}, []string{"run_id"}),
		},
		{
			Name:        "task_tree_list_node_runs",
			Description: "列出某个节点下的 Run 历史。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
				"limit":   intSchema("返回数量"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_append_run_log",
			Description: "为指定 Run 追加一条日志。",
			InputSchema: objectSchema(map[string]any{
				"run_id":  stringSchema("Run ID"),
				"kind":    stringSchema("日志类型"),
				"content": stringSchema("日志内容"),
				"payload": mapSchema("扩展负载"),
			}, []string{"run_id", "kind"}),
		},
		{
			Name:        "task_tree_get_node_context",
			Description: "读取节点上下文聚合视图，包含 memory、runs、artifacts 等。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_list_nodes",
			Description: "列出某个任务下的节点，支持筛选、排序、分页和视图模式。",
			InputSchema: objectSchema(map[string]any{
				"task_id":         stringSchema("任务 ID"),
				"status":          arrayStringSchema("按状态过滤"),
				"kind":            arrayStringSchema("按 kind 过滤"),
				"depth":           intSchema("仅返回指定深度"),
				"max_depth":       intSchema("仅返回不超过该深度"),
				"updated_after":   stringSchema("仅返回更新时间晚于该值"),
				"has_children":    boolSchema("按是否有子节点过滤"),
				"q":               stringSchema("标题/路径/说明关键字"),
				"filter_mode":     stringSchema("all/focus/active/blocked/done"),
				"view_mode":       stringSchema("summary/detail/events"),
				"sort_by":         stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":      stringSchema("asc/desc"),
				"cursor":          stringSchema("分页游标"),
				"limit":           intSchema("返回数量"),
				"include_deleted": boolSchema("是否包含已删除节点"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_list_nodes_summary",
			Description: "轻量节点读取：仅返回高价值摘要字段。",
			InputSchema: objectSchema(map[string]any{
				"task_id":     stringSchema("任务 ID"),
				"status":      arrayStringSchema("按状态过滤"),
				"kind":        arrayStringSchema("按 kind 过滤"),
				"depth":       intSchema("仅返回指定深度"),
				"max_depth":   intSchema("仅返回不超过该深度"),
				"q":           stringSchema("标题/路径关键字"),
				"filter_mode": stringSchema("all/focus/active/blocked/done"),
				"sort_by":     stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":  stringSchema("asc/desc"),
				"cursor":      stringSchema("分页游标"),
				"limit":       intSchema("返回数量"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_focus_nodes",
			Description: "聚焦读取：仅返回可执行节点及其祖先链。",
			InputSchema: objectSchema(map[string]any{
				"task_id":    stringSchema("任务 ID"),
				"status":     arrayStringSchema("可执行状态，默认 ready/running"),
				"view_mode":  stringSchema("summary/detail/events"),
				"sort_by":    stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order": stringSchema("asc/desc"),
				"cursor":     stringSchema("分页游标"),
				"limit":      intSchema("返回数量"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_update_node",
			Description: "更新节点标题、instruction、验收标准、estimate 或 sort_order。",
			InputSchema: objectSchema(map[string]any{
				"node_id":             stringSchema("节点 ID"),
				"title":               stringSchema("节点标题"),
				"instruction":         stringSchema("执行说明"),
				"acceptance_criteria": arrayStringSchema("验收标准"),
				"estimate":            numberSchema("预计工时"),
				"sort_order":          intSchema("排序序号"),
				"expected_version":    intSchema("预期版本"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_reorder_nodes",
			Description: "批量重排同级节点顺序。传入按目标顺序排列的 node_id 数组，所有节点必须是同一父节点下的兄弟节点。",
			InputSchema: objectSchema(map[string]any{
				"node_ids": arrayStringSchema("按目标顺序排列的节点 ID 数组"),
			}, []string{"node_ids"}),
		},
		{
			Name:        "task_tree_move_node",
			Description: "移动单个节点到同级中的指定位置。指定 after_node_id 放在某节点之后，或 before_node_id 放在某节点之前；都不指定则移到最前面。",
			InputSchema: objectSchema(map[string]any{
				"node_id":        stringSchema("要移动的节点 ID"),
				"after_node_id":  stringSchema("放在此节点之后"),
				"before_node_id": stringSchema("放在此节点之前"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_get_node",
			Description: "获取单个节点详情。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"include_deleted": boolSchema("是否允许读取已删除节点"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_progress",
			Description: "上报节点进度。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"delta_progress":  numberSchema("增量进度"),
				"progress":        numberSchema("绝对进度"),
				"message":         stringSchema("进度说明"),
				"idempotency_key": stringSchema("幂等 key"),
				"actor":           actorSchema(),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_complete",
			Description: "完成节点。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"message":         stringSchema("完成说明"),
				"idempotency_key": stringSchema("幂等 key"),
				"actor":           actorSchema(),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_block_node",
			Description: "将节点标记为阻塞。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
				"reason":  stringSchema("阻塞原因"),
				"actor":   actorSchema(),
			}, []string{"node_id", "reason"}),
		},
		{
			Name:        "task_tree_claim",
			Description: "领取节点 lease。",
			InputSchema: objectSchema(map[string]any{
				"node_id":       stringSchema("节点 ID"),
				"lease_seconds": intSchema("租约秒数"),
				"actor":         actorSchema(),
			}, []string{"node_id", "actor"}),
		},
		{
			Name:        "task_tree_release",
			Description: "释放节点 lease。",
			InputSchema: objectSchema(map[string]any{"node_id": stringSchema("节点 ID")}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_retype_node",
			Description: "把无子节点的 group 节点转回 leaf 执行节点。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
				"message": stringSchema("可选说明"),
				"actor":   actorSchema(),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_transition_task",
			Description: "批量流转任务状态，支持 pause、reopen、cancel。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"action":  stringSchema("pause/reopen/cancel"),
				"message": stringSchema("可选说明"),
				"actor":   actorSchema(),
			}, []string{"task_id", "action"}),
		},
		{
			Name:        "task_tree_transition_node",
			Description: "流转叶子节点状态，支持 pause、reopen、cancel、unblock。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
				"action":  stringSchema("pause/reopen/cancel/unblock"),
				"message": stringSchema("可选说明"),
				"actor":   actorSchema(),
			}, []string{"node_id", "action"}),
		},
		{
			Name:        "task_tree_get_remaining",
			Description: "读取任务剩余节点、阻塞数、暂停数和剩余估时。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_get_resume_context",
			Description: "读取某个节点的 resume 上下文，包括近期事件。",
			InputSchema: objectSchema(map[string]any{
				"task_id":     stringSchema("任务 ID"),
				"node_id":     stringSchema("节点 ID"),
				"event_limit": intSchema("事件数量上限"),
			}, []string{"task_id", "node_id"}),
		},
		{
			Name:        "task_tree_list_events",
			Description: "列出任务或节点事件流。",
			InputSchema: objectSchema(map[string]any{
				"task_id":             stringSchema("任务 ID"),
				"node_id":             stringSchema("节点 ID"),
				"include_descendants": boolSchema("父节点是否包含全部后代节点事件"),
				"type":                arrayStringSchema("事件类型过滤"),
				"q":                   stringSchema("类型/消息关键字"),
				"view_mode":           stringSchema("summary/detail"),
				"sort_order":          stringSchema("asc/desc"),
				"cursor":              stringSchema("分页游标"),
				"before":              stringSchema("before 时间戳"),
				"after":               stringSchema("after 时间戳"),
				"limit":               intSchema("返回数量"),
			}, nil),
		},
		{
			Name:        "task_tree_list_artifacts",
			Description: "列出任务或节点的产物。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"node_id": stringSchema("节点 ID，可选"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_create_artifact",
			Description: "创建链接型产物记录。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"node_id": stringSchema("节点 ID，可选"),
				"kind":    stringSchema("link/file/..."),
				"title":   stringSchema("产物标题"),
				"uri":     stringSchema("产物 URI"),
				"meta":    mapSchema("产物元数据"),
			}, []string{"task_id", "uri"}),
		},
		{
			Name:        "task_tree_upload_artifact",
			Description: "以 base64 内容上传文件产物，便于 MCP 客户端调用。",
			InputSchema: objectSchema(map[string]any{
				"task_id":        stringSchema("任务 ID"),
				"node_id":        stringSchema("节点 ID，可选"),
				"kind":           stringSchema("产物 kind"),
				"title":          stringSchema("产物标题"),
				"filename":       stringSchema("原始文件名"),
				"content_base64": stringSchema("文件 base64 内容"),
				"meta":           mapSchema("产物元数据"),
			}, []string{"task_id", "filename", "content_base64"}),
		},
		{
			Name:        "task_tree_delete_task",
			Description: "将任务移入回收站。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_restore_task",
			Description: "恢复回收站中的任务。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_hard_delete_task",
			Description: "彻底删除已进入回收站的任务。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_empty_trash",
			Description: "清空回收站。",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
		{
			Name:        "task_tree_sweep_leases",
			Description: "清理过期 lease。",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
		{
			Name:        "task_tree_search",
			Description: "搜索任务和节点。",
			InputSchema: objectSchema(map[string]any{
				"q":     stringSchema("搜索关键词"),
				"kind":  stringSchema("all/tasks/nodes"),
				"limit": intSchema("返回数量"),
			}, []string{"q"}),
		},
		{
			Name:        "task_tree_work_items",
			Description: "获取当前可执行 work items。",
			InputSchema: objectSchema(map[string]any{
				"status":          stringSchema("ready/running/..."),
				"include_claimed": boolSchema("是否包含已 claim 项"),
				"limit":           intSchema("返回数量"),
			}, nil),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	out := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func taskCreateSchema() map[string]any {
	schema := objectSchema(map[string]any{
		"title":             stringSchema("任务标题"),
		"goal":              stringSchema("任务目标"),
		"task_key":          stringSchema("任务短 key"),
		"project_id":        stringSchema("所属项目 ID"),
		"project_key":       stringSchema("所属项目 key"),
		"source_tool":       stringSchema("来源工具"),
		"source_session_id": stringSchema("来源会话"),
		"tags":              arrayStringSchema("标签"),
		"nodes": map[string]any{
			"type":        "array",
			"description": "初始节点树；每个节点可继续带 children 递归细分。",
			"items": map[string]any{
				"$ref": "#/$defs/task_node_seed",
			},
		},
		"metadata":        mapSchema("扩展元数据"),
		"created_by_type": stringSchema("创建者类型"),
		"created_by_id":   stringSchema("创建者 ID"),
		"creation_reason": stringSchema("创建原因"),
	}, []string{"title"})
	schema["$defs"] = map[string]any{
		"task_node_seed": taskNodeSeedSchema(),
	}
	return schema
}

func taskNodeSeedSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":               stringSchema("节点标题"),
			"node_key":            stringSchema("节点 key"),
			"kind":                stringSchema("leaf/group；未填且带 children 时按 group 处理"),
			"instruction":         stringSchema("执行说明"),
			"acceptance_criteria": arrayStringSchema("验收标准"),
			"estimate":            numberSchema("预计工时"),
			"status":              stringSchema("节点状态"),
			"sort_order":          intSchema("排序"),
			"metadata":            mapSchema("扩展元数据"),
			"created_by_type":     stringSchema("创建者类型"),
			"created_by_id":       stringSchema("创建者 ID"),
			"creation_reason":     stringSchema("创建原因"),
			"children": map[string]any{
				"type":        "array",
				"description": "子节点数组；结构与当前节点相同，可递归继续细分。",
				"items": map[string]any{
					"$ref": "#/$defs/task_node_seed",
				},
			},
		},
		"required": []string{"title"},
	}
}

func stringSchema(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func intSchema(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
func numberSchema(desc string) map[string]any {
	return map[string]any{"type": "number", "description": desc}
}
func boolSchema(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func mapSchema(desc string) map[string]any {
	return map[string]any{"type": "object", "description": desc}
}
func arrayStringSchema(desc string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": "string"}}
}
func actorSchema() map[string]any {
	return objectSchema(map[string]any{
		"tool":     stringSchema("actor tool"),
		"agent_id": stringSchema("actor id"),
	}, []string{"tool", "agent_id"})
}

func readRPCMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &contentLength)
			}
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("invalid Content-Length")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeRPCMessage(w io.Writer, resp *rpcResponse) error {
	body, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func stringArg(args map[string]any, key string) string { return stringArgDefault(args, key, "") }
func stringArgDefault(args map[string]any, key, fallback string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return fallback
}
func optStringArg(args map[string]any, key string) *string {
	if v, ok := args[key].(string); ok {
		return &v
	}
	return nil
}
func optStringSliceArg(args map[string]any, key string) *[]string {
	if raw, ok := args[key].([]any); ok {
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return &out
	}
	if raw, ok := args[key].([]string); ok {
		out := append([]string(nil), raw...)
		return &out
	}
	return nil
}
func boolArg(args map[string]any, key string) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}
func optBoolArg(args map[string]any, key string) *bool {
	if v, ok := args[key].(bool); ok {
		return &v
	}
	return nil
}
func intArg(args map[string]any, key string, fallback int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return fallback
}
func optIntArg(args map[string]any, key string) *int {
	if v, ok := args[key].(float64); ok {
		i := int(v)
		return &i
	}
	return nil
}
func optFloatArg(args map[string]any, key string) *float64 {
	if v, ok := args[key].(float64); ok {
		return &v
	}
	return nil
}
func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
func mapArg(args map[string]any, key string) map[string]any {
	if v, ok := args[key].(map[string]any); ok {
		return v
	}
	return nil
}
func actorArg(args map[string]any, key string) *actor {
	v, ok := args[key].(map[string]any)
	if !ok {
		return nil
	}
	out := actor{}
	if tool, ok := v["tool"].(string); ok {
		out.Tool = &tool
	}
	if agentID, ok := v["agent_id"].(string); ok {
		out.AgentID = &agentID
	}
	return &out
}
func mustActorArg(args map[string]any, key string) actor {
	if a := actorArg(args, key); a != nil {
		return *a
	}
	tool := "mcp"
	agentID := "client"
	return actor{Tool: &tool, AgentID: &agentID}
}

func taskNodeSeedsArg(args map[string]any, key string) []taskNodeSeed {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out []taskNodeSeed
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil
	}
	return out
}

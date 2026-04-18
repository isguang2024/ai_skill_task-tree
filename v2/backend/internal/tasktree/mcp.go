package tasktree

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const mcpCallTimeout = 60 * time.Second

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

type mcpToolHandler func(ctx context.Context, s *mcpServer, args map[string]any) (any, error)

func (s *mcpServer) callTool(params json.RawMessage) (map[string]any, error) {
	var payload struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &payload); err != nil {
		return nil, err
	}
	payload.Name = canonicalToolName(payload.Name)
	handler, ok := mcpToolHandlers[payload.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", payload.Name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), mcpCallTimeout)
	defer cancel()
	result, err := handler(ctx, s, payload.Arguments)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(jsonMap); ok {
		normalizeListEnvelope(m)
		omitEmpty(m)
	}
	if items, ok := result.([]jsonMap); ok {
		omitEmptySlice(items)
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
			Description: "根据 task_id 返回轻量 resume 包与推荐下一步；重上下文需通过 include 显式附带。",
			InputSchema: objectSchema(map[string]any{
				"task_id":              stringSchema("任务 ID"),
				"include":              arrayStringSchema("按需附带 events/runs/artifacts/next_node_context/task_memory/stage_memory"),
				"status":               arrayStringSchema("节点状态过滤"),
				"kind":                 arrayStringSchema("节点类型过滤"),
				"depth":                intSchema("仅返回指定深度节点"),
				"max_depth":            intSchema("仅返回不超过该深度节点"),
				"parent_node_id":       stringSchema("仅返回某个父节点的直接 children"),
				"subtree_root_node_id": stringSchema("仅返回某个子树根节点及其后代"),
				"max_relative_depth":   intSchema("配合 subtree_root_node_id 使用，限制相对子树根的深度"),
				"has_children":         boolSchema("按是否有子节点过滤"),
				"q":                    stringSchema("节点关键字搜索"),
				"filter_mode":          stringSchema("all/focus/active/blocked/done"),
				"view_mode":            stringSchema("slim/summary/detail/events"),
				"sort_by":              stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":           stringSchema("asc/desc"),
				"cursor":               stringSchema("tree 分页游标"),
				"limit":                intSchema("tree 返回数量"),
				"event_type":           arrayStringSchema("事件类型过滤"),
				"event_q":              stringSchema("事件关键字搜索"),
				"event_view_mode":      stringSchema("事件视图 summary/detail"),
				"event_sort_order":     stringSchema("事件排序 asc/desc"),
				"event_cursor":         stringSchema("事件分页游标"),
				"event_limit":          intSchema("事件返回数量"),
				"include_full_tree":    boolSchema("是否返回 full_tree（默认 false，节省内存）"),
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
				"depends_on":          arrayStringSchema("前置依赖节点 ID 列表，这些节点完成后此节点才可执行"),
				"depends_on_keys":     arrayStringSchema("前置依赖节点 key 列表（创建时推荐）"),
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
			Name:        "task_tree_batch_create_stages",
			Description: "批量创建阶段节点（原子事务）。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"stages": arrayOrJSONStringSchema(map[string]any{
					"type":        "array",
					"description": "阶段数组",
					"items": objectSchema(map[string]any{
						"title":               stringSchema("阶段标题"),
						"node_key":            stringSchema("阶段 key"),
						"instruction":         stringSchema("执行说明"),
						"acceptance_criteria": arrayStringSchema("验收标准"),
						"estimate":            numberSchema("预计工时"),
						"sort_order":          intSchema("排序序号"),
						"metadata":            mapSchema("扩展元数据"),
						"activate":            boolSchema("是否创建后立即激活"),
						"expected_version":    intSchema("预期版本"),
					}, []string{"title"}),
				}, "阶段数组"),
			}, []string{"task_id", "stages"}),
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
				"usage_tokens":      intSchema("本次运行用量（tokens）"),
				"structured_result": mapSchema("结构化结果"),
				"error_text":        stringSchema("错误信息"),
			}, []string{"run_id"}),
		},
		{
			Name:        "task_tree_get_run",
			Description: "读取单个 Run 详情及其日志。",
			InputSchema: objectSchema(map[string]any{
				"run_id":       stringSchema("Run ID"),
				"include_logs": boolSchema("是否返回日志，默认 false"),
			}, []string{"run_id"}),
		},
		{
			Name:        "task_tree_list_node_runs",
			Description: "列出某个节点下的 Run 历史。",
			InputSchema: objectSchema(map[string]any{
				"node_id":   stringSchema("节点 ID"),
				"view_mode": stringSchema("summary/detail"),
				"cursor":    stringSchema("分页游标"),
				"limit":     intSchema("返回数量"),
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
				"preset":  stringSchema("上下文预设：summary/work/memory/full"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_list_nodes",
			Description: "列出某个任务下的节点，支持筛选、排序、分页和视图模式。",
			InputSchema: objectSchema(map[string]any{
				"task_id":              stringSchema("任务 ID"),
				"status":               arrayStringSchema("按状态过滤"),
				"kind":                 arrayStringSchema("按 kind 过滤"),
				"depth":                intSchema("仅返回指定深度"),
				"max_depth":            intSchema("仅返回不超过该深度"),
				"parent_node_id":       stringSchema("仅返回某个父节点的直接 children"),
				"subtree_root_node_id": stringSchema("仅返回某个子树根节点及其后代"),
				"max_relative_depth":   intSchema("配合 subtree_root_node_id 使用，限制相对子树根的深度"),
				"updated_after":        stringSchema("仅返回更新时间晚于该值"),
				"has_children":         boolSchema("按是否有子节点过滤"),
				"q":                    stringSchema("标题/路径/说明关键字"),
				"filter_mode":          stringSchema("all/focus/active/blocked/done"),
				"view_mode":            stringSchema("slim/summary/detail/events"),
				"sort_by":              stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":           stringSchema("asc/desc"),
				"cursor":               stringSchema("分页游标"),
				"limit":                intSchema("返回数量"),
				"include_deleted":      boolSchema("是否包含已删除节点"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_list_nodes_summary",
			Description: "[已废弃] 请改用 task_tree_list_nodes(view_mode: 'summary')。本工具保留向后兼容。",
			InputSchema: objectSchema(map[string]any{
				"task_id":              stringSchema("任务 ID"),
				"status":               arrayStringSchema("按状态过滤"),
				"kind":                 arrayStringSchema("按 kind 过滤"),
				"depth":                intSchema("仅返回指定深度"),
				"max_depth":            intSchema("仅返回不超过该深度"),
				"parent_node_id":       stringSchema("仅返回某个父节点的直接 children"),
				"subtree_root_node_id": stringSchema("仅返回某个子树根节点及其后代"),
				"max_relative_depth":   intSchema("配合 subtree_root_node_id 使用，限制相对子树根的深度"),
				"q":                    stringSchema("标题/路径关键字"),
				"filter_mode":          stringSchema("all/focus/active/blocked/done"),
				"sort_by":              stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":           stringSchema("asc/desc"),
				"cursor":               stringSchema("分页游标"),
				"limit":                intSchema("返回数量"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_list_children",
			Description: "显式读取某个父节点的直接 children，默认只返回 summary 字段。",
			InputSchema: objectSchema(map[string]any{
				"task_id":    stringSchema("任务 ID"),
				"node_id":    stringSchema("父节点 ID"),
				"status":     arrayStringSchema("按状态过滤"),
				"sort_by":    stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order": stringSchema("asc/desc"),
				"cursor":     stringSchema("分页游标"),
				"limit":      intSchema("返回数量"),
			}, []string{"task_id", "node_id"}),
		},
		{
			Name:        "task_tree_list_subtree_summary",
			Description: "显式读取某个子树根及其后代摘要，可限制相对子树根的深度。",
			InputSchema: objectSchema(map[string]any{
				"task_id":            stringSchema("任务 ID"),
				"root_node_id":       stringSchema("子树根节点 ID"),
				"status":             arrayStringSchema("按状态过滤"),
				"max_relative_depth": intSchema("限制相对子树根的深度"),
				"sort_by":            stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order":         stringSchema("asc/desc"),
				"cursor":             stringSchema("分页游标"),
				"limit":              intSchema("返回数量"),
			}, []string{"task_id", "root_node_id"}),
		},
		{
			Name:        "task_tree_focus_nodes",
			Description: "聚焦读取：仅返回可执行节点及其祖先链。",
			InputSchema: objectSchema(map[string]any{
				"task_id":    stringSchema("任务 ID"),
				"status":     arrayStringSchema("可执行状态，默认 ready/running"),
				"view_mode":  stringSchema("slim/summary/detail/events"),
				"sort_by":    stringSchema("path/updated_at/created_at/status/progress"),
				"sort_order": stringSchema("asc/desc"),
				"cursor":     stringSchema("分页游标"),
				"limit":      intSchema("返回数量"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_update_node",
			Description: "更新节点标题、instruction、验收标准、depends_on、estimate 或 sort_order。",
			InputSchema: objectSchema(map[string]any{
				"node_id":             stringSchema("节点 ID"),
				"title":               stringSchema("节点标题"),
				"instruction":         stringSchema("执行说明"),
				"acceptance_criteria": arrayStringSchema("验收标准"),
				"depends_on":          arrayStringSchema("前置依赖节点 ID 列表"),
				"depends_on_keys":     arrayStringSchema("前置依赖节点 key 列表"),
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
			Description: "获取单个节点详情。传 include_context=true 可一步获取完整上下文（含 Memory、Runs、祖先链），省去单独调用 get_node_context。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"include_deleted": boolSchema("是否允许读取已删除节点"),
				"include_context": boolSchema("是否包含完整上下文（Memory、Runs、祖先链等）"),
				"preset":          stringSchema("上下文预设：summary/work/memory/full"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_progress",
			Description: "上报节点进度。可选内联日志（省去单独 append_run_log 调用）。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"delta_progress":  numberSchema("增量进度"),
				"progress":        numberSchema("绝对进度"),
				"message":         stringSchema("进度说明（简短）"),
				"log_content":     stringSchema("可选：详细日志内容，自动追加到活跃 Run（省去 append_run_log）"),
				"idempotency_key": stringSchema("幂等 key"),
				"actor":           actorSchema(),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_complete",
			Description: "完成节点。支持内联 Memory 写入（省去单独 patch_memory 调用），自动返回下一个可执行节点（省去单独 next_node 调用）。响应中 next 字段包含推荐的下一步。",
			InputSchema: objectSchema(map[string]any{
				"node_id":         stringSchema("节点 ID"),
				"message":         stringSchema("完成说明"),
				"usage_tokens":    intSchema("本节点本次完成用量（tokens）"),
				"memory":          mapSchema("可选：内联写入 Memory（含 summary_text, conclusions, decisions, risks, evidence 等），省去单独调用 patch_node_memory。注意：execution_log 已改为系统自动从 run_logs 聚合，无需手写"),
				"result_payload":  mapSchema("结构化执行结果，如 files_created/files_modified/commands_verified/notes"),
				"idempotency_key": stringSchema("幂等 key"),
				"actor":           actorSchema(),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_block_node",
			Description: "[已废弃] 请改用 task_tree_transition_node(action: 'block', message: reason)。本工具保留向后兼容。",
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
			Description: "流转节点状态。leaf 支持 block/pause/reopen/cancel/unblock；对 group 调 cancel 会级联取消所有子节点。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
				"action":  stringSchema("block/pause/reopen/cancel/unblock"),
				"message": stringSchema("可选说明"),
				"actor":   actorSchema(),
			}, []string{"node_id", "action"}),
		},
		{
			Name:        "task_tree_delete_node",
			Description: "删除节点及其所有子节点（软删除）。不能删除正在执行中的节点。",
			InputSchema: objectSchema(map[string]any{
				"node_id": stringSchema("节点 ID"),
			}, []string{"node_id"}),
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
				"task_id":   stringSchema("任务 ID"),
				"node_id":   stringSchema("节点 ID，可选"),
				"kind":      stringSchema("按 kind 过滤"),
				"view_mode": stringSchema("summary/detail"),
				"cursor":    stringSchema("分页游标"),
				"limit":     intSchema("返回数量"),
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
			Description: "[已废弃] 请改用 task_tree_smart_search。本工具已内部转发到 smart_search。",
			InputSchema: objectSchema(map[string]any{
				"q":     stringSchema("搜索关键词"),
				"kind":  stringSchema("all/tasks/nodes"),
				"limit": intSchema("返回数量"),
			}, []string{"q"}),
		},
		{
			Name:        "task_tree_smart_search",
			Description: "全文检索任务、节点和 Memory（FTS5 + BM25 排名）。搜索范围包括节点标题、instruction、Memory 摘要、run logs 等。推荐用于查找相关上下文和历史执行记录。",
			InputSchema: objectSchema(map[string]any{
				"q":       stringSchema("搜索关键词（支持多词 AND、引号精确匹配）"),
				"scope":   stringSchema("搜索范围：all/task/node/memory（默认 all）"),
				"task_id": stringSchema("限定在某个任务内搜索"),
				"limit":   intSchema("返回数量（默认 20，最大 100）"),
			}, []string{"q"}),
		},
		{
			Name:        "task_tree_batch_create_nodes",
			Description: "批量创建节点（N 个 create_node 合并为 1 次调用）。每个节点支持与 create_node 相同的参数。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"nodes": arrayOrJSONStringSchema(map[string]any{
					"type":        "array",
					"description": "节点数组，每个元素与 create_node 参数相同",
					"items": objectSchema(map[string]any{
						"title":               stringSchema("节点标题"),
						"parent_node_id":      stringSchema("父节点 ID"),
						"stage_node_id":       stringSchema("所属阶段节点 ID"),
						"node_key":            stringSchema("节点 key"),
						"kind":                stringSchema("leaf/group"),
						"role":                stringSchema("step/container/stage"),
						"instruction":         stringSchema("执行说明"),
						"acceptance_criteria": arrayStringSchema("验收标准"),
						"depends_on":          arrayStringSchema("前置依赖节点 ID"),
						"depends_on_keys":     arrayStringSchema("前置依赖节点 key"),
						"estimate":            numberSchema("预计工时"),
						"status":              stringSchema("节点状态"),
						"sort_order":          intSchema("排序"),
						"metadata":            mapSchema("扩展元数据"),
					}, []string{"title"}),
				}, "节点数组，每个元素与 create_node 参数相同"),
			}, []string{"task_id", "nodes"}),
		},
		{
			Name:        "task_tree_tree_view",
			Description: "输出任务树文本视图（缩进树），便于快速核对层级与依赖。",
			InputSchema: objectSchema(map[string]any{
				"task_id":         stringSchema("任务 ID"),
				"stage_node_id":   stringSchema("可选：仅显示某个阶段下的节点"),
				"only_executable": boolSchema("可选：仅显示可执行叶子与祖先链"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_import_plan",
			Description: "导入 Markdown/YAML 任务计划，支持 dry-run 与 apply。",
			InputSchema: objectSchema(map[string]any{
				"format": stringSchema("markdown/yaml/json；默认 markdown"),
				"data":   stringSchema("导入内容"),
				"apply":  boolSchema("是否直接落库（默认 false）"),
			}, []string{"data"}),
		},
		{
			Name:        "task_tree_patch_task_context",
			Description: "更新任务上下文快照字段（architecture_decisions/reference_files/context_doc_text）。",
			InputSchema: objectSchema(map[string]any{
				"task_id":                stringSchema("任务 ID"),
				"architecture_decisions": arrayStringSchema("架构决策列表"),
				"reference_files":        arrayStringSchema("执行参考文件列表"),
				"context_doc_text":       stringSchema("上下文文档文本"),
				"expected_version":       intSchema("预期版本"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_get_task_context",
			Description: "读取任务上下文快照字段。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree.batch_create_nodes",
			Description: "短别名：等价于 task_tree_batch_create_nodes。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"nodes":   map[string]any{"type": "array"},
			}, []string{"task_id", "nodes"}),
		},
		{
			Name:        "task_tree.activate_stage",
			Description: "短别名：等价于 task_tree_activate_stage。",
			InputSchema: objectSchema(map[string]any{
				"task_id":       stringSchema("任务 ID"),
				"stage_node_id": stringSchema("阶段节点 ID"),
			}, []string{"task_id", "stage_node_id"}),
		},
		{
			Name:        "task_tree.list_nodes_summary",
			Description: "短别名：等价于 task_tree_list_nodes_summary。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
		{
			Name:        "task_tree_claim_and_start_run",
			Description: "领取节点并自动创建 Run（合并 claim + start_run，2 次调用 → 1 次）。返回 node 和 run 信息。",
			InputSchema: objectSchema(map[string]any{
				"node_id":       stringSchema("节点 ID"),
				"actor":         actorSchema(),
				"lease_seconds": intSchema("租约秒数"),
				"input_summary": stringSchema("输入摘要"),
				"trigger_kind":  stringSchema("触发类型"),
				"metadata":      mapSchema("扩展元数据"),
			}, []string{"node_id", "actor"}),
		},
		{
			Name:        "task_tree_rebuild_index",
			Description: "重建全文检索索引。在数据迁移或索引异常时使用。",
			InputSchema: objectSchema(map[string]any{}, nil),
		},
		{
			Name:        "task_tree_wrapup",
			Description: "写入任务收尾总结。在任务全部完成或阶段性收尾时调用，把本次改动、影响范围、验证结果、遗留问题等写入 wrapup_summary 字段。支持覆盖更新。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
				"summary": stringSchema("收尾总结文本：包含本次改动、影响范围、验证结果、遗留问题和下次方向"),
			}, []string{"task_id", "summary"}),
		},
		{
			Name:        "task_tree_get_wrapup",
			Description: "获取任务的收尾总结。通过任务 ID 直接获取 wrapup_summary 文本。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
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
		{
			Name:        "task_tree_patch_node_memory",
			Description: "更新节点 Memory 的结构化字段。支持部分更新：只传需要修改的字段，未传的字段保持不变。注意：execution_log 已改为系统自动从 run_logs 聚合生成，无需手写。用 progress(log_content=...) 记录执行过程。",
			InputSchema: objectSchema(map[string]any{
				"node_id":          stringSchema("节点 ID"),
				"summary_text":     stringSchema("摘要：做了什么 + 量化结果"),
				"conclusions":      arrayStringSchema("结论列表：分析发现和判断依据"),
				"decisions":        arrayStringSchema("决策列表：选择了什么方案、为什么"),
				"risks":            arrayStringSchema("风险列表：已知风险和隐患"),
				"blockers":         arrayStringSchema("阻塞项列表"),
				"next_actions":     arrayStringSchema("下一步行动列表"),
				"evidence":         arrayStringSchema("证据列表：改动的文件路径、命令输出、验证结果"),
				"manual_note_text": stringSchema("人工备注"),
				"expected_version": intSchema("乐观锁版本号"),
			}, []string{"node_id"}),
		},
		{
			Name:        "task_tree_next_node",
			Description: "获取当前任务中下一个应该执行的节点。优先返回当前阶段的 running 节点，其次是 ready 节点。包含推荐动作（claim/continue）。",
			InputSchema: objectSchema(map[string]any{
				"task_id": stringSchema("任务 ID"),
			}, []string{"task_id"}),
		},
	}
}

func canonicalToolName(name string) string {
	if strings.HasPrefix(name, "task_tree.") {
		return "task_tree_" + strings.ReplaceAll(strings.TrimPrefix(name, "task_tree."), ".", "_")
	}
	return name
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

func oneOfSchema(schemas ...map[string]any) map[string]any {
	items := make([]any, 0, len(schemas))
	for _, schema := range schemas {
		items = append(items, schema)
	}
	return map[string]any{"oneOf": items}
}

func taskCreateSchema() map[string]any {
	schema := objectSchema(map[string]any{
		"title":             stringSchema("任务标题"),
		"goal":              stringSchema("任务目标"),
		"task_key":          stringSchema("任务短 key"),
		"dry_run":           boolSchema("仅校验并返回预览，不持久化"),
		"project_id":        stringSchema("所属项目 ID"),
		"project_key":       stringSchema("所属项目 key"),
		"source_tool":       stringSchema("来源工具"),
		"source_session_id": stringSchema("来源会话"),
		"tags":              arrayStringSchema("标签"),
		"stages": arrayOrJSONStringSchema(map[string]any{
			"type":        "array",
			"description": "初始阶段列表；在 nodes 之前创建，第一个阶段自动激活。",
			"items": objectSchema(map[string]any{
				"key":                 stringSchema("阶段匹配 key；节点通过 stage_key 引用此值来归入对应阶段"),
				"title":               stringSchema("阶段标题"),
				"node_key":            stringSchema("阶段节点 key（用于路径）"),
				"instruction":         stringSchema("执行说明"),
				"acceptance_criteria": arrayStringSchema("验收标准"),
				"estimate":            numberSchema("预计工时"),
				"sort_order":          intSchema("排序序号"),
				"metadata":            mapSchema("扩展元数据"),
				"activate":            boolSchema("是否创建后立即激活"),
			}, []string{"title"}),
		}, "初始阶段列表；在 nodes 之前创建，第一个阶段自动激活。"),
		"nodes": arrayOrJSONStringSchema(map[string]any{
			"type":        "array",
			"description": "初始节点树；每个节点可继续带 children 递归细分。如果有 stages，节点将归入当前激活阶段。",
			"items": map[string]any{
				"$ref": "#/$defs/task_node_seed",
			},
		}, "初始节点树；每个节点可继续带 children 递归细分。如果有 stages，节点将归入当前激活阶段。"),
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
			"stage_key":           stringSchema("目标阶段 key；对应 stages 中的 key 字段，节点将归入该阶段而非默认的激活阶段"),
			"instruction":         stringSchema("执行说明"),
			"acceptance_criteria": arrayStringSchema("验收标准"),
			"depends_on":          arrayStringSchema("前置依赖节点 ID 列表"),
			"depends_on_keys":     arrayStringSchema("前置依赖节点 key 列表"),
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
	return oneOfSchema(
		map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": "string"}},
		map[string]any{"type": "string", "description": desc + "（兼容 JSON 字符串数组）"},
	)
}

func arrayOrJSONStringSchema(arraySchema map[string]any, desc string) map[string]any {
	return oneOfSchema(
		arraySchema,
		map[string]any{"type": "string", "description": desc + "（兼容 JSON 字符串数组）"},
	)
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
	if _, exists := args[key]; !exists {
		return nil
	}
	out := stringSliceArg(args, key)
	return &out
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
	raw, provided, err := anyArrayArg(args, key)
	if err != nil || !provided {
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
func hasStringArg(args map[string]any, key, expected string) bool {
	for _, item := range stringSliceArg(args, key) {
		if strings.EqualFold(strings.TrimSpace(item), expected) {
			return true
		}
	}
	return false
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

func anyArrayArg(args map[string]any, key string) ([]any, bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, false, nil
	}
	switch v := raw.(type) {
	case []any:
		return v, true, nil
	case []string:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out, true, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return []any{}, true, nil
		}
		if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
			unquoted, err := strconv.Unquote(s)
			if err == nil {
				s = unquoted
			}
		}
		if !strings.HasPrefix(s, "[") {
			return []any{s}, true, nil
		}
		var out []any
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, true, fmt.Errorf("%s 期望为数组（array），也可传 JSON 字符串数组", key)
		}
		return out, true, nil
	default:
		return nil, true, fmt.Errorf("%s 期望为数组（array）", key)
	}
}

func taskNodeSeedsArg(args map[string]any, key string) ([]taskNodeSeed, error) {
	raw, provided, err := anyArrayArg(args, key)
	if err != nil {
		return nil, err
	}
	if !provided {
		return nil, nil
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []taskNodeSeed
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("%s 解析失败：%w", key, err)
	}
	return out, nil
}

func stageCreatesArg(args map[string]any, key string) ([]stageCreate, error) {
	raw, provided, err := anyArrayArg(args, key)
	if err != nil {
		return nil, err
	}
	if !provided {
		return nil, nil
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []stageCreate
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("%s 解析失败：%w", key, err)
	}
	return out, nil
}

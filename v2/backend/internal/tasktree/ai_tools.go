package tasktree

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// aiTool is an Anthropic tool definition.
type aiTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

func aiToolDefinitions() []aiTool {
	str := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	strEnum := func(desc string, vals ...string) map[string]any {
		return map[string]any{"type": "string", "description": desc, "enum": vals}
	}
	num := func(desc string) map[string]any {
		return map[string]any{"type": "number", "description": desc}
	}
	obj := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}
	arr := func(itemType, desc string) map[string]any {
		return map[string]any{
			"type":        "array",
			"description": desc,
			"items":       map[string]any{"type": itemType},
		}
	}

	return []aiTool{
		{
			Name:        "list_tasks",
			Description: "列出任务总览。返回所有任务的标题、状态、进度、节点数等。",
			InputSchema: obj(map[string]any{
				"status": strEnum("按状态过滤（可选）", "ready", "running", "blocked", "paused", "done"),
				"q":      str("按标题或 goal 模糊搜索（可选）"),
			}),
		},
		{
			Name:        "get_task",
			Description: "获取任务摘要。默认返回 task、remaining、recommended_action；只有 include_tree=true 时才附带节点树摘要。",
			InputSchema: obj(map[string]any{
				"task_id":      str("任务 ID，格式 tsk_xxx"),
				"include_tree": map[string]any{"type": "boolean", "description": "是否附带节点树摘要，默认 false"},
			}, "task_id"),
		},
		{
			Name:        "resume_task",
			Description: "获取任务摘要与当前聚焦树，适合作为 AI 第一步读取。",
			InputSchema: obj(map[string]any{
				"task_id": str("任务 ID，格式 tsk_xxx"),
				"limit":   num("树节点返回数量（可选）"),
			}, "task_id"),
		},
		{
			Name:        "list_nodes_summary",
			Description: "按 summary 模式列出节点，可按 focus/父节点/子树逐层展开。",
			InputSchema: obj(map[string]any{
				"task_id":              str("任务 ID"),
				"filter_mode":          str("all/focus/active/blocked/done（可选）"),
				"status":               arr("string", "状态过滤（可选）"),
				"q":                    str("关键字过滤（可选）"),
				"parent_node_id":       str("只看某个父节点的直接 children（可选）"),
				"subtree_root_node_id": str("只看某个子树根及其后代（可选）"),
				"max_relative_depth":   num("限制相对子树根的深度（可选）"),
				"limit":                num("返回数量（可选）"),
			}, "task_id"),
		},
		{
			Name:        "list_node_children",
			Description: "按父节点读取直接 children 摘要，适合逐层展开局部节点树。",
			InputSchema: obj(map[string]any{
				"task_id":    str("任务 ID"),
				"node_id":    str("父节点 ID"),
				"status":     arr("string", "状态过滤（可选）"),
				"limit":      num("返回数量（可选）"),
				"cursor":     str("分页游标（可选）"),
				"sort_by":    str("排序字段（可选）"),
				"sort_order": str("排序方向（可选）"),
			}, "task_id", "node_id"),
		},
		{
			Name:        "list_node_subtree_summary",
			Description: "按子树根读取 summary 摘要，可限制相对子树根的深度。",
			InputSchema: obj(map[string]any{
				"task_id":            str("任务 ID"),
				"root_node_id":       str("子树根节点 ID"),
				"status":             arr("string", "状态过滤（可选）"),
				"max_relative_depth": num("限制相对子树根的深度（可选）"),
				"limit":              num("返回数量（可选）"),
				"cursor":             str("分页游标（可选）"),
				"sort_by":            str("排序字段（可选）"),
				"sort_order":         str("排序方向（可选）"),
			}, "task_id", "root_node_id"),
		},
		{
			Name:        "get_node_context",
			Description: "读取节点上下文，支持 summary/work/memory/full 预设。",
			InputSchema: obj(map[string]any{
				"node_id": str("节点 ID"),
				"preset":  strEnum("上下文预设", "summary", "work", "memory", "full"),
			}, "node_id"),
		},
		{
			Name:        "claim_and_start_run",
			Description: "领取节点并自动创建运行 Run，适合开始执行。",
			InputSchema: obj(map[string]any{
				"node_id":       str("节点 ID"),
				"input_summary": str("本次执行摘要（可选）"),
			}, "node_id"),
		},
		{
			Name:        "batch_create_nodes",
			Description: "批量创建多个节点。",
			InputSchema: obj(map[string]any{
				"task_id": str("任务 ID"),
				"nodes":   arr("object", "节点对象数组，支持 title/instruction/parent_node_id/node_key/estimate"),
			}, "task_id", "nodes"),
		},
		{
			Name:        "import_plan",
			Description: "导入 markdown/yaml 计划，可 dry-run 或 apply。",
			InputSchema: obj(map[string]any{
				"format": strEnum("格式", "markdown", "yaml", "json"),
				"data":   str("计划内容"),
				"apply":  map[string]any{"type": "boolean", "description": "是否直接落库"},
			}, "data"),
		},
		{
			Name:        "smart_search",
			Description: "全文检索任务、节点和 Memory。",
			InputSchema: obj(map[string]any{
				"q":       str("搜索关键词"),
				"scope":   strEnum("搜索范围（可选）", "all", "task", "node", "memory"),
				"task_id": str("限定任务 ID（可选）"),
				"limit":   num("返回数量（可选）"),
			}, "q"),
		},
		{
			Name:        "create_task",
			Description: "创建一个新任务。",
			InputSchema: obj(map[string]any{
				"title":    str("任务标题，动词短语"),
				"goal":     str("任务目标，2-4 句，描述交付标准、约束、范围外项"),
				"task_key": str("任务 key，短标识（可选）"),
			}, "title"),
		},
		{
			Name:        "create_node",
			Description: "在指定任务下创建节点。可挂在父节点下（group）或作为根节点。",
			InputSchema: obj(map[string]any{
				"task_id":             str("任务 ID"),
				"title":               str("节点标题，动词短语"),
				"instruction":         str("具体步骤/文件/命令（可选）"),
				"acceptance_criteria": arr("string", "验收标准，每条一个字符串（可选）"),
				"parent_node_id":      str("父节点 ID，为空则为根节点（可选）"),
				"estimate":            num("估时，单位小时（可选）"),
				"node_key":            str("节点短标识（可选）"),
			}, "task_id", "title"),
		},
		{
			Name:        "update_node",
			Description: "修改节点的标题、instruction 或验收标准。",
			InputSchema: obj(map[string]any{
				"node_id":             str("节点 ID，格式 nd_xxx"),
				"title":               str("新标题（可选）"),
				"instruction":         str("新 instruction（可选）"),
				"acceptance_criteria": arr("string", "新验收标准（可选）"),
				"estimate":            num("估时，单位小时（可选）"),
			}, "node_id"),
		},
		{
			Name:        "claim_node",
			Description: "领取（开始）一个节点。节点必须处于 ready 状态。",
			InputSchema: obj(map[string]any{
				"node_id":       str("节点 ID"),
				"lease_seconds": num("租约时长（秒），默认 3600（可选）"),
			}, "node_id"),
		},
		{
			Name:        "progress_node",
			Description: "为节点上报进度（0.0–1.0 增量）并写入说明。",
			InputSchema: obj(map[string]any{
				"node_id": str("节点 ID"),
				"delta":   num("进度增量，0.0–1.0，例如 0.3"),
				"message": str("做了什么 / 证据 / 偏差 / 遗留"),
			}, "node_id", "delta"),
		},
		{
			Name:        "complete_node",
			Description: "将节点标记为完成。需要一条说明（做了什么/证据/偏差/遗留）。",
			InputSchema: obj(map[string]any{
				"node_id": str("节点 ID"),
				"message": str("完成说明（四段：做了什么/证据/偏差/遗留）"),
			}, "node_id", "message"),
		},
		{
			Name:        "transition_node",
			Description: "对节点执行状态流转：block（阻塞）、unblock（解除阻塞）、pause（暂停）、reopen（重开）、cancel（取消）。",
			InputSchema: obj(map[string]any{
				"node_id": str("节点 ID，格式 nd_xxx，从 get_task 返回的 node_id 字段获取"),
				"action":  strEnum("操作", "block", "unblock", "pause", "reopen", "cancel"),
				"message": str("原因说明（可选）"),
			}, "node_id", "action"),
		},
		{
			Name:        "delete_task",
			Description: "删除任务（软删除）。会同时软删除任务下所有节点。不可恢复，请在用户明确确认后再调用。",
			InputSchema: obj(map[string]any{
				"task_id": str("任务 ID，格式 tsk_xxx"),
				"confirm": strEnum("必须传 'yes' 才会执行删除，防止误操作", "yes"),
			}, "task_id", "confirm"),
		},
	}
}

// executeAITool dispatches a tool call to the corresponding App method.
func (a *App) executeAITool(ctx context.Context, name string, inputRaw json.RawMessage) string {
	var in map[string]any
	if err := json.Unmarshal(inputRaw, &in); err != nil {
		return "参数解析失败: " + err.Error()
	}
	str := func(key string) string { return asString(in[key]) }
	flt := func(key string, def float64) float64 {
		if v, ok := in[key]; ok {
			return asFloat(v)
		}
		return def
	}
	boolean := func(key string) bool {
		v, ok := in[key]
		if !ok {
			return false
		}
		b, ok := v.(bool)
		return ok && b
	}

	switch name {
	case "list_tasks":
		items, err := a.listTasks(ctx, str("status"), str("q"), false, false, 100)
		if err != nil {
			return "list_tasks 失败: " + err.Error()
		}
		if len(items) == 0 {
			return "当前没有任务。"
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "共 %d 个任务：\n", len(items))
		for _, t := range items {
			pct := int(asFloat(t["percent"]))
			sb.WriteString(fmt.Sprintf("• [%s] %s  状态:%s  进度:%d%%  剩余:%v 节点\n",
				asString(t["id"]), asString(t["title"]),
				asString(t["status"]), pct, t["remaining_nodes"]))
		}
		return sb.String()

	case "get_task":
		taskID := str("task_id")
		resume, err := a.resumeTaskWithOptions(ctx, taskID, nodeListOptions{ViewMode: "summary", Limit: 20}, eventListOptions{}, resumeOptions{})
		if err != nil {
			return "get_task 失败: " + err.Error()
		}
		if boolean("include_tree") {
			treeWrap, err := a.listNodesWithOptions(ctx, taskID, nodeListOptions{ViewMode: "summary", Limit: 50, SortBy: "path", SortOrder: "asc"})
			if err != nil {
				return "get_task 失败: " + err.Error()
			}
			resume["tree"] = treeWrap["items"]
			resume["tree_cursor"] = treeWrap["next_cursor"]
		} else {
			resume["tree"] = []jsonMap{}
			resume["tree_cursor"] = nil
		}
		return summarizeResumeForAI(resume)

	case "resume_task":
		taskID := str("task_id")
		limit := int(flt("limit", 20))
		resume, err := a.resumeTaskWithOptions(ctx, taskID, nodeListOptions{ViewMode: "summary", Limit: limit}, eventListOptions{}, resumeOptions{})
		if err != nil {
			return "resume_task 失败: " + err.Error()
		}
		return summarizeResumeForAI(resume)

	case "list_nodes_summary":
		taskID := str("task_id")
		opts := nodeListOptions{
			ViewMode:          "summary",
			FilterMode:        str("filter_mode"),
			Query:             str("q"),
			ParentNodeID:      str("parent_node_id"),
			SubtreeRootNodeID: str("subtree_root_node_id"),
			Limit:             int(flt("limit", 50)),
		}
		if rawStatuses, ok := in["status"].([]any); ok {
			for _, raw := range rawStatuses {
				opts.Statuses = append(opts.Statuses, asString(raw))
			}
		}
		if _, ok := in["max_relative_depth"]; ok {
			n := int(asFloat(in["max_relative_depth"]))
			opts.MaxRelativeDepth = &n
		}
		result, err := a.listNodesWithOptions(ctx, taskID, opts)
		if err != nil {
			return "list_nodes_summary 失败: " + err.Error()
		}
		return summarizeNodeItemsForAI(workspaceAsItems(result["items"]), result["next_cursor"])

	case "list_node_children":
		taskID := str("task_id")
		opts := nodeListOptions{
			ViewMode:     "summary",
			ParentNodeID: str("node_id"),
			Limit:        int(flt("limit", 50)),
			Cursor:       str("cursor"),
			SortBy:       str("sort_by"),
			SortOrder:    str("sort_order"),
		}
		if rawStatuses, ok := in["status"].([]any); ok {
			for _, raw := range rawStatuses {
				opts.Statuses = append(opts.Statuses, asString(raw))
			}
		}
		result, err := a.listNodesWithOptions(ctx, taskID, opts)
		if err != nil {
			return "list_node_children 失败: " + err.Error()
		}
		return summarizeNodeItemsForAI(workspaceAsItems(result["items"]), result["next_cursor"])

	case "list_node_subtree_summary":
		taskID := str("task_id")
		opts := nodeListOptions{
			ViewMode:          "summary",
			SubtreeRootNodeID: str("root_node_id"),
			Limit:             int(flt("limit", 50)),
			Cursor:            str("cursor"),
			SortBy:            str("sort_by"),
			SortOrder:         str("sort_order"),
		}
		if rawStatuses, ok := in["status"].([]any); ok {
			for _, raw := range rawStatuses {
				opts.Statuses = append(opts.Statuses, asString(raw))
			}
		}
		if _, ok := in["max_relative_depth"]; ok {
			n := int(asFloat(in["max_relative_depth"]))
			opts.MaxRelativeDepth = &n
		}
		result, err := a.listNodesWithOptions(ctx, taskID, opts)
		if err != nil {
			return "list_node_subtree_summary 失败: " + err.Error()
		}
		return summarizeNodeItemsForAI(workspaceAsItems(result["items"]), result["next_cursor"])

	case "get_node_context":
		result, err := a.buildNodeContextWithOptions(ctx, str("node_id"), nodeContextOptions{Preset: str("preset")})
		if err != nil {
			return "get_node_context 失败: " + err.Error()
		}
		return summarizeNodeContextForAI(result)

	case "create_task":
		goal := str("goal")
		taskKey := str("task_key")
		body := taskCreate{Title: str("title")}
		if goal != "" {
			body.Goal = &goal
		}
		if taskKey != "" {
			body.TaskKey = &taskKey
		}
		item, err := a.createTask(ctx, body)
		if err != nil {
			return "create_task 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 任务已创建：%s (%s)", asString(item["title"]), asString(item["id"]))

	case "create_node":
		taskID := str("task_id")
		instr := str("instruction")
		parentID := str("parent_node_id")
		nodeKey := str("node_key")
		est := flt("estimate", 0)
		body := nodeCreate{Title: str("title")}
		if instr != "" {
			body.Instruction = &instr
		}
		if parentID != "" {
			body.ParentNodeID = &parentID
		}
		if nodeKey != "" {
			body.NodeKey = &nodeKey
		}
		if est > 0 {
			body.Estimate = &est
		}
		if criArr, ok := in["acceptance_criteria"]; ok {
			if items, ok := criArr.([]any); ok {
				for _, item := range items {
					body.AcceptanceCriteria = append(body.AcceptanceCriteria, asString(item))
				}
			}
		}
		item, err := a.createNode(ctx, taskID, body)
		if err != nil {
			return "create_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 节点已创建：%s  路径:%s  ID:%s",
			asString(item["title"]), asString(item["path"]), asString(item["id"]))

	case "batch_create_nodes":
		taskID := str("task_id")
		rawNodes, _ := in["nodes"].([]any)
		bodies := make([]nodeCreate, 0, len(rawNodes))
		for _, raw := range rawNodes {
			item, _ := raw.(map[string]any)
			if item == nil {
				continue
			}
			body := nodeCreate{Title: asString(item["title"])}
			if v := strings.TrimSpace(asString(item["instruction"])); v != "" {
				body.Instruction = &v
			}
			if v := strings.TrimSpace(asString(item["parent_node_id"])); v != "" {
				body.ParentNodeID = &v
			}
			if v := strings.TrimSpace(asString(item["node_key"])); v != "" {
				body.NodeKey = &v
			}
			if est := asFloat(item["estimate"]); est > 0 {
				body.Estimate = &est
			}
			bodies = append(bodies, body)
		}
		items, err := a.batchCreateNodes(ctx, taskID, bodies)
		if err != nil {
			return "batch_create_nodes 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 已批量创建 %d 个节点", len(items))

	case "update_node":
		nodeID := str("node_id")
		body := nodeUpdate{}
		if v := str("title"); v != "" {
			body.Title = &v
		}
		if v := str("instruction"); v != "" {
			body.Instruction = &v
		}
		if est := flt("estimate", -1); est >= 0 {
			body.Estimate = &est
		}
		if criArr, ok := in["acceptance_criteria"]; ok {
			if items, ok := criArr.([]any); ok {
				var cri []string
				for _, item := range items {
					cri = append(cri, asString(item))
				}
				body.AcceptanceCriteria = &cri
			}
		}
		item, err := a.updateNode(ctx, nodeID, body)
		if err != nil {
			return "update_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 节点已更新：%s (%s)", asString(item["title"]), nodeID)

	case "claim_node":
		nodeID := str("node_id")
		leaseF := flt("lease_seconds", 3600)
		lease := int(leaseF)
		agentID := "ai-assistant"
		body := claimBody{
			Actor:        actor{AgentID: &agentID},
			LeaseSeconds: &lease,
		}
		item, err := a.claimNode(ctx, nodeID, body)
		if err != nil {
			return "claim_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 节点已领取：%s  状态:%s", asString(item["title"]), asString(item["status"]))

	case "claim_and_start_run":
		inputSummary := nullableOptString(str("input_summary"))
		item, err := a.claimAndStartRun(ctx, str("node_id"), claimStartBody{
			Actor:        actor{Tool: strPtr("ai"), AgentID: strPtr("builtin-ai")},
			InputSummary: inputSummary,
		})
		if err != nil {
			return "claim_and_start_run 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 已领取并启动运行：node=%s run=%s", asString(item["node_id"]), asString(item["run_id"]))

	case "progress_node":
		nodeID := str("node_id")
		delta := flt("delta", 0.1)
		msg := str("message")
		body := progressBody{DeltaProgress: &delta}
		if msg != "" {
			body.Message = &msg
		}
		item, err := a.reportProgress(ctx, nodeID, body)
		if err != nil {
			return "progress_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 进度已更新：%s  进度:%v%%", asString(item["title"]), int(asFloat(item["progress"])*100))

	case "complete_node":
		nodeID := str("node_id")
		msg := str("message")
		body := completeBody{}
		if msg != "" {
			body.Message = &msg
		}
		item, err := a.completeNode(ctx, nodeID, body)
		if err != nil {
			return "complete_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 节点已完成：%s (%s)", asString(item["title"]), nodeID)

	case "transition_node":
		nodeID := str("node_id")
		action := str("action")
		msg := str("message")
		body := transitionBody{Action: action}
		if msg != "" {
			body.Message = &msg
		}
		item, err := a.transitionNode(ctx, nodeID, body)
		if err != nil {
			return "transition_node 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 节点状态已变更：%s → %s", asString(item["title"]), asString(item["status"]))

	case "delete_task":
		if str("confirm") != "yes" {
			return "❌ 删除未执行：confirm 参数必须为 'yes'"
		}
		taskID := str("task_id")
		if _, err := a.softDeleteTask(ctx, taskID); err != nil {
			return "delete_task 失败: " + err.Error()
		}
		return fmt.Sprintf("✅ 任务已删除（软删除）：%s", taskID)

	case "import_plan":
		apply := false
		if v, ok := in["apply"].(bool); ok {
			apply = v
		}
		item, err := a.importPlan(ctx, importPlanBody{
			Format: str("format"),
			Data:   str("data"),
			Apply:  &apply,
		})
		if err != nil {
			return "import_plan 失败: " + err.Error()
		}
		return summarizeMap(item)

	case "smart_search":
		item, err := a.smartSearch(ctx, str("q"), str("scope"), str("task_id"), int(flt("limit", 10)))
		if err != nil {
			return "smart_search 失败: " + err.Error()
		}
		return summarizeMap(item)

	default:
		return "未知工具: " + name
	}
}

func nullableOptString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func summarizeResumeForAI(resume jsonMap) string {
	task := asAnyMap(resume["task"])
	tree := workspaceAsItems(resume["tree"])
	var sb strings.Builder
	fmt.Fprintf(&sb, "任务：%s (%s)\n状态：%s\n", asString(task["title"]), asString(task["id"]), asString(task["status"]))
	if remaining := asAnyMap(resume["remaining"]); remaining != nil {
		fmt.Fprintf(&sb, "剩余节点：%v  阻塞：%v  暂停：%v\n", remaining["remaining_nodes"], remaining["blocked_nodes"], remaining["paused_nodes"])
	}
	if action := asAnyMap(resume["recommended_action"]); action != nil {
		fmt.Fprintf(&sb, "推荐动作：%s  node=%s\n", asString(action["action"]), asString(action["node_id"]))
	}
	if len(tree) > 0 || asString(resume["tree_cursor"]) != "" {
		sb.WriteString("聚焦树：\n")
		sb.WriteString(summarizeNodeItemsForAI(tree, resume["tree_cursor"]))
	} else {
		sb.WriteString("节点树：未加载（需要时传 include_tree=true 或再调用 list_nodes_summary）。")
	}
	return sb.String()
}

func summarizeNodeItemsForAI(items []jsonMap, nextCursor any) string {
	if len(items) == 0 {
		return "无节点。"
	}
	var sb strings.Builder
	for _, item := range items {
		fmt.Fprintf(&sb, "- %s [%s] %s (%s)\n", asString(item["path"]), asString(item["status"]), asString(item["title"]), asString(item["id"]))
	}
	if asString(nextCursor) != "" {
		fmt.Fprintf(&sb, "next_cursor=%s\n", asString(nextCursor))
	}
	return sb.String()
}

func summarizeNodeContextForAI(ctxMap jsonMap) string {
	node := asAnyMap(ctxMap["node"])
	var sb strings.Builder
	fmt.Fprintf(&sb, "节点：%s (%s)\n路径：%s\n状态：%s\n", asString(node["title"]), asString(node["id"]), asString(node["path"]), asString(node["status"]))
	if mem := asAnyMap(ctxMap["memory"]); mem != nil && asString(mem["summary_text"]) != "" {
		fmt.Fprintf(&sb, "摘要：%s\n", asString(mem["summary_text"]))
	}
	if runs := workspaceAsItems(ctxMap["recent_runs"]); len(runs) > 0 {
		fmt.Fprintf(&sb, "最近 Run：%d 条\n", len(runs))
	}
	if events := workspaceAsItems(ctxMap["recent_events"]); len(events) > 0 {
		fmt.Fprintf(&sb, "最近事件：%d 条\n", len(events))
	}
	return sb.String()
}

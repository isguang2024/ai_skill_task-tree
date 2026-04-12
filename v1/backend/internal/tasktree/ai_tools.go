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
			Description: "获取任务详情，包含完整的节点树（path、状态、进度、instruction）。分析任务前必先调用此接口。",
			InputSchema: obj(map[string]any{
				"task_id": str("任务 ID，格式 tsk_xxx"),
			}, "task_id"),
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
		task, err := a.getTask(ctx, taskID, false)
		if err != nil {
			return "get_task 失败: " + err.Error()
		}
		nodes, err := a.listNodes(ctx, taskID)
		if err != nil {
			return "listNodes 失败: " + err.Error()
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "任务：%s (%s)\n目标：%s\n状态：%s  进度：%v%%  剩余：%v\n\n节点列表（%d 条）：\n",
			asString(task["title"]), taskID,
			asString(task["goal"]),
			asString(task["status"]), task["percent"], task["remaining_nodes"],
			len(nodes))
		for _, n := range nodes {
			instr := asString(n["instruction"])
			if len(instr) > 60 {
				instr = instr[:60] + "..."
			}
			claimed := ""
			if asString(n["claimed_by_id"]) != "" {
				claimed = "  👤" + asString(n["claimed_by_id"])
			}
			sb.WriteString(fmt.Sprintf("  node_id:%s  path:%s  [%s] %s  %v%%%s\n",
				asString(n["id"]), asString(n["path"]),
				asString(n["status"]), asString(n["title"]),
				int(asFloat(n["progress"])*100), claimed))
			if instr != "" {
				sb.WriteString(fmt.Sprintf("       → %s\n", instr))
			}
		}
		return sb.String()

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

	default:
		return "未知工具: " + name
	}
}


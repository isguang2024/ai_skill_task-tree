package tasktree

import (
	"context"
	"sort"
	"strings"
)

func (a *App) buildFocusNodes(ctx context.Context, taskID string) ([]jsonMap, jsonMap, error) {
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, nil, err
	}
	currentStageNodeID := asString(task["current_stage_node_id"])
	ordered := orderedExecutableLeaves(nodes)
	// Build ID->node map for dependency checks
	nodesByID := make(map[string]jsonMap, len(nodes))
	for _, n := range nodes {
		nodesByID[asString(n["id"])] = n
	}
	var nextNode jsonMap
	for _, node := range ordered {
		if currentStageNodeID != "" && asString(node["stage_node_id"]) != currentStageNodeID {
			continue
		}
		if asString(node["status"]) == "running" {
			nextNode = node
			break
		}
	}
	if nextNode == nil {
		for _, node := range ordered {
			if currentStageNodeID != "" && asString(node["stage_node_id"]) != currentStageNodeID {
				continue
			}
			if asString(node["status"]) == "ready" && dependsMet(node, nodesByID) {
				nextNode = node
				break
			}
		}
	}
	if nextNode == nil {
		for _, node := range ordered {
			if (asString(node["status"]) == "running" || asString(node["status"]) == "ready") && dependsMet(node, nodesByID) {
				nextNode = node
				break
			}
		}
	}
	active := focusNodeIDsFromNodes(nodes, currentStageNodeID, nil)
	items := make([]jsonMap, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := active[asString(node["id"])]; !ok {
			continue
		}
		items = append(items, buildNodeSummary(node))
	}
	sortJSONMapsByPath(items)
	return items, nextNode, nil
}

func (a *App) findNextNode(ctx context.Context, taskID string) (jsonMap, error) {
	_, nextNode, err := a.buildFocusNodes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if nextNode == nil {
		return jsonMap{
			"found":   false,
			"message": "当前没有可执行的节点（所有节点已完成或被阻塞）",
		}, nil
	}
	alternatives, err := a.buildParallelAlternatives(ctx, taskID, nextNode)
	if err != nil {
		return nil, err
	}

	nodeID := asString(nextNode["id"])
	memory, _ := a.getNodeMemoryStructured(ctx, nodeID) // no execution_log for next_node

	// 构建推荐动作
	status := asString(nextNode["status"])
	var action string
	switch status {
	case "ready":
		action = "claim"
	case "running":
		if leaseActive(nextNode) {
			action = "continue"
		} else {
			action = "claim"
		}
	default:
		action = "review"
	}

	resp := jsonMap{
		"found":  true,
		"node":   buildNodeSummary(nextNode),
		"memory": memory,
		"recommended_action": jsonMap{
			"action":       action,
			"node_id":      nodeID,
			"alternatives": alternatives,
			"hint": func() string {
				switch action {
				case "claim":
					return "调用 task_tree_claim(node_id) 领取此节点，然后开始执行"
				case "continue":
					return "此节点已被领取且 lease 有效，直接继续执行"
				default:
					return "检查节点状态后决定下一步"
				}
			}(),
		},
	}
	task, taskErr := a.getTask(ctx, taskID, false)
	if taskErr == nil {
		currentStageID := asString(task["current_stage_node_id"])
		if currentStageID != "" {
			if stage, stageErr := a.findNode(ctx, currentStageID, false); stageErr == nil && stageLooksCompleted(stage) {
				resp["pr_suggestion"] = buildPRSuggestion(task, stage)
			}
		}
	}
	return resp, nil
}

func (a *App) buildNodeContext(ctx context.Context, nodeID string) (jsonMap, error) {
	return a.buildNodeContextWithOptions(ctx, nodeID, nodeContextOptions{Preset: "full"})
}

func (a *App) buildNodeContextWithOptions(ctx context.Context, nodeID string, opts nodeContextOptions) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	preset := strings.ToLower(strings.TrimSpace(opts.Preset))
	if preset == "" {
		preset = "full"
	}
	taskID := asString(node["task_id"])
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	// Batch-fetch ancestor chain using recursive CTE (replaces N+1 findNode loop)
	ancestorChain, err := a.fetchAncestorChain(ctx, asString(node["parent_node_id"]))
	if err != nil {
		return nil, err
	}
	var ancestorMemories map[string]jsonMap
	if preset == "full" && len(ancestorChain) > 0 {
		memIDs := make([]string, 0, len(ancestorChain))
		for _, anc := range ancestorChain {
			memIDs = append(memIDs, asString(anc["id"]))
		}
		ancestorMemories, _ = a.batchGetNodeMemorySummaries(ctx, memIDs)
	}
	ancestors := make([]jsonMap, 0, len(ancestorChain))
	for _, parent := range ancestorChain {
		ancestorEntry := jsonMap{
			"node_id": parent["id"],
			"path":    parent["path"],
			"title":   parent["title"],
			"status":  parent["status"],
		}
		if preset == "full" && ancestorMemories != nil {
			if mem, ok := ancestorMemories[asString(parent["id"])]; ok {
				ancestorEntry["memory_summary"] = asString(mem["summary_text"])
				ancestorEntry["decisions"] = mem["decisions"]
				ancestorEntry["blockers"] = mem["blockers"]
			}
		}
		ancestors = append(ancestors, ancestorEntry)
	}

	rows, err := a.queryContext(ctx, `SELECT id, task_id, parent_node_id, path, title, kind, status, progress, estimate, depends_on_json, updated_at, sort_order, created_at FROM nodes WHERE task_id = ? AND COALESCE(parent_node_id, '') = ? AND deleted_at IS NULL ORDER BY sort_order, created_at`, taskID, asString(node["parent_node_id"]))
	if err != nil {
		return nil, err
	}
	siblings, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	siblingSummaries := make([]jsonMap, 0, len(siblings))
	for _, sibling := range siblings {
		if asString(sibling["id"]) == nodeID {
			continue
		}
		siblingSummaries = append(siblingSummaries, buildNodeSummary(sibling))
	}
	resp := jsonMap{
		"node":      node,
		"ancestors": ancestors,
		"siblings":  siblingSummaries,
	}
	if preset == "summary" {
		return resp, nil
	}

	// preset=work → structured memory (no execution_log); preset=memory/full → include execution_log
	var memory jsonMap
	if preset == "full" {
		memory, err = a.getNodeMemory(ctx, nodeID) // full: includes execution_log
	} else {
		memory, err = a.getNodeMemoryStructured(ctx, nodeID) // work/memory: excludes execution_log
	}
	if err != nil {
		return nil, err
	}
	resp["memory"] = memory

	var stageSummary jsonMap
	stageNodeID := asString(node["stage_node_id"])
	if stageNodeID != "" {
		stage, _ := a.findNode(ctx, stageNodeID, false)
		stageSummary = jsonMap{"stage": stage}
		if preset == "work" || preset == "memory" || preset == "full" {
			stageMemory, _ := a.getStageMemory(ctx, stageNodeID)
			stageSummary["memory"] = stageMemory
		}
	}
	resp["stage_summary"] = stageSummary

	if preset == "memory" || preset == "full" {
		taskMemory, _ := a.getTaskMemory(ctx, taskID)
		resp["task"] = task
		resp["task_memory"] = taskMemory
	}

	if preset == "work" || preset == "full" {
		runs, err := a.listNodeRuns(ctx, nodeID, 10)
		if err != nil {
			return nil, err
		}
		events, err := a.listEventsScoped(ctx, taskID, nodeID, false, "", "", 10, eventListOptions{})
		if err != nil {
			return nil, err
		}
		artifacts, err := a.listArtifacts(ctx, taskID, &nodeID)
		if err != nil {
			return nil, err
		}
		latestResultPayload := map[string]any{}
		if len(runs) > 0 {
			if structured := asAnyMap(runs[0]["structured_result"]); structured != nil {
				latestResultPayload = structured
			}
		}
		resp["recent_runs"] = runs
		resp["latest_result_payload"] = latestResultPayload
		resp["recent_events"] = events["items"]
		resp["artifacts"] = artifacts
	}
	return resp, nil
}

func (a *App) buildResumeV2(ctx context.Context, taskID string, nodeOpts nodeListOptions, eventOpts eventListOptions, resumeOpts resumeOptions) (jsonMap, error) {
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	taskMemory, err := a.getTaskMemory(ctx, taskID)
	if err != nil {
		return nil, err
	}
	var currentStage jsonMap
	var currentStageMemory jsonMap
	currentStageNodeID := asString(task["current_stage_node_id"])
	if currentStageNodeID != "" {
		currentStage, _ = a.findNode(ctx, currentStageNodeID, false)
		currentStageMemory, _ = a.getStageMemory(ctx, currentStageNodeID)
	}
	taskMemorySummary := trimTaskMemoryForResume(taskMemory)
	currentStageMemorySummary := trimStageMemoryForResume(currentStageMemory)

	useFilteredTree := nodeOpts.ViewMode != "" ||
		nodeOpts.FilterMode != "" ||
		len(nodeOpts.Statuses) > 0 ||
		len(nodeOpts.Kinds) > 0 ||
		nodeOpts.Query != "" ||
		nodeOpts.Depth != nil ||
		nodeOpts.MaxDepth != nil ||
		nodeOpts.ParentNodeID != "" ||
		nodeOpts.SubtreeRootNodeID != "" ||
		nodeOpts.MaxRelativeDepth != nil ||
		nodeOpts.HasChildren != nil ||
		nodeOpts.Limit > 0 ||
		nodeOpts.Cursor != "" ||
		nodeOpts.SortBy != "" ||
		nodeOpts.SortOrder != "" ||
		nodeOpts.IncludeHidden

	var focusNodes []jsonMap
	var nextNode jsonMap
	treeHasMore := false
	var treeCursor any

	if useFilteredTree {
		// When filtered tree is requested, skip building full focus tree.
		// Only load nodes list for nextNode computation (needed for recommended_action).
		filteredNodesWrap, err := a.listNodesWithOptions(ctx, taskID, nodeOpts)
		if err != nil {
			return nil, err
		}
		focusNodes = workspaceAsItems(filteredNodesWrap["items"])
		treeHasMore, _ = filteredNodesWrap["has_more"].(bool)
		treeCursor = filteredNodesWrap["next_cursor"]
		// Lightweight nextNode: load nodes once, find next executable in-memory
		allNodes, err := a.listNodes(ctx, taskID)
		if err != nil {
			return nil, err
		}
		nextNode = findNextExecutableFromNodes(allNodes, currentStageNodeID)
	} else {
		// Default path: build focus tree + nextNode together (single listNodes call)
		var err2 error
		focusNodes, nextNode, err2 = a.buildFocusNodes(ctx, taskID)
		if err2 != nil {
			return nil, err2
		}
	}
	remaining, err := a.getRemaining(ctx, taskID)
	if err != nil {
		return nil, err
	}
	events := jsonMap{"items": []jsonMap{}, "next_cursor": nil}
	runs := []jsonMap{}
	artifacts := []jsonMap{}
	if resumeOpts.IncludeEvents {
		eventFallback := 5
		if taskStatus := asString(task["status"]); taskStatus == "done" || taskStatus == "canceled" {
			eventFallback = 3
		}
		events, err = a.listEventsScoped(ctx, taskID, "", false, eventOpts.Before, eventOpts.After, limitWithFallback(eventOpts.Limit, eventFallback), eventOpts)
		if err != nil {
			return nil, err
		}
	}
	if resumeOpts.IncludeRuns {
		runs, err = a.listTaskRuns(ctx, taskID, 5)
		if err != nil {
			return nil, err
		}
	}
	if resumeOpts.IncludeArtifacts {
		artifacts, err = a.listArtifacts(ctx, taskID, nil)
		if err != nil {
			return nil, err
		}
	}
	debug := jsonMap{
		"used_stage":          currentStageNodeID != "",
		"focus_nodes_count":   len(focusNodes),
		"recent_runs_count":   len(runs),
		"recent_events_count": len(workspaceAsItems(events["items"])),
		"used_snapshot":       false,
		"detail_fallback":     len(focusNodes) == 0,
	}
	var nextCtx any
	if resumeOpts.IncludeNextNodeContext && nextNode != nil {
		nextCtx, err = a.getResumeContext(ctx, taskID, asString(nextNode["id"]), 10)
		if err != nil {
			return nil, err
		}
	}
	// 构建推荐动作
	var recommendedAction jsonMap
	if nextNode != nil {
		nodeStatus := asString(nextNode["status"])
		action := "claim"
		hint := "调用 task_tree_claim(node_id) 领取此节点，然后按节点要求执行实际操作"
		if nodeStatus == "running" && leaseActive(nextNode) {
			action = "continue"
			hint = "此节点已被领取且 lease 有效，直接继续执行节点要求的操作"
		}
		recommendedAction = jsonMap{
			"action":  action,
			"node_id": asString(nextNode["id"]),
			"title":   asString(nextNode["title"]),
			"hint":    hint,
		}
	} else {
		// 判断是否全部完成
		remainingCount := asFloat(remaining["total"]) - asFloat(remaining["done"]) - asFloat(remaining["canceled"])
		if remainingCount <= 0 {
			recommendedAction = jsonMap{
				"action": "all_done",
				"hint":   "所有节点已完成，可以进行任务收尾",
			}
		} else {
			recommendedAction = jsonMap{
				"action": "blocked",
				"hint":   "当前阶段没有可执行节点，所有待执行节点处于 blocked 或 paused 状态",
			}
		}
	}

	resp := jsonMap{
		"task":                  trimTaskForResume(task),
		"task_memory_summary":   taskMemorySummary,
		"current_stage":         trimStageForResume(currentStage),
		"current_stage_summary": trimStageForResume(currentStage),
		"tree":                  focusNodes,
		"tree_has_more":         treeHasMore,
		"tree_cursor":           treeCursor,
		"remaining":             remaining,
		"recent_events":         events["items"],
		"events_cursor":         events["next_cursor"],
		"recent_runs":           runs,
		"artifacts":             artifacts,
		"next_node":             nextCtx,
		"next_node_summary":     buildNodeSummary(nextNode),
		"recommended_action":    recommendedAction,
		"debug":                 debug,
	}
	if resumeOpts.IncludeTaskMemory {
		resp["task_memory"] = taskMemory
	}
	if resumeOpts.IncludeStageMemory {
		resp["current_stage_memory"] = currentStageMemory
		resp["current_stage_memory_summary"] = currentStageMemorySummary
	}
	if currentStage != nil && stageLooksCompleted(currentStage) {
		resp["pr_suggestion"] = buildPRSuggestion(task, currentStage)
	}
	if nodeOpts.IncludeFullTree {
		nodes, err := a.listNodes(ctx, taskID)
		if err != nil {
			return nil, err
		}
		tree := make([]jsonMap, 0, len(nodes))
		for _, node := range nodes {
			tree = append(tree, jsonMap{
				"node_id":        node["id"],
				"parent_node_id": node["parent_node_id"],
				"path":           node["path"],
				"title":          node["title"],
				"status":         node["status"],
				"progress":       node["progress"],
				"kind":           node["kind"],
				"role":           node["role"],
				"depth":          node["depth"],
				"estimate":       node["estimate"],
				"has_children":   node["has_children"],
				"stage_node_id":  node["stage_node_id"],
			})
		}
		resp["full_tree"] = tree
	}
	return resp, nil
}

func (a *App) buildParallelAlternatives(ctx context.Context, taskID string, nextNode jsonMap) ([]jsonMap, error) {
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	nodesByID := make(map[string]jsonMap, len(nodes))
	for _, node := range nodes {
		nodesByID[asString(node["id"])] = node
	}
	ordered := orderedExecutableLeaves(nodes)
	nextID := asString(nextNode["id"])
	nextParentID := asString(nextNode["parent_node_id"])
	nextStageID := asString(nextNode["stage_node_id"])
	parentRole := ""
	if nextParentID != "" {
		parentRole = asString(nodesByID[nextParentID]["role"])
	}

	candidates := make([]jsonMap, 0, 3)
	for _, node := range ordered {
		id := asString(node["id"])
		if id == "" || id == nextID {
			continue
		}
		status := asString(node["status"])
		if status != "ready" && status != "running" {
			continue
		}
		if !dependsMet(node, nodesByID) {
			continue
		}
		if parentRole == "parallel_group" {
			if asString(node["parent_node_id"]) != nextParentID {
				continue
			}
		} else if nextStageID != "" && asString(node["stage_node_id"]) != nextStageID {
			continue
		}
		candidates = append(candidates, jsonMap{
			"node_id":        id,
			"parent_node_id": node["parent_node_id"],
			"path":           asString(node["path"]),
			"title":          asString(node["title"]),
			"status":         status,
			"kind":           node["kind"],
			"role":           node["role"],
			"depth":          node["depth"],
		})
		if len(candidates) >= 3 {
			break
		}
	}
	return candidates, nil
}

// dependsMet checks if all dependencies of a node are in "done" status.
func dependsMet(node jsonMap, nodesByID map[string]jsonMap) bool {
	deps := stringSliceFromAny(node["depends_on"])
	if len(deps) == 0 {
		return true
	}
	for _, depID := range deps {
		dep, ok := nodesByID[depID]
		if !ok {
			continue // missing dep node, don't block
		}
		s := asString(dep["status"])
		if s != "done" && s != "canceled" {
			return false
		}
	}
	return true
}

func sortJSONMapsByPath(items []jsonMap) {
	sort.Slice(items, func(i, j int) bool {
		return naturalPathLess(asString(items[i]["path"]), asString(items[j]["path"]))
	})
}

func dirtyEnvelopeForEvent(event jsonMap, node jsonMap) jsonMap {
	dirty := []string{"events"}
	eventType := strings.TrimSpace(asString(event["type"]))
	payload := asAnyMap(event["payload"])
	var runID any
	if payload != nil {
		runID = payload["run_id"]
	}
	switch eventType {
	case "run_started", "run_finished":
		dirty = append(dirty, "runs", "node", "resume")
	case "artifact":
		dirty = append(dirty, "artifacts", "resume")
	case "progress", "complete", "blocked", "paused", "canceled", "reopened", "unblocked", "claim", "release":
		dirty = append(dirty, "node", "resume")
	case "task_paused", "task_reopened", "task_canceled":
		dirty = append(dirty, "task", "resume")
	}
	return jsonMap{
		"type":          eventType,
		"task_id":       event["task_id"],
		"node_id":       event["node_id"],
		"stage_node_id": node["stage_node_id"],
		"run_id":        runID,
		"dirty":         mergeStringSlice(nil, dirty),
		"ts":            event["created_at"],
	}
}

func trimTaskForResume(task jsonMap) jsonMap {
	if task == nil {
		return nil
	}
	return jsonMap{
		"id":                    task["id"],
		"task_key":              task["task_key"],
		"title":                 task["title"],
		"status":                task["status"],
		"project_id":            task["project_id"],
		"current_stage_node_id": task["current_stage_node_id"],
		"summary_percent":       task["summary_percent"],
		"summary_done":          task["summary_done"],
		"summary_total":         task["summary_total"],
		"summary_blocked":       task["summary_blocked"],
		"updated_at":            task["updated_at"],
		"version":               task["version"],
	}
}

func trimTaskMemoryForResume(mem jsonMap) jsonMap {
	if mem == nil {
		return nil
	}
	return jsonMap{
		"task_id":      mem["task_id"],
		"summary":      mem["summary_text"],
		"summary_text": mem["summary_text"],
		"decisions":    mem["decisions"],
		"next_actions": mem["next_actions"],
		"risks":        mem["risks"],
		"blockers":     mem["blockers"],
		"version":      mem["version"],
		"updated_at":   mem["updated_at"],
	}
}

func trimStageMemoryForResume(mem jsonMap) jsonMap {
	if mem == nil {
		return nil
	}
	return jsonMap{
		"stage_node_id": mem["stage_node_id"],
		"summary":       mem["summary_text"],
		"summary_text":  mem["summary_text"],
		"decisions":     mem["decisions"],
		"next_actions":  mem["next_actions"],
		"risks":         mem["risks"],
		"blockers":      mem["blockers"],
		"version":       mem["version"],
		"updated_at":    mem["updated_at"],
	}
}

func trimStageForResume(stage jsonMap) jsonMap {
	if stage == nil {
		return nil
	}
	return jsonMap{
		"id":          stage["id"],
		"title":       stage["title"],
		"status":      stage["status"],
		"progress":    stage["progress"],
		"node_key":    stage["node_key"],
		"instruction": stage["instruction"],
	}
}

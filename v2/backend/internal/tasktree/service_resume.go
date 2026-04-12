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
			if asString(node["status"]) == "ready" {
				nextNode = node
				break
			}
		}
	}
	if nextNode == nil {
		for _, node := range ordered {
			if asString(node["status"]) == "running" || asString(node["status"]) == "ready" {
				nextNode = node
				break
			}
		}
	}
	active := map[string]struct{}{}
	for _, node := range nodes {
		status := asString(node["status"])
		if status != "ready" && status != "running" && status != "blocked" && status != "paused" {
			continue
		}
		if asString(node["role"]) == "stage" {
			continue
		}
		if currentStageNodeID != "" && asString(node["stage_node_id"]) != currentStageNodeID {
			continue
		}
		id := asString(node["id"])
		active[id] = struct{}{}
		parentID := asString(node["parent_node_id"])
		for parentID != "" {
			active[parentID] = struct{}{}
			parent, err := a.findNode(ctx, parentID, false)
			if err != nil {
				break
			}
			parentID = asString(parent["parent_node_id"])
		}
	}
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

func (a *App) buildNodeContext(ctx context.Context, nodeID string) (jsonMap, error) {
	node, err := a.findNode(ctx, nodeID, false)
	if err != nil {
		return nil, err
	}
	taskID := asString(node["task_id"])
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	memory, err := a.getNodeMemory(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	ancestors := []jsonMap{}
	parentID := asString(node["parent_node_id"])
	for parentID != "" {
		parent, err := a.findNode(ctx, parentID, false)
		if err != nil {
			break
		}
		ancestors = append([]jsonMap{{"node_id": parent["id"], "path": parent["path"], "title": parent["title"]}}, ancestors...)
		parentID = asString(parent["parent_node_id"])
	}
	rows, err := a.queryContext(ctx, `SELECT * FROM nodes WHERE task_id = ? AND parent_node_id = ? AND deleted_at IS NULL ORDER BY sort_order, created_at`, taskID, node["parent_node_id"])
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
	var stageSummary jsonMap
	stageNodeID := asString(node["stage_node_id"])
	if stageNodeID != "" {
		stage, _ := a.findNode(ctx, stageNodeID, false)
		stageMemory, _ := a.getStageMemory(ctx, stageNodeID)
		stageSummary = jsonMap{
			"stage":  stage,
			"memory": stageMemory,
		}
	}
	return jsonMap{
		"task":          task,
		"node":          node,
		"memory":        memory,
		"ancestors":     ancestors,
		"siblings":      siblingSummaries,
		"recent_runs":   runs,
		"recent_events": events["items"],
		"artifacts":     artifacts,
		"stage_summary": stageSummary,
	}, nil
}

func (a *App) buildResumeV2(ctx context.Context, taskID string, nodeOpts nodeListOptions, eventOpts eventListOptions) (jsonMap, error) {
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
	focusNodes, nextNode, err := a.buildFocusNodes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if nodeOpts.ViewMode != "" || nodeOpts.FilterMode != "" || len(nodeOpts.Statuses) > 0 || len(nodeOpts.Kinds) > 0 || nodeOpts.Query != "" || nodeOpts.Depth != nil || nodeOpts.MaxDepth != nil {
		filteredNodesWrap, err := a.listNodesWithOptions(ctx, taskID, nodeOpts)
		if err != nil {
			return nil, err
		}
		if items := workspaceAsItems(filteredNodesWrap["items"]); len(items) > 0 {
			focusNodes = items
		}
	}
	remaining, err := a.getRemaining(ctx, taskID)
	if err != nil {
		return nil, err
	}
	events, err := a.listEventsScoped(ctx, taskID, "", false, eventOpts.Before, eventOpts.After, limitWithFallback(eventOpts.Limit, 15), eventOpts)
	if err != nil {
		return nil, err
	}
	runs, err := a.listTaskRuns(ctx, taskID, 10)
	if err != nil {
		return nil, err
	}
	artifacts, err := a.listArtifacts(ctx, taskID, nil)
	if err != nil {
		return nil, err
	}
	debug := jsonMap{
		"used_stage":        currentStageNodeID != "",
		"focus_nodes_count": len(focusNodes),
		"recent_runs_count": len(runs),
		"recent_events_count": len(workspaceAsItems(events["items"])),
		"used_snapshot":     false,
		"detail_fallback":   len(focusNodes) == 0,
	}
	var nextCtx any
	if nextNode != nil {
		nextCtx, err = a.getResumeContext(ctx, taskID, asString(nextNode["id"]), 10)
		if err != nil {
			return nil, err
		}
	}
	resp := jsonMap{
		"task":               task,
		"task_memory":        taskMemory,
		"current_stage":      currentStage,
		"current_stage_memory": currentStageMemory,
		"tree":               focusNodes,
		"tree_has_more":      false,
		"tree_cursor":        nil,
		"remaining":          remaining,
		"recent_events":      events["items"],
		"events_cursor":      events["next_cursor"],
		"recent_runs":        runs,
		"artifacts":          artifacts,
		"next_node":          nextCtx,
		"debug":              debug,
	}
	if nodeOpts.IncludeFullTree {
		nodes, err := a.listNodes(ctx, taskID)
		if err != nil {
			return nil, err
		}
		tree := make([]jsonMap, 0, len(nodes))
		for _, node := range nodes {
			tree = append(tree, jsonMap{
				"node_id":       node["id"],
				"path":          node["path"],
				"title":         node["title"],
				"status":        node["status"],
				"progress":      node["progress"],
				"kind":          node["kind"],
				"stage_node_id": node["stage_node_id"],
			})
		}
		resp["full_tree"] = tree
	}
	return resp, nil
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

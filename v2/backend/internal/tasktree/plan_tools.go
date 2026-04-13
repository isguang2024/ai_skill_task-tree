package tasktree

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func (a *App) treeView(ctx context.Context, taskID string, stageNodeID *string, onlyExecutable bool) (jsonMap, error) {
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		return nil, err
	}
	children := map[string][]jsonMap{}
	byID := map[string]jsonMap{}
	for _, node := range nodes {
		parentID := asString(node["parent_node_id"])
		if parentID == "" {
			parentID = "__root__"
		}
		children[parentID] = append(children[parentID], node)
		byID[asString(node["id"])] = node
	}
	for parent := range children {
		sort.Slice(children[parent], func(i, j int) bool {
			return naturalPathLess(asString(children[parent][i]["path"]), asString(children[parent][j]["path"]))
		})
	}
	includeIDs := map[string]struct{}{}
	if onlyExecutable {
		currentStageID := asString(task["current_stage_node_id"])
		if stageNodeID != nil && strings.TrimSpace(*stageNodeID) != "" {
			currentStageID = strings.TrimSpace(*stageNodeID)
		}
		ordered := orderedExecutableLeaves(nodes)
		for _, node := range ordered {
			if currentStageID != "" && asString(node["stage_node_id"]) != currentStageID {
				continue
			}
			if asString(node["status"]) != "ready" && asString(node["status"]) != "running" {
				continue
			}
			if !dependsMet(node, byID) {
				continue
			}
			id := asString(node["id"])
			includeIDs[id] = struct{}{}
			parentID := asString(node["parent_node_id"])
			for parentID != "" {
				includeIDs[parentID] = struct{}{}
				parentID = asString(byID[parentID]["parent_node_id"])
			}
		}
	}
	lines := []string{}
	var render func(parentID, prefix string)
	render = func(parentID, prefix string) {
		items := children[parentID]
		for i, node := range items {
			id := asString(node["id"])
			if stageNodeID != nil && strings.TrimSpace(*stageNodeID) != "" {
				stage := strings.TrimSpace(*stageNodeID)
				if asString(node["role"]) != "stage" && asString(node["stage_node_id"]) != stage {
					continue
				}
			}
			if onlyExecutable {
				if _, ok := includeIDs[id]; !ok {
					continue
				}
			}
			branch := "├── "
			nextPrefix := prefix + "│   "
			if i == len(items)-1 {
				branch = "└── "
				nextPrefix = prefix + "    "
			}
			dependsOn := stringSliceFromAny(node["depends_on"])
			line := fmt.Sprintf("%s%s%s [%s/%s]", prefix, branch, asString(node["title"]), asString(node["status"]), asString(node["role"]))
			if len(dependsOn) > 0 {
				line += " deps=" + strings.Join(dependsOn, ",")
			}
			lines = append(lines, line)
			render(id, nextPrefix)
		}
	}
	render("__root__", "")
	return jsonMap{
		"task_id": taskID,
		"title":   asString(task["title"]),
		"text":    strings.Join(lines, "\n"),
		"lines":   lines,
	}, nil
}

func (a *App) importPlan(ctx context.Context, body importPlanBody) (jsonMap, error) {
	format := strings.ToLower(strings.TrimSpace(body.Format))
	if format == "" {
		format = "markdown"
	}
	raw := strings.TrimSpace(body.Data)
	if raw == "" {
		return nil, &appError{Code: 400, Msg: "data required"}
	}
	content := raw
	if format == "markdown" {
		content = extractFencedPlanBlock(raw)
		if content == "" {
			return nil, &appError{Code: 400, Msg: "markdown 导入需要包含 ```yaml 或 ```json 代码块"}
		}
		format = detectStructuredFormat(content)
	}
	var task taskCreate
	switch format {
	case "json":
		if err := json.Unmarshal([]byte(content), &task); err != nil {
			return nil, &appError{Code: 400, Msg: fmt.Sprintf("json 解析失败: %v", err)}
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal([]byte(content), &task); err != nil {
			return nil, &appError{Code: 400, Msg: fmt.Sprintf("yaml 解析失败: %v", err)}
		}
	default:
		return nil, &appError{Code: 400, Msg: "format 仅支持 markdown/yaml/json"}
	}
	apply := body.Apply != nil && *body.Apply
	if !apply {
		dr := true
		task.DryRun = &dr
	}
	return a.createTask(ctx, task)
}

func extractFencedPlanBlock(md string) string {
	lines := strings.Split(md, "\n")
	inBlock := false
	buf := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				return strings.TrimSpace(strings.Join(buf, "\n"))
			}
			lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			if lang == "yaml" || lang == "yml" || lang == "json" || lang == "" {
				inBlock = true
				buf = buf[:0]
			}
			continue
		}
		if inBlock {
			buf = append(buf, line)
		}
	}
	return ""
}

func detectStructuredFormat(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}
	return "yaml"
}

package tasktree

import (
	"encoding/json"
	"strings"
)

func mergeNodeMemoryRuns(node jsonMap, runs []jsonMap) jsonMap {
	out := jsonMap{
		"summary_text":  defaultNodeMemorySummary(node),
		"conclusions":   []string{},
		"decisions":     []any{},
		"risks":         []any{},
		"blockers":      []any{},
		"next_actions":  []any{},
		"evidence":      []string{},
		"source_run_id": nil,
	}
	summary := strings.TrimSpace(asString(out["summary_text"]))
	evidence := []string{}
	latestRunID := ""
	for _, run := range runs {
		if asString(run["status"]) == "running" {
			continue
		}
		latestRunID = asString(run["id"])
		structured := asAnyMap(run["structured_result"])
		if structured == nil {
			structured = jsonMap{}
		}
		if text := firstNonEmpty(
			asString(structured["summary_text"]),
			asString(structured["summary"]),
			asString(run["output_preview"]),
		); text != "" {
			summary = text
		}
		out["conclusions"] = mergeStringSlice(out["conclusions"], structured["conclusions"])
		out["decisions"] = mergeAnySlice(out["decisions"], structured["decisions"])
		out["risks"] = mergeAnySlice(out["risks"], structured["risks"])
		out["blockers"] = mergeAnySlice(out["blockers"], structured["blockers"])
		out["next_actions"] = mergeAnySlice(out["next_actions"], structured["next_actions"])
		evidence = mergeStringSlice(evidence,
			structured["evidence"],
			run["output_preview"],
			run["error_text"],
		)
	}
	if summary == "" {
		summary = defaultNodeMemorySummary(node)
	}
	out["summary_text"] = summary
	out["evidence"] = trimTailStrings(evidence, 10)
	if latestRunID != "" {
		out["source_run_id"] = latestRunID
	}
	return out
}

func mergeStageMemory(stage jsonMap, nodeMemories []jsonMap) jsonMap {
	summaryParts := []string{}
	conclusions := []string{}
	decisions := []any{}
	risks := []any{}
	blockers := []any{}
	nextActions := []any{}
	evidence := []string{}
	for _, mem := range nodeMemories {
		if text := strings.TrimSpace(asString(mem["summary_text"])); text != "" {
			summaryParts = append(summaryParts, text)
		}
		conclusions = mergeStringSlice(conclusions, mem["conclusions"])
		decisions = mergeAnySlice(decisions, mem["decisions"])
		risks = mergeAnySlice(risks, mem["risks"])
		blockers = mergeAnySlice(blockers, mem["blockers"])
		nextActions = mergeAnySlice(nextActions, mem["next_actions"])
		evidence = mergeStringSlice(evidence, mem["evidence"])
	}
	return jsonMap{
		"summary_text": joinSummaryParts(defaultStageMemorySummary(stage), summaryParts),
		"conclusions":  conclusions,
		"decisions":    decisions,
		"risks":        risks,
		"blockers":     blockers,
		"next_actions": nextActions,
		"evidence":     trimTailStrings(evidence, 12),
	}
}

func mergeTaskMemory(task jsonMap, currentStage jsonMap, stageMemories []jsonMap, remaining jsonMap) jsonMap {
	summaryParts := []string{}
	conclusions := []string{}
	decisions := []any{}
	risks := []any{}
	blockers := []any{}
	nextActions := []any{}
	evidence := []string{}
	for _, mem := range stageMemories {
		if text := strings.TrimSpace(asString(mem["summary_text"])); text != "" {
			summaryParts = append(summaryParts, text)
		}
		conclusions = mergeStringSlice(conclusions, mem["conclusions"])
		decisions = mergeAnySlice(decisions, mem["decisions"])
		risks = mergeAnySlice(risks, mem["risks"])
		blockers = mergeAnySlice(blockers, mem["blockers"])
		nextActions = mergeAnySlice(nextActions, mem["next_actions"])
		evidence = mergeStringSlice(evidence, mem["evidence"])
	}
	summary := defaultTaskMemorySummary(task, currentStage, remaining)
	if len(summaryParts) > 0 {
		summary = joinSummaryParts(summary, summaryParts)
	}
	return jsonMap{
		"summary_text":  summary,
		"conclusions":   conclusions,
		"decisions":     decisions,
		"risks":         risks,
		"blockers":      blockers,
		"next_actions":  nextActions,
		"evidence":      trimTailStrings(evidence, 15),
		"source_run_id": nil,
	}
}

func defaultNodeMemorySummary(node jsonMap) string {
	return strings.TrimSpace(strings.Join([]string{
		"节点：", asString(node["title"]),
		"；状态：", asString(node["status"]),
		"；路径：", asString(node["path"]),
	}, ""))
}

func defaultStageMemorySummary(stage jsonMap) string {
	return strings.TrimSpace("阶段：" + asString(stage["title"]) + "（" + asString(stage["path"]) + "）")
}

func defaultTaskMemorySummary(task jsonMap, currentStage jsonMap, remaining jsonMap) string {
	parts := []string{"任务：" + asString(task["title"])}
	if goal := strings.TrimSpace(asString(task["goal"])); goal != "" {
		parts = append(parts, "目标："+goal)
	}
	if stageTitle := strings.TrimSpace(asString(currentStage["title"])); stageTitle != "" {
		parts = append(parts, "当前阶段："+stageTitle)
	}
	if remaining != nil {
		parts = append(parts, "剩余节点："+asString(remaining["remaining_nodes"]))
	}
	return strings.Join(parts, "；")
}

func joinSummaryParts(base string, parts []string) string {
	if len(parts) == 0 {
		return base
	}
	if base == "" {
		return strings.Join(parts, " | ")
	}
	return base + " | " + strings.Join(parts, " | ")
}

func mergeStringSlice(current any, values ...any) []string {
	seen := map[string]struct{}{}
	out := []string{}
	add := func(text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		if _, ok := seen[text]; ok {
			return
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	for _, existing := range toStringSlice(current) {
		add(existing)
	}
	for _, value := range values {
		for _, item := range toStringSlice(value) {
			add(item)
		}
	}
	return out
}

func mergeAnySlice(current any, values ...any) []any {
	seen := map[string]struct{}{}
	out := []any{}
	add := func(v any) {
		keyBytes, _ := json.Marshal(v)
		key := string(keyBytes)
		if key == "" || key == "null" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	for _, item := range toAnySlice(current) {
		add(item)
	}
	for _, value := range values {
		for _, item := range toAnySlice(value) {
			add(item)
		}
	}
	return out
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{t}
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s := strings.TrimSpace(asString(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		s := strings.TrimSpace(asString(v))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func toAnySlice(v any) []any {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		return t
	case []string:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, item)
		}
		return out
	default:
		if asString(v) == "" {
			return nil
		}
		return []any{v}
	}
}

func trimTailStrings(items []string, limit int) []string {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

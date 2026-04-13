package tasktree

import "strings"

func slugifyToken(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "x"
	}
	return out
}

func preferredTaskToken(task jsonMap) string {
	if token := slugifyToken(asString(task["task_key"])); token != "" && token != "x" {
		return token
	}
	if token := slugifyToken(asString(task["title"])); token != "" {
		return token
	}
	return "task"
}

func preferredStageToken(stage jsonMap) string {
	if token := slugifyToken(asString(stage["node_key"])); token != "" && token != "x" {
		return token
	}
	if token := slugifyToken(asString(stage["title"])); token != "" {
		return token
	}
	return "stage"
}

func buildGitSuggestion(task jsonMap, stage jsonMap) jsonMap {
	branch := "feature/" + preferredTaskToken(task) + "-" + preferredStageToken(stage)
	return jsonMap{
		"branch_name": branch,
		"reason":      "阶段切换后建议在独立分支上执行，便于阶段提交与回滚",
	}
}

func buildPRSuggestion(task jsonMap, stage jsonMap) jsonMap {
	stagePath := asString(stage["path"])
	if stagePath == "" {
		stagePath = asString(stage["title"])
	}
	return jsonMap{
		"stage_node_id": asString(stage["id"]),
		"stage_path":    stagePath,
		"checklist": []string{
			"确认当前阶段节点及子节点均为 done/canceled",
			"整理本阶段 files_created/files_modified 与 commands_verified",
			"补充阶段验收证据（测试输出、截图或日志链接）",
			"准备 PR 描述：目标、改动、验证、风险与回滚方案",
		},
	}
}

func stageLooksCompleted(stage jsonMap) bool {
	status := strings.TrimSpace(strings.ToLower(asString(stage["status"])))
	return status == "done" || status == "canceled" || status == "closed"
}

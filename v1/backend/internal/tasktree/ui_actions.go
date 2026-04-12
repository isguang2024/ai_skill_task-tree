package tasktree

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type uiFormPageData struct {
	Title     string
	Heading   string
	Copy      string
	BackURL   string
	SubmitURL string
	Flash     string
	Error     string
	Task      *uiTaskCard
	Node      *uiNodeCard
}

var uiFormTpl = template.Must(template.New("form").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f7f8fb; --panel: #fff; --panel-soft: #f3f5fa;
      --text: #0b1220; --muted: #6b7280; --muted-strong: #4b5563;
      --line: #e5e7eb; --line-strong: #cbd5e1;
      --accent: #2563eb; --accent-hover: #1d4ed8; --accent-soft: rgba(37,99,235,.08);
      --ok: #16a34a; --bad: #dc2626; --bad-soft: rgba(220,38,38,.10);
      --radius-lg: 14px; --radius-md: 10px;
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; padding: 0; }
    body { min-height: 100vh; color: var(--text); font-family: "Segoe UI Variable Text","Segoe UI","PingFang SC","Microsoft YaHei",system-ui,sans-serif; background: var(--bg); font-size: 14px; line-height: 1.5; -webkit-font-smoothing: antialiased; }
    a { color: inherit; text-decoration: none; }
    button, input, textarea, select { font: inherit; color: inherit; }
    form { display: grid; gap: 12px; }
    .app { max-width: 760px; margin: 0 auto; padding: 20px 18px 40px; }
    .topbar { display: flex; align-items: center; gap: 10px; padding: 4px 0 16px; border-bottom: 1px solid var(--line); margin-bottom: 20px; }
    .topbar .brand { font-size: 15px; font-weight: 700; letter-spacing: -.02em; }
    .topbar .sep { color: var(--line-strong); }
    .topbar .page { color: var(--muted); font-size: 14px; }
    .btn-back { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); padding: 6px 12px; border-radius: 8px; background: #fff; color: var(--muted-strong); font-size: 13px; font-weight: 600; cursor: pointer; margin-left: auto; }
    .btn-back:hover { background: var(--panel-soft); border-color: var(--line-strong); }
    .panel { background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 20px 22px; margin-bottom: 14px; }
    .panel h1 { margin: 0 0 6px; font-size: 22px; font-weight: 700; letter-spacing: -.02em; }
    .panel .copy { margin: 0 0 18px; color: var(--muted); font-size: 13px; line-height: 1.65; }
    .banner { padding: 10px 14px; border-radius: var(--radius-md); margin-bottom: 14px; font-size: 13px; border: 1px solid transparent; }
    .banner.flash { background: rgba(22,163,74,.10); color: var(--ok); border-color: rgba(22,163,74,.22); }
    .banner.error { background: var(--bad-soft); color: var(--bad); border-color: rgba(220,38,38,.22); }
    .field { width: 100%; border: 1px solid var(--line); border-radius: 8px; background: #fff; padding: 9px 12px; }
    .field:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-soft); }
    textarea.field { min-height: 110px; resize: vertical; }
    .grid-2 { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 12px; }
    .full { grid-column: 1 / -1; }
    label { display: block; color: var(--muted); font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: .04em; margin-bottom: 4px; }
    .fgroup { display: grid; gap: 4px; }
    .btn { display: inline-flex; align-items: center; border: 1px solid var(--line); padding: 8px 14px; border-radius: 8px; background: #fff; color: var(--text); font-size: 13px; font-weight: 600; cursor: pointer; }
    .btn:hover { background: var(--panel-soft); border-color: var(--line-strong); }
    .btn.primary { background: var(--accent); border-color: var(--accent); color: #fff; }
    .btn.primary:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
    .btn.ok { background: var(--ok); border-color: var(--ok); color: #fff; }
    .btn.bad { color: var(--bad); }
    .btn.bad:hover { background: var(--bad-soft); }
    .btn.ghost { background: transparent; border-color: transparent; color: var(--muted); }
    .btn.ghost:hover { background: var(--panel-soft); color: var(--text); }
    .meta { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 16px; }
    .pill { display: inline-flex; align-items: center; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; background: var(--panel-soft); color: var(--muted-strong); border: 1px solid var(--line); }
    .section-divider { height: 1px; background: var(--line); margin: 16px 0; }
    .section-title { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); margin: 0 0 12px; }
    @media (max-width: 640px) { .grid-2 { grid-template-columns: 1fr; } .app { padding: 14px 14px 32px; } }
  </style>
</head>
<body>
  <div class="app">
    <div class="topbar">
      <span class="brand">Task Tree</span>
      <span class="sep">/</span>
      <span class="page">{{.Heading}}</span>
      <a class="btn-back" href="{{.BackURL}}">← 返回</a>
    </div>
    {{if .Flash}}<div class="banner flash">{{.Flash}}</div>{{end}}
    {{if .Error}}<div class="banner error">{{.Error}}</div>{{end}}
    {{if .Task}}
    <div class="meta">
      <span class="pill">{{if .Task.TaskKey}}{{.Task.TaskKey}}{{else}}{{.Task.ID}}{{end}}</span>
      <span class="pill">{{.Task.Title}}</span>
      <span class="pill">{{.Task.Status}}</span>
    </div>
    {{end}}
    {{if .Node}}
    <div class="meta">
      <span class="pill">{{.Node.Path}}</span>
      <span class="pill">{{.Node.Title}}</span>
      <span class="pill">{{.Node.Status}}</span>
    </div>
    {{end}}
    {{template "body" .}}
  </div>
</body>
</html>`))

var uiCreateTaskTpl = template.Must(template.Must(uiFormTpl.Clone()).Parse(`{{define "body"}}
<section class="panel">
  <h1>新建任务</h1>
  <p class="copy">标题用名词短语，goal 写清交付标准、约束和范围外项（2-4 句）。</p>
  <form method="post" action="{{.SubmitURL}}">
    <div class="fgroup">
      <label>任务标题</label>
      <input class="field" type="text" name="title" placeholder="例如：重构订单同步链路" required>
    </div>
    <div class="fgroup">
      <label>任务目标 (goal)</label>
      <textarea class="field" name="goal" rows="5" placeholder="例如：把 task-tree 的 MCP 能力与 HTTP API 对齐，覆盖增删改查/lease/claim 全链路。不重写 UI。不换端口。"></textarea>
    </div>
    <div class="fgroup">
      <label>任务 Key（可选）</label>
      <input class="field" type="text" name="task_key" placeholder="例如：SYNC">
    </div>
    <div>
      <button class="btn primary" type="submit">创建任务</button>
    </div>
  </form>
</section>
{{end}}`))

var uiCreateNodeTpl = template.Must(template.Must(uiFormTpl.Clone()).Parse(`{{define "body"}}
<section class="panel">
  <h1>新增节点</h1>
  <p class="copy">标题用动词短语；instruction 具体到文件、函数、步骤、命令。</p>
  <form method="post" action="{{.SubmitURL}}">
    <div class="fgroup">
      <label>节点标题</label>
      <input class="field" type="text" name="title" placeholder="例如：补 happy path 回归测试" required>
    </div>
    <div class="grid-2">
      <div class="fgroup">
        <label>节点 Key（可选）</label>
        <input class="field" type="text" name="node_key" placeholder="留空自动生成">
      </div>
      <div class="fgroup">
        <label>估时（小时）</label>
        <input class="field" type="text" name="estimate" placeholder="例如：1.5">
      </div>
    </div>
    <div class="fgroup">
      <label>父节点 ID（空 = 根节点）</label>
      <input class="field" type="text" name="parent_node_id" value="{{if .Node}}{{.Node.ID}}{{end}}">
    </div>
    <div class="fgroup">
      <label>Instruction</label>
      <textarea class="field" name="instruction" rows="5" placeholder="具体到文件、函数、步骤、命令"></textarea>
    </div>
    <div class="fgroup">
      <label>验收标准（一行一条）</label>
      <textarea class="field" name="acceptance" rows="4" placeholder="一行一条可验证条件"></textarea>
    </div>
    <div>
      <button class="btn primary" type="submit">创建节点</button>
    </div>
  </form>
</section>
{{end}}`))

var uiNodeActionTpl = template.Must(template.Must(uiFormTpl.Clone()).Parse(`{{define "body"}}
<section class="panel">
  <h1>详细填写</h1>
  <p class="copy">这里专门用于填写较长的进度说明和完成说明。工作台负责快操作，这里负责写完整内容。</p>

  <p class="section-title">进度上报</p>
  <form method="post" action="/ui/nodes/{{.Node.ID}}/progress">
    <div class="grid-2">
      <div class="fgroup">
        <label>进度增量 (0.0 – 1.0)</label>
        <input class="field" type="text" name="delta" value="0.1">
      </div>
    </div>
    <div class="fgroup">
      <label>说明（做了什么 / 证据 / 偏差 / 遗留）</label>
      <textarea class="field" name="message" rows="7" placeholder="做了什么:&#10;- 改动 1&#10;&#10;证据:&#10;- 测试输出 / 命令&#10;&#10;偏差:&#10;- 实际调整&#10;&#10;遗留:&#10;- 还剩什么"></textarea>
    </div>
    <div>
      <button class="btn primary" type="submit">写入进度</button>
    </div>
  </form>

  <div class="section-divider"></div>
  <p class="section-title">完成节点</p>
  <form method="post" action="/ui/nodes/{{.Node.ID}}/complete">
    <div class="fgroup">
      <label>完成说明（必填，四段结构）</label>
      <textarea class="field" name="message" rows="7" required placeholder="做了什么:&#10;- xxx&#10;&#10;证据:&#10;- go test ./... 通过&#10;&#10;偏差:&#10;- 无&#10;&#10;遗留:&#10;- 无"></textarea>
    </div>
    <div>
      <button class="btn ok" type="submit">标记完成</button>
    </div>
  </form>
</section>
{{end}}`))

func (a *App) renderCreateTaskPage(w http.ResponseWriter, r *http.Request) {
	a.renderFormPage(w, uiCreateTaskTpl, uiFormPageData{
		Title:     "新建任务 · Task Tree",
		Heading:   "新建任务",
		Copy:      "适合刚起步的大任务。创建后会立刻跳到任务详情页，方便继续补节点。",
		BackURL:   "/",
		SubmitURL: "/ui/tasks/create",
		Flash:     r.URL.Query().Get("flash"),
		Error:     r.URL.Query().Get("error"),
	})
}

func (a *App) renderCreateNodePage(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := a.getTask(r.Context(), taskID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	card := toUITaskCard(task)
	var parentNode *uiNodeCard
	if parentID := strings.TrimSpace(r.URL.Query().Get("parent_node_id")); parentID != "" {
		node, err := a.findNode(r.Context(), parentID, false)
		if err == nil && asString(node["task_id"]) == taskID {
			tmp := toUINodeCard(node)
			parentNode = &tmp
		}
	}
	a.renderFormPage(w, uiCreateNodeTpl, uiFormPageData{
		Title:     "新增节点 · Task Tree",
		Heading:   "为任务新增节点",
		Copy:      "这里适合录入较完整的 instruction 和验收标准，不挤在详情页里填写。",
		BackURL:   "/tasks/" + taskID,
		SubmitURL: "/ui/tasks/" + taskID + "/nodes/create",
		Flash:     r.URL.Query().Get("flash"),
		Error:     r.URL.Query().Get("error"),
		Task:      &card,
		Node:      parentNode,
	})
}

func (a *App) renderNodeActionsPage(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	task, err := a.getTask(r.Context(), asString(node["task_id"]), false)
	if err != nil {
		writeErr(w, err)
		return
	}
	taskCard := toUITaskCard(task)
	nodeCard := toUINodeCard(node)
	a.renderFormPage(w, uiNodeActionTpl, uiFormPageData{
		Title:   "详细填写 · Task Tree",
		Heading: "详细填写",
		Copy:    "这里适合专心填写长一点的进度说明或完成说明，不和工作台里的快操作混在一起。",
		BackURL: "/tasks/" + asString(node["task_id"]),
		Flash:   r.URL.Query().Get("flash"),
		Error:   r.URL.Query().Get("error"),
		Task:    &taskCard,
		Node:    &nodeCard,
	})
}

func (a *App) renderFormPage(w http.ResponseWriter, tpl *template.Template, data uiFormPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		writeErr(w, err)
	}
}

func (a *App) handleUICreateTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.redirectWithError(w, r, "/new-task", err.Error())
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	goal := strings.TrimSpace(r.FormValue("goal"))
	taskKey := strings.TrimSpace(r.FormValue("task_key"))
	body := taskCreate{Title: title}
	if goal != "" {
		body.Goal = &goal
	}
	if taskKey != "" {
		body.TaskKey = &taskKey
	}
	task, err := a.createTask(r.Context(), body)
	if err != nil {
		a.redirectWithError(w, r, "/new-task", err.Error())
		return
	}
	http.Redirect(w, r, "/tasks/"+asString(task["id"])+"?flash="+url.QueryEscape("任务已创建"), http.StatusSeeOther)
}

func (a *App) handleUICreateNode(w http.ResponseWriter, r *http.Request, taskID string) {
	if err := r.ParseForm(); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID+"/new-node", err.Error())
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	nodeKey := strings.TrimSpace(r.FormValue("node_key"))
	parentNodeID := strings.TrimSpace(r.FormValue("parent_node_id"))
	instruction := strings.TrimSpace(r.FormValue("instruction"))
	body := nodeCreate{
		Title:              title,
		AcceptanceCriteria: parseCriteriaLines(r.FormValue("acceptance")),
	}
	if nodeKey != "" {
		body.NodeKey = &nodeKey
	}
	if parentNodeID != "" {
		body.ParentNodeID = &parentNodeID
	}
	if instruction != "" {
		body.Instruction = &instruction
	}
	if estimate := parseEstimate(strings.TrimSpace(r.FormValue("estimate"))); estimate != nil {
		body.Estimate = estimate
	}
	if _, err := a.createNode(r.Context(), taskID, body); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID+"/new-node", err.Error())
		return
	}
	http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(parentNodeID), "children", "节点已创建", ""), http.StatusSeeOther)
}

func (a *App) handleUIClaimNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	tool := "ui"
	agentID := "browser"
	_, err = a.claimNode(r.Context(), nodeID, claimBody{
		Actor: actor{Tool: &tool, AgentID: &agentID},
	})
	if err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已 claim", ""), http.StatusSeeOther)
}

func (a *App) handleUIReleaseNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if _, err := a.releaseNode(r.Context(), nodeID); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已 release", ""), http.StatusSeeOther)
}

func (a *App) handleUIRetypeNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	tool := "ui"
	agentID := "browser"
	message := strings.TrimSpace(r.FormValue("message"))
	body := retypeBody{Actor: &actor{Tool: &tool, AgentID: &agentID}}
	if message != "" {
		body.Message = &message
	}
	if _, err := a.retypeNodeToLeaf(r.Context(), nodeID, body); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已转回执行节点", ""), http.StatusSeeOther)
}

func (a *App) handleUIProgressNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	delta, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("delta")), 64)
	if err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, "delta 不是合法数字")
		return
	}
	tool := "ui"
	agentID := "browser"
	message := strings.TrimSpace(r.FormValue("message"))
	body := progressBody{DeltaProgress: &delta, Actor: &actor{Tool: &tool, AgentID: &agentID}}
	if message != "" {
		body.Message = &message
	}
	if _, err := a.reportProgress(r.Context(), nodeID, body); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "progress 已更新", ""), http.StatusSeeOther)
}

func (a *App) handleUICompleteNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	tool := "ui"
	agentID := "browser"
	message := strings.TrimSpace(r.FormValue("message"))
	body := completeBody{Actor: &actor{Tool: &tool, AgentID: &agentID}}
	if message != "" {
		body.Message = &message
	}
	if _, err := a.completeNode(r.Context(), nodeID, body); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已完成", ""), http.StatusSeeOther)
}

func (a *App) handleUIBlockNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, "reason 不能为空")
		return
	}
	tool := "ui"
	agentID := "browser"
	if _, err := a.blockNode(r.Context(), nodeID, blockBody{Reason: reason, Actor: &actor{Tool: &tool, AgentID: &agentID}}); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已阻塞", ""), http.StatusSeeOther)
}

func (a *App) handleUIUpdateTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if err := r.ParseForm(); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID, err.Error())
		return
	}
	taskKey := strings.TrimSpace(r.FormValue("task_key"))
	title := strings.TrimSpace(r.FormValue("title"))
	goal := strings.TrimSpace(r.FormValue("goal"))
	body := taskUpdate{
		TaskKey: &taskKey,
		Title:   &title,
		Goal:    &goal,
	}
	if _, err := a.updateTask(r.Context(), taskID, body); err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, "", "edit", "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(r.FormValue("node_id")), strings.TrimSpace(r.FormValue("tab")), "任务已保存", ""), http.StatusSeeOther)
}

func (a *App) handleUITaskTransition(w http.ResponseWriter, r *http.Request, taskID string) {
	if err := r.ParseForm(); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID, err.Error())
		return
	}
	action := strings.TrimSpace(r.FormValue("action"))
	message := strings.TrimSpace(r.FormValue("message"))
	tool := "ui"
	agentID := "browser"
	body := transitionBody{
		Action: action,
		Actor:  &actor{Tool: &tool, AgentID: &agentID},
	}
	if message != "" {
		body.Message = &message
	}
	if _, err := a.transitionTask(r.Context(), taskID, body); err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(r.FormValue("node_id")), strings.TrimSpace(r.FormValue("tab")), "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(r.FormValue("node_id")), strings.TrimSpace(r.FormValue("tab")), taskTransitionSuccessLabel(action), ""), http.StatusSeeOther)
}

func (a *App) handleUIUpdateNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	instruction := strings.TrimSpace(r.FormValue("instruction"))
	estimate := parseEstimate(strings.TrimSpace(r.FormValue("estimate")))
	criteria := parseCriteriaLines(r.FormValue("acceptance"))
	body := nodeUpdate{
		Title:              &title,
		Instruction:        &instruction,
		AcceptanceCriteria: &criteria,
		Estimate:           estimate,
	}
	if _, err := a.updateNode(r.Context(), nodeID, body); err != nil {
		http.Redirect(w, r, buildTaskURL(asString(node["task_id"]), nodeID, "edit", "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, "节点已保存", ""), http.StatusSeeOther)
}

func (a *App) handleUINodeTransition(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, err := a.findNode(r.Context(), nodeID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.redirectNodeError(w, r, asString(node["task_id"]), nodeID, err.Error())
		return
	}
	action := strings.TrimSpace(r.FormValue("action"))
	message := strings.TrimSpace(r.FormValue("message"))
	tool := "ui"
	agentID := "browser"
	body := transitionBody{
		Action: action,
		Actor:  &actor{Tool: &tool, AgentID: &agentID},
	}
	if message != "" {
		body.Message = &message
	}
	if _, err := a.transitionNode(r.Context(), nodeID, body); err != nil {
		http.Redirect(w, r, buildTaskURL(asString(node["task_id"]), nodeID, strings.TrimSpace(r.FormValue("tab")), "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, uiNodeTargetURL(r, asString(node["task_id"]), nodeID, nodeTransitionSuccessLabel(action), ""), http.StatusSeeOther)
}

func (a *App) handleUICreateArtifact(w http.ResponseWriter, r *http.Request, taskID string) {
	if err := r.ParseForm(); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID, err.Error())
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	kind := strings.TrimSpace(r.FormValue("kind"))
	uri := strings.TrimSpace(r.FormValue("uri"))
	nodeID := strings.TrimSpace(r.FormValue("node_id"))
	if uri == "" {
		http.Redirect(w, r, buildTaskURL(taskID, nodeID, "artifacts", "", "uri 不能为空"), http.StatusSeeOther)
		return
	}
	body := artifactCreate{
		NodeID: strPtr(nodeID),
		Kind:   strPtr(kind),
		Title:  strPtr(title),
		URI:    uri,
	}
	if _, err := a.createArtifact(r.Context(), taskID, body); err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, nodeID, "artifacts", "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, buildTaskURL(taskID, nodeID, "artifacts", "链接产物已添加", ""), http.StatusSeeOther)
}

func (a *App) handleUIUploadArtifact(w http.ResponseWriter, r *http.Request, taskID string) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(r.FormValue("node_id")), "artifacts", "", "请选择文件"), http.StatusSeeOther)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, strings.TrimSpace(r.FormValue("node_id")), "artifacts", "", err.Error()), http.StatusSeeOther)
		return
	}
	nodeID := strings.TrimSpace(r.FormValue("node_id"))
	var nodePtr *string
	if nodeID != "" {
		nodePtr = &nodeID
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if kind == "" {
		kind = "file"
	}
	if _, err := a.storeArtifactFile(r.Context(), taskID, header, content, nodePtr, kind); err != nil {
		http.Redirect(w, r, buildTaskURL(taskID, nodeID, "artifacts", "", err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, buildTaskURL(taskID, nodeID, "artifacts", "文件产物已上传", ""), http.StatusSeeOther)
}

func (a *App) handleUISoftDeleteTask(w http.ResponseWriter, r *http.Request, taskID string) {
	result, err := a.softDeleteTask(r.Context(), taskID)
	if err != nil {
		a.redirectWithError(w, r, "/tasks/"+taskID, err.Error())
		return
	}
	flash := "任务已移入回收站"
	if count := int(asFloat(result["unlinked_references"])); count > 0 {
		flash = fmt.Sprintf("任务已移入回收站，并自动解除 %d 个关联节点引用", count)
	}
	target := uiReturnURL(r, "/")
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	http.Redirect(w, r, target+sep+"flash="+url.QueryEscape(flash), http.StatusSeeOther)
}

func (a *App) handleUIRestoreTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if _, err := a.restoreTask(r.Context(), taskID); err != nil {
		a.redirectWithError(w, r, "/trash", err.Error())
		return
	}
	http.Redirect(w, r, "/tasks/"+taskID+"?flash="+url.QueryEscape("任务已恢复"), http.StatusSeeOther)
}

func (a *App) handleUIHardDeleteTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if _, err := a.hardDeleteTask(r.Context(), taskID); err != nil {
		a.redirectWithError(w, r, "/trash", err.Error())
		return
	}
	http.Redirect(w, r, "/trash?flash="+url.QueryEscape("任务已彻底删除"), http.StatusSeeOther)
}

func (a *App) handleUIEmptyTrash(w http.ResponseWriter, r *http.Request) {
	if _, err := a.emptyTrash(r.Context()); err != nil {
		a.redirectWithError(w, r, "/trash", err.Error())
		return
	}
	http.Redirect(w, r, "/trash?flash="+url.QueryEscape("回收站已清空"), http.StatusSeeOther)
}

func (a *App) redirectWithError(w http.ResponseWriter, r *http.Request, target, msg string) {
	http.Redirect(w, r, target+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

func (a *App) redirectNodeError(w http.ResponseWriter, r *http.Request, taskID, nodeID, msg string) {
	back := "/tasks/" + taskID
	if strings.Contains(r.URL.Path, "/nodes/") {
		back = "/nodes/" + nodeID + "/actions"
	}
	a.redirectWithError(w, r, back, msg)
}

func uiNodeTargetURL(r *http.Request, taskID, nodeID, flash, errMsg string) string {
	tab := strings.TrimSpace(r.FormValue("tab"))
	if tab == "" {
		tab = strings.TrimSpace(r.URL.Query().Get("tab"))
	}
	return buildTaskURL(taskID, nodeID, tab, flash, errMsg)
}

func buildTaskURL(taskID, nodeID, tab, flash, errMsg string) string {
	values := url.Values{}
	if strings.TrimSpace(nodeID) != "" {
		values.Set("node", strings.TrimSpace(nodeID))
	}
	if strings.TrimSpace(tab) != "" {
		values.Set("tab", strings.TrimSpace(tab))
	}
	if strings.TrimSpace(flash) != "" {
		values.Set("flash", strings.TrimSpace(flash))
	}
	if strings.TrimSpace(errMsg) != "" {
		values.Set("error", strings.TrimSpace(errMsg))
	}
	target := "/tasks/" + taskID
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	return target
}

func taskTransitionSuccessLabel(action string) string {
	switch action {
	case "pause":
		return "任务已暂停"
	case "reopen":
		return "任务已恢复"
	case "cancel":
		return "任务已取消"
	default:
		return "任务状态已更新"
	}
}

func nodeTransitionSuccessLabel(action string) string {
	switch action {
	case "pause":
		return "节点已暂停"
	case "reopen":
		return "节点已重开"
	case "cancel":
		return "节点已取消"
	case "unblock":
		return "节点已解除阻塞"
	default:
		return "节点状态已更新"
	}
}

func parseCriteriaLines(v string) []string {
	lines := strings.Split(strings.ReplaceAll(v, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func parseEstimate(v string) *float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &n
}

func uiTaskCopy(task jsonMap) uiTaskCard {
	card := toUITaskCard(task)
	return card
}

func uiNodeCopy(node jsonMap) uiNodeCard {
	card := toUINodeCard(node)
	return card
}

func (a *App) uiContextTask(taskID string) string {
	return "/tasks/" + taskID
}

func (a *App) uiContextNode(nodeID string) string {
	return "/nodes/" + nodeID + "/actions"
}

func uiSuccess(msg string) string {
	return fmt.Sprintf("?flash=%s", url.QueryEscape(msg))
}

func uiReturnURL(r *http.Request, fallback string) string {
	ref := strings.TrimSpace(r.Referer())
	if ref == "" {
		return fallback
	}
	u, err := url.Parse(ref)
	if err != nil {
		return fallback
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") {
		return fallback
	}
	target := u.Path
	if u.RawQuery != "" {
		target += "?" + u.RawQuery
	}
	return target
}


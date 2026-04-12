package tasktree

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
)

type uiPageData struct {
	Title       string
	Section     string
	Flash       string
	Error       string
	Tasks       []uiTaskCard
	Task        *uiTaskDetail
	WorkItems   []uiWorkItem
	SearchQuery string
	Search      *uiSearchResult
}

type uiTaskCard struct {
	ID        string
	TaskKey   string
	Title     string
	Goal      string
	Status    string
	Percent   int
	UpdatedAt string
	Deleted   bool
}

type uiTaskDetail struct {
	ID                string
	TaskKey           string
	Title             string
	Goal              string
	Status            string
	Percent           int
	UpdatedAt         string
	Remaining         int
	BlockCount        int
	Estimate          string
	NodeCount         int
	NextNode          *uiNodeCard
	NodeTrees         []uiNodeTree
	Events            []uiEventCard
	Artifacts         []uiArtifactCard
	ActiveTab         string
	SelectedNode      *uiNodeCard
	SelectedChildren  []uiNodeCard
	SelectedEvents    []uiEventCard
	SelectedArtifacts []uiArtifactCard
}

type uiNodeCard struct {
	ID           string
	ParentNodeID string
	Path         string
	Title        string
	Kind         string
	Status       string
	Progress     int
	Estimate     string
	Instruction  string
	Depth        int
	HasChildren  bool
	ChildCount   int
	IsSelected   bool
}

type uiNodeTree struct {
	Node     uiNodeCard
	Children []uiNodeTree
}

type uiEventCard struct {
	Type      string
	Message   string
	Actor     string
	CreatedAt string
}

type uiArtifactCard struct {
	ID        string
	Title     string
	Kind      string
	URI       string
	CreatedAt string
}

type uiWorkItem struct {
	TaskID    string
	TaskTitle string
	NodeID    string
	Path      string
	Title     string
	Status    string
	UpdatedAt string
}

type uiSearchResult struct {
	Tasks []uiTaskCard
	Nodes []uiSearchNode
}

type uiSearchNode struct {
	TaskID    string
	TaskTitle string
	NodeID    string
	Path      string
	Title     string
	Status    string
}

var uiPageTpl = template.Must(template.New("page").Funcs(template.FuncMap{
	"statusClass": func(v string) string {
		switch strings.ToLower(v) {
		case "done":
			return "ok"
		case "running":
			return "run"
		case "blocked", "canceled":
			return "bad"
		case "paused":
			return "pause"
		case "closed", "deleted":
			return "muted"
		default:
			return "todo"
		}
	},
	"eqs": func(a, b string) bool { return a == b },
}).Parse(`{{define "nodeTree"}}
<div class="tree-node depth-{{.Node.Depth}}">
  <div class="tree-card {{if .Node.HasChildren}}branch{{else}}leaf{{end}} {{if .Node.IsSelected}}selected{{end}}">
    <div class="tree-main">
      <a class="tree-link" href="/tasks/{{$.Task.ID}}?node={{.Node.ID}}&tab=edit">
        <strong>{{.Node.Title}}</strong>
        <small>{{.Node.Path}}</small>
      </a>
      <div class="meta">
        <span class="pill {{statusClass .Node.Status}}">{{.Node.Status}}</span>
        <span class="pill">{{.Node.Progress}}%</span>
        {{if .Node.HasChildren}}<span class="pill">{{.Node.ChildCount}} 子节点</span>{{end}}
      </div>
    </div>
  </div>
  {{if .Children}}
  <div class="tree-children">
    {{range .Children}}{{template "nodeTree" .}}{{end}}
  </div>
  {{end}}
</div>
{{end}}
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f5f1e8;
      --panel: rgba(255,255,255,.76);
      --panel-strong: rgba(255,255,255,.92);
      --text: #161616;
      --muted: #6a645d;
      --line: rgba(22,22,22,.12);
      --accent: #bc4f2b;
      --accent-soft: #f1c3a2;
      --ok: #207a47;
      --run: #2452d1;
      --bad: #b02f2f;
      --todo: #8a5a16;
      --shadow: 0 16px 40px rgba(40, 30, 20, .09);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(188,79,43,.20), transparent 28%),
        radial-gradient(circle at 80% 20%, rgba(36,82,209,.12), transparent 24%),
        linear-gradient(180deg, #f7f3eb 0%, #efe6d7 100%);
      min-height: 100vh;
    }
    a { color: inherit; text-decoration: none; }
    .shell {
      max-width: 1280px;
      margin: 0 auto;
      padding: 28px;
    }
    .hero {
      display: grid;
      grid-template-columns: 1.2fr .8fr;
      gap: 20px;
      margin-bottom: 22px;
    }
    .hero-main, .hero-side, .panel, .card {
      background: var(--panel);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255,255,255,.6);
      border-radius: 24px;
      box-shadow: var(--shadow);
    }
    .hero-main {
      padding: 28px;
    }
    .hero-main h1 {
      margin: 0;
      font-size: clamp(36px, 6vw, 64px);
      line-height: .95;
      letter-spacing: -0.04em;
    }
    .hero-main p {
      margin: 18px 0 0;
      max-width: 54ch;
      color: var(--muted);
      line-height: 1.7;
      font-size: 15px;
    }
    .hero-side {
      padding: 24px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .hero-side strong {
      display: block;
      font-size: 14px;
      color: var(--muted);
      margin-bottom: 12px;
    }
    .hero-side .stat {
      font-size: 44px;
      font-weight: 700;
      letter-spacing: -0.04em;
    }
    .hero-side .sub {
      color: var(--muted);
      font-size: 13px;
    }
    .nav {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin: 0 0 22px;
    }
    .nav a {
      padding: 11px 16px;
      border-radius: 999px;
      background: rgba(255,255,255,.55);
      border: 1px solid rgba(22,22,22,.08);
      font-size: 14px;
    }
    .nav a.active {
      background: var(--text);
      color: white;
      border-color: var(--text);
    }
    .toolbar {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      align-items: center;
      margin-bottom: 18px;
    }
    .toolbar form {
      display: flex;
      gap: 10px;
      flex: 1;
      min-width: 260px;
    }
    .toolbar input {
      flex: 1;
      min-width: 0;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 12px 14px;
      background: var(--panel-strong);
      font: inherit;
    }
    .toolbar button, .toolbar .ghost {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      background: var(--accent);
      color: white;
      font: inherit;
      cursor: pointer;
    }
    .toolbar .ghost {
      background: rgba(22,22,22,.08);
      color: var(--text);
    }
    .grid {
      display: grid;
      gap: 16px;
    }
    .task-grid {
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    }
    .card {
      padding: 20px;
    }
    .eyebrow {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: center;
      margin-bottom: 14px;
      color: var(--muted);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .12em;
    }
    .card h2, .card h3 {
      margin: 0 0 10px;
      font-size: 22px;
      letter-spacing: -0.03em;
    }
    .card p {
      margin: 0;
      color: var(--muted);
      line-height: 1.7;
    }
    .meta {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 16px;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 10px;
      border-radius: 999px;
      font-size: 12px;
      background: rgba(22,22,22,.06);
      color: var(--muted);
    }
    .pill.ok { color: var(--ok); background: rgba(32,122,71,.10); }
    .pill.run { color: var(--run); background: rgba(36,82,209,.10); }
    .pill.bad { color: var(--bad); background: rgba(176,47,47,.10); }
    .pill.todo { color: var(--todo); background: rgba(138,90,22,.11); }
    .pill.muted { color: var(--muted); background: rgba(22,22,22,.06); }
    .progress {
      height: 10px;
      border-radius: 999px;
      background: rgba(22,22,22,.08);
      overflow: hidden;
      margin-top: 16px;
    }
    .progress span {
      display: block;
      height: 100%;
      background: linear-gradient(90deg, var(--accent) 0%, #dc8f53 100%);
    }
    .section {
      margin-top: 24px;
    }
    .section-head {
      display: flex;
      justify-content: space-between;
      align-items: end;
      gap: 12px;
      margin-bottom: 14px;
    }
    .section-head h2 {
      margin: 0;
      font-size: 24px;
      letter-spacing: -0.03em;
    }
    .section-head span {
      color: var(--muted);
      font-size: 13px;
    }
    .detail-grid {
      display: grid;
      grid-template-columns: 340px minmax(0, 1fr);
      gap: 18px;
    }
    .stack {
      display: grid;
      gap: 14px;
    }
    .list {
      display: grid;
      gap: 12px;
    }
    .row {
      padding: 14px 16px;
      border-radius: 18px;
      background: rgba(255,255,255,.6);
      border: 1px solid rgba(22,22,22,.06);
    }
    .row strong {
      display: block;
      margin-bottom: 6px;
      font-size: 15px;
    }
    .row small {
      color: var(--muted);
      display: block;
      line-height: 1.6;
    }
    .empty {
      padding: 26px;
      border-radius: 22px;
      background: rgba(255,255,255,.5);
      border: 1px dashed rgba(22,22,22,.16);
      color: var(--muted);
    }
    .tree {
      display: grid;
      gap: 10px;
    }
    .tree-node {
      position: relative;
      display: grid;
      gap: 10px;
    }
    .tree-children {
      margin-left: 22px;
      padding-left: 18px;
      border-left: 2px solid rgba(22,22,22,.08);
      display: grid;
      gap: 10px;
    }
    .tree-card {
      padding: 10px 12px;
      border-radius: 14px;
      background: rgba(255,255,255,.72);
      border: 1px solid rgba(22,22,22,.06);
    }
    .tree-card.branch {
      background: rgba(255,255,255,.9);
      border-color: rgba(188,79,43,.18);
    }
    .tree-card.selected {
      border-color: rgba(36,82,209,.28);
      box-shadow: inset 3px 0 0 #2452d1;
      background: #fff;
    }
    .tree-link {
      display: block;
    }
    .tree-main strong {
      display: block;
      margin-bottom: 4px;
      font-size: 14px;
    }
    .tree-main small {
      display: block;
      color: var(--muted);
      line-height: 1.4;
      font-size: 12px;
    }
    .detail-shell {
      display: grid;
      gap: 16px;
    }
    .detail-top {
      padding: 22px 24px;
    }
    .detail-top h2 {
      margin: 0 0 10px;
      font-size: 38px;
      letter-spacing: -0.04em;
    }
    .detail-actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 18px;
    }
    .detail-actions form, .detail-actions a {
      display: inline-flex;
    }
    .action-btn {
      border: 0;
      border-radius: 10px;
      padding: 10px 14px;
      color: #fff;
      font: inherit;
      cursor: pointer;
      background: #2452d1;
      text-decoration: none;
    }
    .action-btn.warn { background: #d58a00; }
    .action-btn.ok { background: #2e8a57; }
    .action-btn.bad { background: #cc4040; }
    .action-btn.muted { background: #6f7b90; }
    .tabs {
      display: flex;
      gap: 22px;
      border-bottom: 1px solid rgba(22,22,22,.1);
      margin-bottom: 18px;
      padding: 0 2px;
    }
    .tabs a {
      padding: 0 0 12px;
      color: var(--muted);
      font-weight: 600;
      border-bottom: 2px solid transparent;
    }
    .tabs a.active {
      color: var(--text);
      border-color: #2452d1;
    }
    .editor-grid {
      display: grid;
      gap: 14px;
    }
    .editor-row {
      display: grid;
      gap: 8px;
    }
    .editor-row label {
      font-size: 13px;
      color: var(--muted);
    }
    .editor-box {
      padding: 14px 16px;
      border-radius: 14px;
      border: 1px solid rgba(22,22,22,.08);
      background: rgba(255,255,255,.72);
      white-space: pre-wrap;
      line-height: 1.7;
    }
    .simple-list {
      display: grid;
      gap: 12px;
    }
    @media (max-width: 920px) {
      .hero, .detail-grid {
        grid-template-columns: 1fr;
      }
      .shell {
        padding: 18px;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="hero-main">
        <h1>Task Tree</h1>
        <p>本地任务树现在不只是接口。这里可以直接浏览任务状态、下一步节点、最近事件和产物，用浏览器也能把长任务的上下文捞回来。</p>
      </div>
      <div class="hero-side">
        <div>
          <strong>Current View</strong>
          <div class="stat">{{.Section}}</div>
        </div>
        <div class="sub">Go 内置页面，直接读取同一套 SQLite 与事件流。</div>
      </div>
    </section>

    <nav class="nav">
      <a href="/" class="{{if eq .Section "任务总览"}}active{{end}}">任务总览</a>
      <a href="/work" class="{{if eq .Section "可领取工作"}}active{{end}}">可领取工作</a>
      <a href="/trash" class="{{if eq .Section "回收站"}}active{{end}}">回收站</a>
      <a href="/search" class="{{if eq .Section "搜索"}}active{{end}}">搜索</a>
      <a href="/new-task">新建任务</a>
      {{if .Task}}<a href="/tasks/{{.Task.ID}}" class="active">当前任务</a>{{end}}
    </nav>

    {{if .Flash}}<div class="card" style="margin-bottom:16px; color: var(--ok);">{{.Flash}}</div>{{end}}
    {{if .Error}}<div class="card" style="margin-bottom:16px; color: var(--bad);">{{.Error}}</div>{{end}}

    {{if .Tasks}}
    <div class="toolbar">
      <form method="get" action="{{if eq .Section "回收站"}}/trash{{else}}/{{end}}">
        <input type="text" name="q" value="{{.SearchQuery}}" placeholder="按标题、目标搜索任务">
        <button type="submit">筛选</button>
      </form>
      {{if .SearchQuery}}<a class="ghost" href="{{if eq .Section "回收站"}}/trash{{else}}/{{end}}">清空</a>{{end}}
    </div>
    <section class="grid task-grid">
      {{range .Tasks}}
      <a class="card" href="/tasks/{{.ID}}">
        <div class="eyebrow">
          <span>{{if .TaskKey}}{{.TaskKey}}{{else}}{{.ID}}{{end}}</span>
          <span>{{.UpdatedAt}}</span>
        </div>
        <h2>{{.Title}}</h2>
        <p>{{if .Goal}}{{.Goal}}{{else}}没有填写 goal。{{end}}</p>
        <div class="progress"><span style="width: {{.Percent}}%"></span></div>
        <div class="meta">
          <span class="pill {{statusClass .Status}}">{{.Status}}</span>
          <span class="pill">{{.Percent}}%</span>
          {{if .Deleted}}<span class="pill muted">deleted</span>{{end}}
        </div>
      </a>
      {{end}}
    </section>
    {{end}}

    {{if .Task}}
    <section class="detail-grid">
      <aside class="panel card">
        <div class="eyebrow">
          <span>任务树</span>
          <a class="pill" href="/tasks/{{.Task.ID}}/new-node">+ 根节点</a>
        </div>
        <section class="section" style="margin-top: 8px;">
          <div class="section-head">
            <h2 style="font-size:20px;">节点树</h2>
            <span>{{.Task.NodeCount}} 个节点</span>
          </div>
          {{if .Task.NodeTrees}}
          <div class="tree">
            {{range .Task.NodeTrees}}{{template "nodeTree" .}}{{end}}
          </div>
          {{else}}
          <div class="empty">当前任务还没有节点。</div>
          {{end}}
        </section>
      </aside>

      <div class="detail-shell">
        <section class="panel detail-top">
          <div class="eyebrow">
            <span>{{if .Task.TaskKey}}{{.Task.TaskKey}}{{else}}{{.Task.ID}}{{end}}</span>
            <span>{{.Task.UpdatedAt}}</span>
          </div>
          {{if .Task.SelectedNode}}
          <h2>{{.Task.SelectedNode.Title}}</h2>
          <div class="meta">
            <span class="pill {{statusClass .Task.SelectedNode.Status}}">{{.Task.SelectedNode.Status}}</span>
            <span class="pill">{{.Task.SelectedNode.Path}}</span>
            <span class="pill">{{.Task.SelectedNode.Progress}}%</span>
            <span class="pill">{{.Task.SelectedNode.Estimate}}</span>
          </div>
          <div class="detail-actions">
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/claim"><button class="action-btn" type="submit">claim</button></form>
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/progress"><input type="hidden" name="delta" value="0.25"><input type="hidden" name="message" value="通过详情页快捷上报 25%"><button class="action-btn warn" type="submit">+25%</button></form>
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/progress"><input type="hidden" name="delta" value="0.5"><input type="hidden" name="message" value="通过详情页快捷上报 50%"><button class="action-btn warn" type="submit">+50%</button></form>
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/complete"><input type="hidden" name="message" value="通过详情页快捷完成"><button class="action-btn ok" type="submit">完成</button></form>
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/block"><input type="hidden" name="reason" value="通过详情页标记阻塞"><button class="action-btn bad" type="submit">阻塞</button></form>
            <form method="post" action="/ui/nodes/{{.Task.SelectedNode.ID}}/release"><button class="action-btn muted" type="submit">释放</button></form>
            <a class="action-btn muted" href="/tasks/{{.Task.ID}}/new-node?parent_node_id={{.Task.SelectedNode.ID}}">加子节点</a>
            <a class="action-btn muted" href="/nodes/{{.Task.SelectedNode.ID}}/actions">更多操作</a>
          </div>
          {{else if .Task.NextNode}}
          <h2>{{.Task.NextNode.Title}}</h2>
          <div class="meta">
            <span class="pill {{statusClass .Task.NextNode.Status}}">{{.Task.NextNode.Status}}</span>
            <span class="pill">{{.Task.NextNode.Path}}</span>
            <span class="pill">{{.Task.NextNode.Progress}}%</span>
          </div>
          {{else}}
          <h2>{{.Task.Title}}</h2>
          <p>{{if .Task.Goal}}{{.Task.Goal}}{{else}}这个任务还没有 goal。{{end}}</p>
          {{end}}
          <div class="progress"><span style="width: {{.Task.Percent}}%"></span></div>
          <div class="meta">
            <span class="pill {{statusClass .Task.Status}}">{{.Task.Status}}</span>
            <span class="pill">总进度 {{.Task.Percent}}%</span>
            <span class="pill">剩余 {{.Task.Remaining}} 节点</span>
            <span class="pill">阻塞 {{.Task.BlockCount}}</span>
            <span class="pill">{{.Task.Estimate}}</span>
          </div>
        </section>

        <section class="panel card">
          <div class="tabs">
            <a href="/tasks/{{.Task.ID}}?node={{if .Task.SelectedNode}}{{.Task.SelectedNode.ID}}{{end}}&tab=edit" class="{{if eqs .Task.ActiveTab "edit"}}active{{end}}">编辑</a>
            <a href="/tasks/{{.Task.ID}}?node={{if .Task.SelectedNode}}{{.Task.SelectedNode.ID}}{{end}}&tab=children" class="{{if eqs .Task.ActiveTab "children"}}active{{end}}">子节点</a>
            <a href="/tasks/{{.Task.ID}}?node={{if .Task.SelectedNode}}{{.Task.SelectedNode.ID}}{{end}}&tab=events" class="{{if eqs .Task.ActiveTab "events"}}active{{end}}">事件</a>
            <a href="/tasks/{{.Task.ID}}?node={{if .Task.SelectedNode}}{{.Task.SelectedNode.ID}}{{end}}&tab=artifacts" class="{{if eqs .Task.ActiveTab "artifacts"}}active{{end}}">产物</a>
          </div>

          {{if eqs .Task.ActiveTab "children"}}
            {{if .Task.SelectedChildren}}
            <div class="simple-list">
              {{range .Task.SelectedChildren}}
              <a class="row" href="/tasks/{{$.Task.ID}}?node={{.ID}}&tab=edit">
                <strong>{{.Title}}</strong>
                <small>{{.Path}}</small>
                <div class="meta">
                  <span class="pill {{statusClass .Status}}">{{.Status}}</span>
                  <span class="pill">{{.Progress}}%</span>
                  <span class="pill">{{.Estimate}}</span>
                </div>
              </a>
              {{end}}
            </div>
            {{else}}
            <div class="empty">当前节点还没有直接子节点。</div>
            {{end}}
          {{else if eqs .Task.ActiveTab "events"}}
            {{if .Task.SelectedEvents}}
            <div class="simple-list">
              {{range .Task.SelectedEvents}}
              <div class="row">
                <strong>{{.Type}}</strong>
                <small>{{if .Message}}{{.Message}}{{else}}无 message{{end}}</small>
                <div class="meta">
                  {{if .Actor}}<span class="pill">{{.Actor}}</span>{{end}}
                  <span class="pill">{{.CreatedAt}}</span>
                </div>
              </div>
              {{end}}
            </div>
            {{else}}
            <div class="empty">当前节点还没有事件。</div>
            {{end}}
          {{else if eqs .Task.ActiveTab "artifacts"}}
            {{if .Task.SelectedArtifacts}}
            <div class="simple-list">
              {{range .Task.SelectedArtifacts}}
              <div class="row">
                <strong>{{if .Title}}{{.Title}}{{else}}{{.ID}}{{end}}</strong>
                <small>{{.URI}}</small>
                <div class="meta">
                  <span class="pill">{{.Kind}}</span>
                  <span class="pill">{{.CreatedAt}}</span>
                </div>
              </div>
              {{end}}
            </div>
            {{else}}
            <div class="empty">当前节点还没有产物。</div>
            {{end}}
          {{else}}
            {{if .Task.SelectedNode}}
            <div class="editor-grid">
              <div class="editor-row">
                <label>标题</label>
                <div class="editor-box">{{.Task.SelectedNode.Title}}</div>
              </div>
              <div class="editor-row">
                <label>路径</label>
                <div class="editor-box">{{.Task.SelectedNode.Path}}</div>
              </div>
              <div class="editor-row">
                <label>指令</label>
                <div class="editor-box">{{if .Task.SelectedNode.Instruction}}{{.Task.SelectedNode.Instruction}}{{else}}无 instruction{{end}}</div>
              </div>
            </div>
            {{else}}
            <div class="empty">选择左侧节点查看详情。</div>
            {{end}}
          {{end}}
        </section>

        <section class="panel card">
          <div class="section-head">
            <h2>任务概览</h2>
            <span>resume 推荐节点</span>
          </div>
          {{if .Task.NextNode}}
          <div class="row">
            <strong>{{.Task.NextNode.Path}} · {{.Task.NextNode.Title}}</strong>
            <small>{{if .Task.NextNode.Instruction}}{{.Task.NextNode.Instruction}}{{else}}无 instruction{{end}}</small>
            <div class="meta">
              <span class="pill {{statusClass .Task.NextNode.Status}}">{{.Task.NextNode.Status}}</span>
              <span class="pill">{{.Task.NextNode.Progress}}%</span>
            </div>
          </div>
          {{else}}
          <div class="empty">当前没有可执行节点，可能任务已完成或被全部阻塞。</div>
          {{end}}
        </section>

        <section class="panel card">
          <div class="section-head">
            <h2>最近事件</h2>
            <span>{{len .Task.Events}} 条</span>
          </div>
          {{if .Task.Events}}
          <div class="list">
            {{range .Task.Events}}
            <div class="row">
              <strong>{{.Type}}</strong>
              <small>{{if .Message}}{{.Message}}{{else}}无 message{{end}}</small>
              <div class="meta">
                {{if .Actor}}<span class="pill">{{.Actor}}</span>{{end}}
                <span class="pill">{{.CreatedAt}}</span>
              </div>
            </div>
            {{end}}
          </div>
          {{else}}
          <div class="empty">还没有事件。</div>
          {{end}}
        </section>

        <section class="panel card">
          <div class="section-head">
            <h2>产物</h2>
            <span>{{len .Task.Artifacts}} 个</span>
          </div>
          {{if .Task.Artifacts}}
          <div class="list">
            {{range .Task.Artifacts}}
            <div class="row">
              <strong>{{if .Title}}{{.Title}}{{else}}{{.ID}}{{end}}</strong>
              <small>{{.URI}}</small>
              <div class="meta">
                <span class="pill">{{.Kind}}</span>
                <span class="pill">{{.CreatedAt}}</span>
              </div>
            </div>
            {{end}}
          </div>
          {{else}}
          <div class="empty">还没有挂载 artifact。</div>
          {{end}}
        </section>
      </div>
    </section>
    {{end}}

    {{if .WorkItems}}
    <section class="section">
      <div class="section-head">
        <h2>可领取工作</h2>
        <span>{{len .WorkItems}} 个候选节点</span>
      </div>
      <div class="list">
        {{range .WorkItems}}
        <a class="row" href="/tasks/{{.TaskID}}">
          <strong>{{.Path}} · {{.Title}}</strong>
          <small>{{.TaskTitle}}</small>
          <div class="meta">
            <span class="pill {{statusClass .Status}}">{{.Status}}</span>
            <span class="pill">{{.UpdatedAt}}</span>
          </div>
        </a>
        {{end}}
      </div>
    </section>
    {{end}}

    {{if .Search}}
    <div class="toolbar">
      <form method="get" action="/search">
        <input type="text" name="q" value="{{.SearchQuery}}" placeholder="搜索任务标题、目标、节点路径或 instruction">
        <button type="submit">搜索</button>
      </form>
    </div>
    <section class="section">
      <div class="section-head">
        <h2>任务结果</h2>
        <span>{{len .Search.Tasks}} 条</span>
      </div>
      {{if .Search.Tasks}}
      <div class="grid task-grid">
        {{range .Search.Tasks}}
        <a class="card" href="/tasks/{{.ID}}">
          <div class="eyebrow">
            <span>{{if .TaskKey}}{{.TaskKey}}{{else}}{{.ID}}{{end}}</span>
            <span>{{.UpdatedAt}}</span>
          </div>
          <h3>{{.Title}}</h3>
          <p>{{if .Goal}}{{.Goal}}{{else}}无 goal{{end}}</p>
          <div class="meta">
            <span class="pill {{statusClass .Status}}">{{.Status}}</span>
            <span class="pill">{{.Percent}}%</span>
          </div>
        </a>
        {{end}}
      </div>
      {{else}}
      <div class="empty">没有匹配的任务。</div>
      {{end}}
    </section>
    <section class="section">
      <div class="section-head">
        <h2>节点结果</h2>
        <span>{{len .Search.Nodes}} 条</span>
      </div>
      {{if .Search.Nodes}}
      <div class="list">
        {{range .Search.Nodes}}
        <a class="row" href="/tasks/{{.TaskID}}">
          <strong>{{.Path}} · {{.Title}}</strong>
          <small>{{.TaskTitle}}</small>
          <div class="meta">
            <span class="pill {{statusClass .Status}}">{{.Status}}</span>
          </div>
        </a>
        {{end}}
      </div>
      {{else}}
      <div class="empty">没有匹配的节点。</div>
      {{end}}
    </section>
    {{end}}
  </div>
</body>
</html>`))

func (a *App) renderUI(w http.ResponseWriter, data uiPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := uiPageTpl.Execute(w, data); err != nil {
		writeErr(w, err)
	}
}

func (a *App) renderTaskListPage(w http.ResponseWriter, r *http.Request, section, status string, includeDeleted bool) {
	tasks, err := a.listTasks(r.Context(), status, r.URL.Query().Get("q"), includeDeleted, false, 100)
	if err != nil {
		writeErr(w, err)
		return
	}
	cards := make([]uiTaskCard, 0, len(tasks))
	for _, task := range tasks {
		cards = append(cards, toUITaskCard(task))
	}
	a.renderUI(w, uiPageData{
		Title:       "Task Tree",
		Section:     section,
		Tasks:       cards,
		SearchQuery: r.URL.Query().Get("q"),
		Flash:       strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:       strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderTaskDetailPage(w http.ResponseWriter, r *http.Request, taskID string) {
	ctx := r.Context()
	task, err := a.getTask(ctx, taskID, false)
	if err != nil {
		writeErr(w, err)
		return
	}
	nodes, err := a.listNodes(ctx, taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	remaining, err := a.getRemaining(ctx, taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	resume, err := a.resumeTask(ctx, taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	eventsWrap, err := a.listEvents(ctx, taskID, "", "", "", 12)
	if err != nil {
		writeErr(w, err)
		return
	}
	artifacts, err := a.listArtifacts(ctx, taskID, nil)
	if err != nil {
		writeErr(w, err)
		return
	}

	detail := &uiTaskDetail{
		ID:         asString(task["id"]),
		TaskKey:    asString(task["task_key"]),
		Title:      asString(task["title"]),
		Goal:       asString(task["goal"]),
		Status:     asString(task["status"]),
		Percent:    percentInt(task["summary_percent"]),
		UpdatedAt:  shortTime(task["updated_at"]),
		Remaining:  int(asFloat(remaining["remaining_nodes"])),
		BlockCount: int(asFloat(remaining["blocked_nodes"])),
		Estimate:   fmt.Sprintf("剩余 %.1fh", asFloat(remaining["remaining_estimate"])),
		NodeCount:  len(nodes),
		ActiveTab:  strings.TrimSpace(r.URL.Query().Get("tab")),
	}
	if detail.ActiveTab == "" {
		detail.ActiveTab = "edit"
	}
	nodeCards := make([]uiNodeCard, 0, len(nodes))
	for _, node := range nodes {
		nodeCards = append(nodeCards, toUINodeCard(node))
	}
	sort.Slice(nodeCards, func(i, j int) bool { return nodeCards[i].Path < nodeCards[j].Path })
	selectedNodeID := strings.TrimSpace(r.URL.Query().Get("node"))
	if selectedNodeID == "" {
		if nextRaw, ok := resume["next_node"].(map[string]any); ok {
			if nextNode, ok := nextRaw["node"].(map[string]any); ok && nextNode != nil {
				selectedNodeID = asString(nextNode["node_id"])
			}
		}
	}
	if selectedNodeID == "" && len(nodeCards) > 0 {
		selectedNodeID = nodeCards[0].ID
	}
	detail.NodeTrees = buildUINodeForest(nodeCards, selectedNodeID)
	for _, node := range nodeCards {
		if node.ID == selectedNodeID {
			tmp := node
			detail.SelectedNode = &tmp
			break
		}
	}
	if detail.SelectedNode != nil {
		for _, node := range nodeCards {
			if node.ParentNodeID == detail.SelectedNode.ID {
				detail.SelectedChildren = append(detail.SelectedChildren, node)
			}
		}
		eventsWrap, err := a.listEvents(ctx, taskID, detail.SelectedNode.ID, "", "", 20)
		if err == nil {
			if items, ok := eventsWrap["items"].([]jsonMap); ok {
				for _, item := range items {
					detail.SelectedEvents = append(detail.SelectedEvents, toUIEvent(item))
				}
			} else if items, ok := eventsWrap["items"].([]any); ok {
				for _, raw := range items {
					if item, ok := raw.(map[string]any); ok {
						detail.SelectedEvents = append(detail.SelectedEvents, toUIEvent(item))
					}
				}
			}
		}
		nodeID := detail.SelectedNode.ID
		nodeArtifacts, err := a.listArtifacts(ctx, taskID, &nodeID)
		if err == nil {
			for _, item := range nodeArtifacts {
				detail.SelectedArtifacts = append(detail.SelectedArtifacts, uiArtifactCard{
					ID:        asString(item["id"]),
					Title:     asString(item["title"]),
					Kind:      asString(item["kind"]),
					URI:       asString(item["uri"]),
					CreatedAt: shortTime(item["created_at"]),
				})
			}
		}
	}

	if nextRaw, ok := resume["next_node"].(map[string]any); ok {
		if nextNode, ok := nextRaw["node"].(map[string]any); ok && nextNode != nil {
			card := toUINodeCard(nextNode)
			detail.NextNode = &card
		}
	}
	if items, ok := eventsWrap["items"].([]jsonMap); ok {
		for _, item := range items {
			detail.Events = append(detail.Events, toUIEvent(item))
		}
	} else if items, ok := eventsWrap["items"].([]any); ok {
		for _, raw := range items {
			if item, ok := raw.(map[string]any); ok {
				detail.Events = append(detail.Events, toUIEvent(item))
			}
		}
	}
	for _, item := range artifacts {
		detail.Artifacts = append(detail.Artifacts, uiArtifactCard{
			ID:        asString(item["id"]),
			Title:     asString(item["title"]),
			Kind:      asString(item["kind"]),
			URI:       asString(item["uri"]),
			CreatedAt: shortTime(item["created_at"]),
		})
	}

	a.renderUI(w, uiPageData{
		Title:   detail.Title + " · Task Tree",
		Section: "任务详情",
		Task:    detail,
		Flash:   strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:   strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderWorkPage(w http.ResponseWriter, r *http.Request) {
	items, err := a.listWorkItems(r.Context(), "ready,running,blocked", true, 60)
	if err != nil {
		writeErr(w, err)
		return
	}
	rows := make([]uiWorkItem, 0, len(items))
	for _, item := range items {
		rows = append(rows, uiWorkItem{
			TaskID:    asString(item["task_id"]),
			TaskTitle: asString(item["task_title"]),
			NodeID:    asString(item["id"]),
			Path:      asString(item["path"]),
			Title:     asString(item["title"]),
			Status:    asString(item["status"]),
			UpdatedAt: shortTime(item["updated_at"]),
		})
	}
	a.renderUI(w, uiPageData{
		Title:     "可领取工作 · Task Tree",
		Section:   "可领取工作",
		WorkItems: rows,
		Flash:     strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:     strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderSearchPage(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	result, err := a.search(r.Context(), q, "all", 40)
	if err != nil {
		writeErr(w, err)
		return
	}
	view := &uiSearchResult{}
	if tasks, ok := result["tasks"].([]jsonMap); ok {
		for _, task := range tasks {
			view.Tasks = append(view.Tasks, toUITaskCard(task))
		}
	} else if tasks, ok := result["tasks"].([]any); ok {
		for _, raw := range tasks {
			if task, ok := raw.(map[string]any); ok {
				view.Tasks = append(view.Tasks, toUITaskCard(task))
			}
		}
	}
	if nodes, ok := result["nodes"].([]jsonMap); ok {
		for _, node := range nodes {
			view.Nodes = append(view.Nodes, toUISearchNode(node))
		}
	} else if nodes, ok := result["nodes"].([]any); ok {
		for _, raw := range nodes {
			if node, ok := raw.(map[string]any); ok {
				view.Nodes = append(view.Nodes, toUISearchNode(node))
			}
		}
	}
	a.renderUI(w, uiPageData{
		Title:       "搜索 · Task Tree",
		Section:     "搜索",
		SearchQuery: q,
		Search:      view,
		Flash:       strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:       strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func toUITaskCard(task map[string]any) uiTaskCard {
	return uiTaskCard{
		ID:        asString(task["id"]),
		TaskKey:   asString(task["task_key"]),
		Title:     asString(task["title"]),
		Goal:      asString(task["goal"]),
		Status:    asString(task["status"]),
		Percent:   percentInt(task["summary_percent"]),
		UpdatedAt: shortTime(task["updated_at"]),
		Deleted:   asString(task["deleted_at"]) != "",
	}
}

func toUINodeCard(node map[string]any) uiNodeCard {
	return uiNodeCard{
		ID:           asString(node["id"]),
		ParentNodeID: asString(node["parent_node_id"]),
		Path:         asString(node["path"]),
		Title:        asString(node["title"]),
		Kind:         asString(node["kind"]),
		Status:       asString(node["status"]),
		Progress:     percentInt(node["progress"]),
		Estimate:     fmt.Sprintf("%.1fh", asFloat(node["estimate"])),
		Instruction:  singleLine(asString(node["instruction"])),
		Depth:        int(asFloat(node["depth"])),
	}
}

func buildUINodeForest(nodes []uiNodeCard, selectedNodeID string) []uiNodeTree {
	byParent := map[string][]uiNodeCard{}
	for _, node := range nodes {
		byParent[node.ParentNodeID] = append(byParent[node.ParentNodeID], node)
	}
	var walk func(parentID string) []uiNodeTree
	walk = func(parentID string) []uiNodeTree {
		items := byParent[parentID]
		out := make([]uiNodeTree, 0, len(items))
		for _, node := range items {
			children := walk(node.ID)
			node.HasChildren = len(children) > 0
			node.ChildCount = len(children)
			node.IsSelected = node.ID == selectedNodeID
			out = append(out, uiNodeTree{
				Node:     node,
				Children: children,
			})
		}
		return out
	}
	return walk("")
}

func toUIEvent(item map[string]any) uiEventCard {
	actor := strings.Trim(strings.Join([]string{asString(item["actor_type"]), asString(item["actor_id"])}, " "), " ")
	return uiEventCard{
		Type:      asString(item["type"]),
		Message:   singleLine(asString(item["message"])),
		Actor:     actor,
		CreatedAt: shortTime(item["created_at"]),
	}
}

func toUISearchNode(node map[string]any) uiSearchNode {
	return uiSearchNode{
		TaskID:    asString(node["task_id"]),
		TaskTitle: asString(node["task_title"]),
		NodeID:    asString(node["id"]),
		Path:      asString(node["path"]),
		Title:     asString(node["title"]),
		Status:    asString(node["status"]),
	}
}

func percentInt(v any) int {
	n := int(asFloat(v) * 100)
	if asFloat(v) > 1 {
		n = int(asFloat(v))
	}
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func shortTime(v any) string {
	s := asString(v)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "T", " ")
	s = strings.TrimSuffix(s, "Z")
	if len(s) >= 16 {
		return s[:16]
	}
	return s
}

func singleLine(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	parts := strings.Fields(strings.ReplaceAll(v, "\n", " "))
	return strings.Join(parts, " ")
}


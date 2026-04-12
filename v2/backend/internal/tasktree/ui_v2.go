package tasktree

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type workspacePageData struct {
	Title        string
	Section      string
	Flash        string
	Error        string
	Summary      *workspaceSummary
	Tasks        []workspaceTaskCard
	Task         *workspaceTaskDetail
	WorkItems    []workspaceWorkItem
	SearchQuery  string
	Search       *workspaceSearchResult
	StatusFilter string
	AIEnabled    bool
}

type workspaceSummary struct {
	Total    int
	Ready    int
	Running  int
	Blocked  int
	Paused   int
	Done     int
	Canceled int
	Closed   int
}

type workspaceTaskCard struct {
	ID            string
	TaskKey       string
	Title         string
	Goal          string
	Status        string
	Result        string
	Percent       int
	UpdatedAt     string
	Deleted       bool
	Remaining     int
	BlockCount    int
	PausedCount   int
	NextNodePath  string
	NextNodeTitle string
}

type workspaceTaskDetail struct {
	ID                  string
	TaskKey             string
	Title               string
	Goal                string
	Status              string
	Result              string
	Percent             int
	UpdatedAt           string
	Remaining           int
	BlockCount          int
	PausedCount         int
	CanceledCount       int
	Estimate            string
	NodeCount           int
	NextNode            *workspaceNodeCard
	NodeTrees           []workspaceNodeTree
	Events              []workspaceEventCard
	Artifacts           []workspaceArtifactCard
	ActiveTab           string
	SelectedNode        *workspaceNodeCard
	SelectedChildren    []workspaceNodeCard
	SelectedEvents      []workspaceEventCard
	SelectedArtifacts   []workspaceArtifactCard
	SelectedIncludeDesc bool
	CanPause            bool
	CanReopen           bool
	CanCancel           bool
}

type workspaceNodeCard struct {
	ID                string
	TaskID            string
	ParentNodeID      string
	Path              string
	PathLeaf          string
	Title             string
	Kind              string
	Status            string
	Result            string
	Progress          int
	Estimate          string
	EstimateValue     string
	Instruction       string
	Acceptance        []string
	Depth             int
	HasChildren       bool
	ChildCount        int
	IsSelected        bool
	Expanded          bool
	ClaimedBy         string
	LeaseUntil        string
	IsClaimed         bool
	CanClaim          bool
	CanRelease        bool
	CanProgress       bool
	CanComplete       bool
	CanBlock          bool
	CanUnblock        bool
	CanPause          bool
	CanReopen         bool
	CanCancel         bool
	CanCreateChildren bool
	CanCreateSibling  bool
	CanConvertToLeaf  bool
}

type workspaceNodeTree struct {
	Node     workspaceNodeCard
	Children []workspaceNodeTree
}

type workspaceEventCard struct {
	ID           string
	NodeID       string
	Type         string
	Message      string
	Actor        string
	CreatedAt    string
	CreatedAtRaw string
	Warnings     []string
	HasWarnings  bool
}

type workspaceArtifactCard struct {
	ID           string
	Title        string
	Kind         string
	URI          string
	CreatedAt    string
	DownloadURL  string
	Downloadable bool
}

type workspaceWorkItem struct {
	TaskID    string
	TaskTitle string
	NodeID    string
	Path      string
	Chain     string
	Title     string
	Status    string
	UpdatedAt string
}

type workspaceSearchResult struct {
	Tasks []workspaceTaskCard
	Nodes []workspaceSearchNode
}

type workspaceSearchNode struct {
	TaskID    string
	TaskTitle string
	NodeID    string
	Path      string
	Title     string
	Status    string
	Result    string
}

var workspaceFuncMap = template.FuncMap{
	"statusClass":    workspaceStatusClass,
	"statusLabel":    workspaceStatusLabel,
	"stateLabel":     workspaceStateLabel,
	"eqs":            func(a, b string) bool { return a == b },
	"criteriaText":   func(items []string) string { return strings.Join(items, "\n") },
	"excerpt":        workspaceExcerpt,
	"eventTypeLabel": workspaceEventTypeLabel,
}

func loadWorkspaceTemplate() (*template.Template, error) {
	paths := []string{
		resolveUpwardPath(filepath.Join("web", "templates", "workspace_part1.html")),
		resolveUpwardPath(filepath.Join("web", "templates", "workspace_part2.html")),
		resolveUpwardPath(filepath.Join("web", "templates", "workspace_part3.html")),
	}
	var builder strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		builder.Write(content)
	}
	return template.New("workspace").Funcs(workspaceFuncMap).Parse(builder.String())
}

func (a *App) renderUIV2(w http.ResponseWriter, data workspacePageData) {
	data.AIEnabled = aiEnabled()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl, err := loadWorkspaceTemplate()
	if err != nil {
		a.renderUIV2Fallback(w, data)
		return
	}
	if err := tpl.Execute(w, data); err != nil {
		a.renderUIV2Fallback(w, data)
	}
}

func (a *App) renderUIV2Fallback(w http.ResponseWriter, data workspacePageData) {
	const fallback = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    body { font-family: "Segoe UI","PingFang SC","Microsoft YaHei",sans-serif; margin: 0; background: #f6f2ea; color: #181818; }
    main { max-width: 1080px; margin: 0 auto; padding: 32px 24px 48px; }
    .hero, .panel { background: #fff; border: 1px solid #e9dfd2; border-radius: 20px; padding: 24px; box-shadow: 0 12px 30px rgba(0,0,0,.06); }
    .hero h1 { margin: 0 0 10px; font-size: 42px; letter-spacing: -0.03em; }
    .muted { color: #6a645d; }
    .grid { display: grid; gap: 16px; margin-top: 20px; }
    .task { padding: 16px 18px; border-radius: 16px; background: #fff; border: 1px solid #eee2d3; }
    .task strong { display: block; margin-bottom: 6px; }
  </style>
</head>
<body>
  <main id="app">
    <section class="hero">
      <h1>Task Tree</h1>
      <p class="muted">{{.Title}}</p>
      {{if .Task}}<p class="muted">当前任务：{{.Task.Title}}</p>{{end}}
    </section>
    <section class="grid">
      {{if .Task}}
      <div class="panel">
        <strong>{{.Task.Title}}</strong>
        <div class="muted">状态：{{.Task.Status}} · 进度：{{.Task.Percent}}%</div>
      </div>
      {{end}}
      {{range .Tasks}}
      <div class="task">
        <strong>{{.Title}}</strong>
        <div class="muted">{{.Status}} · {{.Percent}}%</div>
      </div>
      {{end}}
      {{if and (not .Task) (eq (len .Tasks) 0)}}
      <div class="panel">
        <div class="muted">V2 模板文件暂未落地，当前回退到内置页面。</div>
      </div>
      {{end}}
    </section>
  </main>
</body>
</html>`
	tpl := template.Must(template.New("workspace-fallback").Parse(fallback))
	_ = tpl.Execute(w, data)
}

func (a *App) renderTaskListPageV2(w http.ResponseWriter, r *http.Request, section, status string, includeDeleted bool) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" {
		statusFilter = status
	}
	tasks, err := a.listTasks(r.Context(), statusFilter, q, includeDeleted, includeDeleted, 80)
	if err != nil {
		writeErr(w, err)
		return
	}
	cards := make([]workspaceTaskCard, 0, len(tasks))
	summary := &workspaceSummary{Total: len(tasks)}
	for _, task := range tasks {
		card := workspaceTaskCardFromMap(task)
		switch card.Status {
		case "ready":
			summary.Ready++
		case "running":
			summary.Running++
		case "blocked":
			summary.Blocked++
		case "paused":
			summary.Paused++
		case "done":
			summary.Done++
		case "canceled":
			summary.Canceled++
		case "closed":
			summary.Closed++
		}
		if !includeDeleted {
			if remaining, err := a.getRemaining(r.Context(), card.ID); err == nil {
				card.Remaining = int(asFloat(remaining["remaining_nodes"]))
				card.BlockCount = int(asFloat(remaining["blocked_nodes"]))
				card.PausedCount = int(asFloat(remaining["paused_nodes"]))
			}
			if resume, err := a.resumeTask(r.Context(), card.ID); err == nil {
				card.NextNodePath, card.NextNodeTitle = workspaceNextNodePreview(resume["next_node"])
			}
		}
		cards = append(cards, card)
	}
	if len(cards) == 0 {
		summary = nil
	}
	a.renderUIV2(w, workspacePageData{
		Title:        "Task Tree",
		Section:      section,
		Summary:      summary,
		Tasks:        cards,
		SearchQuery:  q,
		StatusFilter: statusFilter,
		Flash:        strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:        strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderTaskDetailPageV2(w http.ResponseWriter, r *http.Request, taskID string) {
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

	detail := &workspaceTaskDetail{
		ID:            asString(task["id"]),
		TaskKey:       asString(task["task_key"]),
		Title:         asString(task["title"]),
		Goal:          asString(task["goal"]),
		Status:        asString(task["status"]),
		Result:        asString(task["result"]),
		Percent:       workspacePercentInt(task["summary_percent"]),
		UpdatedAt:     workspaceShortTime(task["updated_at"]),
		Remaining:     int(asFloat(remaining["remaining_nodes"])),
		BlockCount:    int(asFloat(remaining["blocked_nodes"])),
		PausedCount:   int(asFloat(remaining["paused_nodes"])),
		CanceledCount: int(asFloat(remaining["canceled_nodes"])),
		Estimate:      fmt.Sprintf("剩余 %.1fh", asFloat(remaining["remaining_estimate"])),
		NodeCount:     len(nodes),
		ActiveTab:     strings.TrimSpace(r.URL.Query().Get("tab")),
	}
	if detail.ActiveTab == "" {
		detail.ActiveTab = "edit"
	}
	switch detail.Status {
	case "ready", "running", "blocked":
		detail.CanPause = true
		detail.CanCancel = true
	case "paused", "canceled", "closed":
		detail.CanReopen = true
		if detail.Status == "paused" {
			detail.CanCancel = true
		}
	}

	nodeCards := make([]workspaceNodeCard, 0, len(nodes))
	for _, node := range nodes {
		nodeCards = append(nodeCards, workspaceNodeCardFromMap(node))
	}
	sort.Slice(nodeCards, func(i, j int) bool { return nodeCards[i].Path < nodeCards[j].Path })
	workspaceAnnotateNodeCards(nodeCards)

	selectedNodeID := strings.TrimSpace(r.URL.Query().Get("node"))
	if selectedNodeID == "" {
		selectedNodeID = workspaceNextNodeID(resume["next_node"])
	}
	if selectedNodeID == "" && len(nodeCards) > 0 {
		selectedNodeID = nodeCards[0].ID
	}
	detail.NodeTrees = workspaceBuildNodeForest(nodeCards, selectedNodeID)
	for _, node := range nodeCards {
		if node.ID == selectedNodeID {
			copyNode := node
			detail.SelectedNode = &copyNode
			break
		}
	}
	if detail.SelectedNode != nil {
		for _, node := range nodeCards {
			if node.ParentNodeID == detail.SelectedNode.ID {
				detail.SelectedChildren = append(detail.SelectedChildren, node)
			}
		}
		nodeID := detail.SelectedNode.ID
		includeDescendants := detail.SelectedNode.HasChildren
		detail.SelectedIncludeDesc = includeDescendants
		selectedEvents, err := a.listEventsScoped(ctx, taskID, nodeID, includeDescendants, "", "", 20, eventListOptions{})
		if err == nil {
			detail.SelectedEvents = workspaceEventsFromMaps(workspaceAsItems(selectedEvents["items"]))
		}
		selectedArtifacts, err := a.listArtifacts(ctx, taskID, &nodeID)
		if err == nil {
			detail.SelectedArtifacts = workspaceArtifactsFromMaps(selectedArtifacts)
		}
	}
	if detail.SelectedNode == nil {
		detail.SelectedEvents = workspaceEventsFromMaps(workspaceAsItems(eventsWrap["items"]))
		detail.SelectedArtifacts = workspaceArtifactsFromMaps(artifacts)
	}
	if nextRaw := asAnyMap(resume["next_node"]); nextRaw != nil {
		if nextNode := asAnyMap(nextRaw["node"]); nextNode != nil {
			card := workspaceNodeCardFromMap(nextNode)
			detail.NextNode = &card
		}
	}
	detail.Events = workspaceEventsFromMaps(workspaceAsItems(eventsWrap["items"]))
	detail.Artifacts = workspaceArtifactsFromMaps(artifacts)

	a.renderUIV2(w, workspacePageData{
		Title:   detail.Title + " · Task Tree",
		Section: "任务详情",
		Task:    detail,
		Flash:   strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:   strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderWorkPageV2(w http.ResponseWriter, r *http.Request) {
	items, err := a.listWorkItems(r.Context(), "ready,running", true, 60)
	if err != nil {
		writeErr(w, err)
		return
	}
	rows := make([]workspaceWorkItem, 0, len(items))
	for _, item := range items {
		taskTitle := asString(item["task_title"])
		chain := workspaceBreadcrumb(asString(item["path"]))
		if chain != "" && taskTitle != "" {
			chain = taskTitle + " / " + chain
		} else if taskTitle != "" {
			chain = taskTitle
		}
		rows = append(rows, workspaceWorkItem{
			TaskID:    asString(item["task_id"]),
			TaskTitle: taskTitle,
			NodeID:    asString(item["id"]),
			Path:      asString(item["path"]),
			Chain:     chain,
			Title:     asString(item["title"]),
			Status:    asString(item["status"]),
			UpdatedAt: workspaceShortTime(item["updated_at"]),
		})
	}
	a.renderUIV2(w, workspacePageData{
		Title:     "可领取工作 · Task Tree",
		Section:   "可领取工作",
		WorkItems: rows,
		Flash:     strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:     strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func (a *App) renderSearchPageV2(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	result, err := a.search(r.Context(), q, "all", 40)
	if err != nil {
		writeErr(w, err)
		return
	}
	view := &workspaceSearchResult{}
	for _, task := range workspaceAsItems(result["tasks"]) {
		card := workspaceTaskCardFromMap(task)
		if remaining, err := a.getRemaining(r.Context(), card.ID); err == nil {
			card.Remaining = int(asFloat(remaining["remaining_nodes"]))
		}
		view.Tasks = append(view.Tasks, card)
	}
	for _, node := range workspaceAsItems(result["nodes"]) {
		view.Nodes = append(view.Nodes, workspaceSearchNode{
			TaskID:    asString(node["task_id"]),
			TaskTitle: asString(node["task_title"]),
			NodeID:    asString(node["id"]),
			Path:      asString(node["path"]),
			Title:     asString(node["title"]),
			Status:    asString(node["status"]),
			Result:    asString(node["result"]),
		})
	}
	a.renderUIV2(w, workspacePageData{
		Title:       "搜索 · Task Tree",
		Section:     "搜索",
		SearchQuery: q,
		Search:      view,
		Flash:       strings.TrimSpace(r.URL.Query().Get("flash")),
		Error:       strings.TrimSpace(r.URL.Query().Get("error")),
	})
}

func workspaceTaskCardFromMap(task map[string]any) workspaceTaskCard {
	return workspaceTaskCard{
		ID:        asString(task["id"]),
		TaskKey:   asString(task["task_key"]),
		Title:     asString(task["title"]),
		Goal:      strings.TrimSpace(asString(task["goal"])),
		Status:    asString(task["status"]),
		Result:    asString(task["result"]),
		Percent:   workspacePercentInt(task["summary_percent"]),
		UpdatedAt: workspaceShortTime(task["updated_at"]),
		Deleted:   asString(task["deleted_at"]) != "",
	}
}

func workspaceNodeCardFromMap(node map[string]any) workspaceNodeCard {
	status := asString(node["status"])
	result := asString(node["result"])
	kind := asString(node["kind"])
	claimedBy := strings.Trim(strings.Join([]string{asString(node["claimed_by_type"]), asString(node["claimed_by_id"])}, "/"), "/")
	pathStr := asString(node["path"])
	pathLeaf := pathStr
	if idx := strings.LastIndex(pathStr, "/"); idx >= 0 {
		pathLeaf = pathStr[idx+1:]
	}
	card := workspaceNodeCard{
		ID:                asString(node["id"]),
		TaskID:            asString(node["task_id"]),
		ParentNodeID:      asString(node["parent_node_id"]),
		Path:              pathStr,
		PathLeaf:          pathLeaf,
		Title:             asString(node["title"]),
		Kind:              kind,
		Status:            status,
		Result:            result,
		Progress:          workspacePercentInt(node["progress"]),
		Estimate:          fmt.Sprintf("%.1fh", asFloat(node["estimate"])),
		EstimateValue:     workspaceTrimFloat(asFloat(node["estimate"])),
		Instruction:       strings.TrimSpace(asString(node["instruction"])),
		Acceptance:        workspaceStringSlice(node["acceptance_criteria"]),
		Depth:             int(asFloat(node["depth"])),
		ClaimedBy:         claimedBy,
		LeaseUntil:        workspaceShortTime(node["lease_until"]),
		IsClaimed:         leaseActive(node),
		CanCreateChildren: kind != "linked_task",
		CanCreateSibling:  kind != "linked_task",
	}
	if card.ID == "" {
		card.ID = asString(node["node_id"])
	}
	if kind == "leaf" {
		isClosed := result == "done" || result == "canceled" || status == "closed"
		card.CanClaim = !isClosed && status != "paused" && status != "blocked" && !card.IsClaimed
		card.CanRelease = card.IsClaimed
		card.CanProgress = status != "blocked" && status != "paused" && !isClosed
		card.CanComplete = !isClosed
		card.CanBlock = status != "blocked" && !isClosed
		card.CanUnblock = status == "blocked"
		card.CanPause = !isClosed && status != "paused"
		card.CanReopen = status == "paused" || result == "done" || result == "canceled"
		card.CanCancel = result == "" && status != "closed"
	}
	return card
}

func workspaceBuildNodeForest(nodes []workspaceNodeCard, selectedNodeID string) []workspaceNodeTree {
	byParent := map[string][]workspaceNodeCard{}
	byID := map[string]workspaceNodeCard{}
	for _, node := range nodes {
		byParent[node.ParentNodeID] = append(byParent[node.ParentNodeID], node)
		byID[node.ID] = node
	}
	ancestors := map[string]bool{}
	cur := selectedNodeID
	for cur != "" {
		ancestors[cur] = true
		cur = byID[cur].ParentNodeID
	}
	var walk func(parentID string) []workspaceNodeTree
	walk = func(parentID string) []workspaceNodeTree {
		items := byParent[parentID]
		out := make([]workspaceNodeTree, 0, len(items))
		for _, node := range items {
			children := walk(node.ID)
			node.HasChildren = len(children) > 0
			node.ChildCount = len(children)
			node.IsSelected = node.ID == selectedNodeID
			node.Expanded = node.Depth == 0 || ancestors[node.ID]
			out = append(out, workspaceNodeTree{Node: node, Children: children})
		}
		return out
	}
	return walk("")
}

func workspaceAnnotateNodeCards(nodes []workspaceNodeCard) {
	childCounts := map[string]int{}
	for _, node := range nodes {
		if node.ParentNodeID != "" {
			childCounts[node.ParentNodeID]++
		}
	}
	for i := range nodes {
		nodes[i].ChildCount = childCounts[nodes[i].ID]
		nodes[i].HasChildren = nodes[i].ChildCount > 0
		nodes[i].CanConvertToLeaf = nodes[i].Kind == "group" && nodes[i].ChildCount == 0
	}
}

func workspaceEventsFromMaps(items []jsonMap) []workspaceEventCard {
	out := make([]workspaceEventCard, 0, len(items))
	for _, item := range items {
		actor := strings.Trim(strings.Join([]string{asString(item["actor_type"]), asString(item["actor_id"])}, " "), " ")
		payload, _ := item["payload"].(map[string]any)
		warnings := workspaceStringSlice(payload["warnings"])
		out = append(out, workspaceEventCard{
			ID:           asString(item["id"]),
			NodeID:       asString(item["node_id"]),
			Type:         asString(item["type"]),
			Message:      strings.TrimSpace(asString(item["message"])),
			Actor:        actor,
			CreatedAt:    workspaceShortTime(item["created_at"]),
			CreatedAtRaw: asString(item["created_at"]),
			Warnings:     warnings,
			HasWarnings:  len(warnings) > 0,
		})
	}
	return out
}

func workspaceArtifactsFromMaps(items []jsonMap) []workspaceArtifactCard {
	out := make([]workspaceArtifactCard, 0, len(items))
	for _, item := range items {
		out = append(out, workspaceArtifactCard{
			ID:           asString(item["id"]),
			Title:        asString(item["title"]),
			Kind:         asString(item["kind"]),
			URI:          asString(item["uri"]),
			CreatedAt:    workspaceShortTime(item["created_at"]),
			Downloadable: strings.HasPrefix(asString(item["uri"]), "local://"),
			DownloadURL:  "/v1/artifacts/" + asString(item["id"]) + "/download",
		})
	}
	return out
}

func workspaceAsItems(v any) []jsonMap {
	switch items := v.(type) {
	case []jsonMap:
		return items
	case []map[string]any:
		out := make([]jsonMap, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		return out
	case []any:
		out := make([]jsonMap, 0, len(items))
		for _, raw := range items {
			if item, ok := raw.(map[string]any); ok {
				out = append(out, item)
			}
		}
		return out
	default:
		return nil
	}
}

func workspaceStringSlice(v any) []string {
	switch items := v.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if s := strings.TrimSpace(asString(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func workspaceNextNodeID(v any) string {
	if node := asAnyMap(v); node != nil {
		if inner := asAnyMap(node["node"]); inner != nil {
			return asString(inner["node_id"])
		}
	}
	return ""
}

func workspaceNextNodePreview(v any) (string, string) {
	if node := asAnyMap(v); node != nil {
		if inner := asAnyMap(node["node"]); inner != nil {
			return asString(inner["path"]), asString(inner["title"])
		}
	}
	return "", ""
}

func workspaceStatusLabel(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "ready":
		return "就绪"
	case "running":
		return "进行中"
	case "blocked":
		return "阻塞"
	case "paused":
		return "暂停"
	case "done":
		return "完成"
	case "canceled":
		return "已取消"
	case "closed":
		return "已关闭"
	default:
		if v == "" {
			return "未知"
		}
		return v
	}
}

func workspaceStatusClass(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
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
}

func workspaceStateLabel(status, result string) string {
	if strings.EqualFold(strings.TrimSpace(result), "mixed") {
		return "混合关闭"
	}
	return workspaceStatusLabel(status)
}

func workspaceExcerpt(v string, limit int) string {
	v = strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(v, "\n", " ")), " "))
	if v == "" {
		return ""
	}
	runes := []rune(v)
	if len(runes) <= limit {
		return v
	}
	return string(runes[:limit]) + "…"
}

func workspaceBreadcrumb(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, " / ")
}

func workspaceTrimFloat(v float64) string {
	text := fmt.Sprintf("%.1f", v)
	return strings.TrimSuffix(strings.TrimSuffix(text, "0"), ".")
}

func workspacePercentInt(v any) int {
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

func workspaceShortTime(v any) string {
	s := asString(v)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "T", " ")
	s = strings.TrimSuffix(s, "Z")
	short := s
	if len(short) >= 16 {
		short = short[:16]
	}
	// try relative time
	rel := workspaceRelTimePart(short)
	if rel != "" {
		return rel + " · " + short[11:16] // "3 分钟前 · 06:01"
	}
	return short
}

// workspaceRelTimePart parses "2006-01-02 15:04" and returns a relative label for past events.
func workspaceRelTimePart(short string) string {
	if len(short) < 16 {
		return ""
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", short[:16], time.UTC)
	if err != nil {
		return ""
	}
	diff := time.Since(t)
	if diff <= 0 {
		return "" // future timestamps (e.g. lease_until) — show absolute
	}
	switch {
	case diff < 2*time.Minute:
		return "刚刚"
	case diff < 60*time.Minute:
		return fmt.Sprintf("%d 分钟前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d 天前", int(diff.Hours()/24))
	default:
		return ""
	}
}

func workspaceEventTypeLabel(v string) string {
	key := strings.ToLower(strings.TrimSpace(v))
	labels := map[string]string{
		// task events
		"task_created":  "任务创建",
		"task_updated":  "任务更新",
		"task_deleted":  "任务删除",
		"task_restored": "任务恢复",
		"task_paused":   "任务暂停",
		"task_reopened": "任务恢复",
		"task_canceled": "任务取消",
		// node lifecycle
		"node_created": "节点创建",
		"node_retyped": "节点转型",
		"node_updated": "节点更新",
		"progress":     "进度更新",
		"complete":     "完成",
		"claim":        "已领取",
		"release":      "已释放",
		"blocked":      "已阻塞",
		"unblocked":    "解除阻塞",
		"paused":       "暂停",
		"reopened":     "重开",
		"canceled":     "已取消",
		// artifacts
		"artifact": "产物",
		// lease
		"lease_sweep": "租约清扫",
	}
	if label, ok := labels[key]; ok {
		return label
	}
	if v == "" {
		return "事件"
	}
	return v
}

package tasktree

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSmokeFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "smoke.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "重构订单模块", "task_key": "A", "goal": "拆 service 补测试"})
	tid := stringValue(task["id"])

	n1 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"title": "梳理接口", "node_key": "1", "estimate": 1})
	n2 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"title": "补回归测试", "node_key": "2", "estimate": 2})
	n21 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"parent_node_id": n2["id"], "title": "happy path", "node_key": "1"})
	n22 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"parent_node_id": n2["id"], "title": "rollback", "node_key": "2"})
	n221 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"parent_node_id": n22["id"], "title": "nested-1", "node_key": "1"})
	n2211 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"parent_node_id": n221["id"], "title": "nested-2", "node_key": "1"})
	n22111 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes", map[string]any{"parent_node_id": n2211["id"], "title": "nested-3", "node_key": "1"})
	if !strings.Contains(stringValue(n22111["path"]), "A/2/2/1/1/1") {
		t.Fatalf("deep node path = %v", n22111["path"])
	}
	allNodesWrap := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes")
	allNodes, _ := allNodesWrap["items"].([]any)
	if len(allNodes) < 7 {
		t.Fatalf("list nodes len = %d", len(allNodes))
	}

	progress := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n1["id"])+"/progress", map[string]any{"delta_progress": 0.5, "message": "half done", "idempotency_key": "k1"})
	if got := progress["progress"].(float64); got != 0.5 {
		t.Fatalf("progress = %v", got)
	}
	progressDup := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n1["id"])+"/progress", map[string]any{"delta_progress": 0.5, "message": "dup", "idempotency_key": "k1"})
	if got := progressDup["progress"].(float64); got != 0.5 {
		t.Fatalf("dup progress = %v", got)
	}

	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n1["id"])+"/complete", map[string]any{"message": "done"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n21["id"])+"/complete", map[string]any{})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n22111["id"])+"/complete", map[string]any{})

	groupNode := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n2["id"]))
	if stringValue(groupNode["status"]) != "done" {
		t.Fatalf("group status = %v", groupNode["status"])
	}

	taskAfter := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid)
	if stringValue(taskAfter["status"]) != "done" {
		t.Fatalf("task status = %v", taskAfter["status"])
	}

	remaining := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/remaining")
	if remaining["remaining_nodes"].(float64) != 0 {
		t.Fatalf("remaining = %v", remaining["remaining_nodes"])
	}

	resumeCtx := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes/"+stringValue(n2["id"])+"/resume-context")
	if resumeCtx["task"] == nil || resumeCtx["node"] == nil {
		t.Fatal("resume context missing")
	}

	tidc := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "claim 测试"})["id"])
	nc := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/nodes", map[string]any{"title": "可领取节点"})
	claim := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/nodes/"+stringValue(nc["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-1"}, "lease_seconds": 60})
	if stringValue(claim["claimed_by_id"]) != "worker-1" {
		t.Fatalf("claim = %v", claim["claimed_by_id"])
	}
	release := postNoBodyJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/nodes/"+stringValue(nc["id"])+"/release")
	if release["claimed_by_id"] != nil {
		t.Fatalf("release claimed_by_id = %v", release["claimed_by_id"])
	}

	art := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/artifacts", map[string]any{"node_id": nc["id"], "kind": "file", "title": "测试产物", "uri": "file:///tmp/out.md"})
	if stringValue(art["id"]) == "" {
		t.Fatal("artifact missing id")
	}
	artsWrap := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/artifacts")
	arts, _ := artsWrap["items"].([]any)
	if len(arts) != 1 {
		t.Fatalf("artifacts len = %d", len(arts))
	}

	up := uploadFile(t, server.URL+"/v1/tasks/"+tidc+"/artifacts/upload", "file", "out.txt", []byte("hello artifact"), map[string]string{"node_id": stringValue(nc["id"]), "kind": "file"})
	downloadURL := server.URL + "/v1/artifacts/" + stringValue(up["id"]) + "/download"
	resp, err := http.Get(downloadURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello artifact" {
		t.Fatalf("download = %q", string(body))
	}

	_ = getJSON[[]map[string]any](t, server.URL+"/v1/work-items?status=ready")
	_ = getJSON[map[string]any](t, server.URL+"/v1/search?q=重构")
	_ = getJSON[map[string]any](t, server.URL+"/v1/events?task_id="+tid+"&limit=3")
	_ = getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/resume")
	_ = getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(nc["id"]))

	updatedTask := patchJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid, map[string]any{
		"title": "重构订单模块 v2",
		"goal":  "拆 service 补测试并收敛 UI",
	})
	if stringValue(updatedTask["title"]) != "重构订单模块 v2" {
		t.Fatalf("patched task title = %v", updatedTask["title"])
	}
	updatedNode := patchJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(n1["id"]), map[string]any{
		"title":               "梳理接口依赖",
		"instruction":         "列出入口、服务层和回归范围",
		"acceptance_criteria": []string{"接口依赖清单完整"},
		"estimate":            1.5,
	})
	if stringValue(updatedNode["title"]) != "梳理接口依赖" {
		t.Fatalf("patched node title = %v", updatedNode["title"])
	}

	tid3 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "transition 测试", "task_key": "T"})["id"])
	n31 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes", map[string]any{"title": "准备数据", "node_key": "1"})
	n32 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes", map[string]any{"title": "写回归", "node_key": "2"})
	pausedTask := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/transition", map[string]any{"action": "pause"})
	if stringValue(pausedTask["status"]) != "paused" {
		t.Fatalf("paused task status = %v", pausedTask["status"])
	}
	pausedNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(n31["id"]))
	if stringValue(pausedNode["status"]) != "paused" {
		t.Fatalf("paused node status = %v", pausedNode["status"])
	}
	reopenedTask := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/transition", map[string]any{"action": "reopen"})
	if stringValue(reopenedTask["status"]) != "ready" {
		t.Fatalf("reopened task status = %v", reopenedTask["status"])
	}
	blockedNode := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes/"+stringValue(n31["id"])+"/block", map[string]any{"reason": "等待输入"})
	if stringValue(blockedNode["status"]) != "blocked" {
		t.Fatalf("blocked node status = %v", blockedNode["status"])
	}
	unblockedNode := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes/"+stringValue(n31["id"])+"/transition", map[string]any{"action": "unblock"})
	if stringValue(unblockedNode["status"]) != "ready" {
		t.Fatalf("unblocked node status = %v", unblockedNode["status"])
	}
	canceledNode := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes/"+stringValue(n32["id"])+"/transition", map[string]any{"action": "cancel"})
	if stringValue(canceledNode["status"]) != "canceled" {
		t.Fatalf("canceled node status = %v", canceledNode["status"])
	}
	if canceledNode["progress"].(float64) != 0 {
		t.Fatalf("canceled node progress = %v", canceledNode["progress"])
	}
	reopenedNode := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/nodes/"+stringValue(n32["id"])+"/transition", map[string]any{"action": "reopen"})
	if stringValue(reopenedNode["status"]) != "ready" {
		t.Fatalf("reopened node status = %v", reopenedNode["status"])
	}
	canceledTask := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/transition", map[string]any{"action": "cancel"})
	if stringValue(canceledTask["status"]) != "canceled" {
		t.Fatalf("canceled task rollup status = %v", canceledTask["status"])
	}
	if stringValue(canceledTask["result"]) != "canceled" {
		t.Fatalf("canceled task result = %v", canceledTask["result"])
	}
	remainingAfterCancel := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/remaining")
	if remainingAfterCancel["remaining_nodes"].(float64) != 0 {
		t.Fatalf("remaining after cancel = %v", remainingAfterCancel["remaining_nodes"])
	}
	resumeAfterCancel := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid3+"/resume")
	if resumeAfterCancel["next_node"] != nil {
		t.Fatalf("resume after cancel next_node = %#v", resumeAfterCancel["next_node"])
	}

	tid4 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "remaining 统计", "task_key": "R"})["id"])
	p41 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/nodes", map[string]any{"title": "父节点", "node_key": "1"})
	c411 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/nodes", map[string]any{"parent_node_id": p41["id"], "title": "子节点 A", "node_key": "1"})
	c412 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/nodes", map[string]any{"parent_node_id": p41["id"], "title": "子节点 B", "node_key": "2"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/nodes/"+stringValue(c411["id"])+"/block", map[string]any{"reason": "等待 A"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/nodes/"+stringValue(c412["id"])+"/block", map[string]any{"reason": "等待 B"})
	groupBlocked := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(p41["id"]))
	if stringValue(groupBlocked["status"]) != "blocked" {
		t.Fatalf("group blocked status = %v", groupBlocked["status"])
	}
	remainingBlocked := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/remaining")
	if remainingBlocked["blocked_nodes"].(float64) != 2 {
		t.Fatalf("blocked leaf count = %v", remainingBlocked["blocked_nodes"])
	}
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/transition", map[string]any{"action": "pause"})
	remainingPaused := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4+"/remaining")
	if remainingPaused["paused_nodes"].(float64) != 2 {
		t.Fatalf("paused leaf count = %v", remainingPaused["paused_nodes"])
	}

	tid4o := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "resume 顺序", "task_key": "O"})["id"])
	orderParent := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/nodes", map[string]any{"title": "第一组", "node_key": "1"})
	firstOrderedLeaf := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/nodes", map[string]any{"parent_node_id": orderParent["id"], "title": "第一叶子", "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/nodes", map[string]any{"title": "较晚创建的根节点", "node_key": "3"})
	resumeOrdered := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/resume")
	recommendedOrdered, _ := resumeOrdered["recommended_action"].(map[string]any)
	if stringValue(recommendedOrdered["node_id"]) != stringValue(firstOrderedLeaf["id"]) {
		t.Fatalf("resume ordered action should point to earliest leaf: %#v", resumeOrdered["recommended_action"])
	}
	remainingOrdered := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/remaining")
	nextReady, _ := remainingOrdered["next_ready_nodes"].([]any)
	if len(nextReady) == 0 {
		t.Fatal("remaining next_ready_nodes empty")
	}
	firstReady, _ := nextReady[0].(map[string]any)
	if stringValue(firstReady["path"]) != "O/1/1" {
		t.Fatalf("remaining next_ready first path = %v", firstReady["path"])
	}

	tid4b := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "完成后拆分回退", "task_key": "RB"})["id"])
	parentLeaf := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/nodes", map[string]any{"title": "先执行后拆分", "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/nodes/"+stringValue(parentLeaf["id"])+"/complete", map[string]any{"message": "先按叶子完成，evidence: test pass"})
	childAfterDone := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/nodes", map[string]any{"parent_node_id": parentLeaf["id"], "title": "补子步骤", "node_key": "1"})
	parentAfterSplit := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(parentLeaf["id"]))
	if stringValue(parentAfterSplit["kind"]) != "group" {
		t.Fatalf("parent kind after split = %v", parentAfterSplit["kind"])
	}
	if stringValue(parentAfterSplit["status"]) != "ready" {
		t.Fatalf("parent status after split = %v", parentAfterSplit["status"])
	}
	if stringValue(parentAfterSplit["result"]) != "" {
		t.Fatalf("parent result after split = %v", parentAfterSplit["result"])
	}
	taskAfterSplit := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b)
	if stringValue(taskAfterSplit["status"]) != "ready" {
		t.Fatalf("task status after split = %v", taskAfterSplit["status"])
	}
	resumeAfterSplit := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/resume")
	recommendedAfterSplit, _ := resumeAfterSplit["recommended_action"].(map[string]any)
	if stringValue(recommendedAfterSplit["node_id"]) == "" {
		t.Fatal("resume after split missing recommended next node")
	}
	if !strings.Contains(stringValue(childAfterDone["path"]), "RB/1/2") {
		t.Fatalf("child path after split = %v", childAfterDone["path"])
	}
	splitNodesWrap := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/nodes")
	splitNodes := workspaceAsItems(splitNodesWrap["items"])
	foundCarry := false
	for _, node := range splitNodes {
		if strings.Contains(stringValue(node["path"]), "RB/1/1") && strings.Contains(stringValue(node["title"]), "原任务") {
			foundCarry = true
			break
		}
	}
	if !foundCarry {
		t.Fatal("expected auto-generated carry child RB/1/1")
	}
	// 模拟历史数据脏状态：父节点仍被标记为 leaf，但实际上已有子节点。
	if _, err := app.db.Exec(`UPDATE nodes SET kind = 'leaf', status = 'ready', result = '' WHERE id = ?`, stringValue(parentLeaf["id"])); err != nil {
		t.Fatalf("inject stale leaf kind failed: %v", err)
	}
	remainingAfterLegacyDrift := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/remaining")
	nextReadyAfterLegacyDrift, _ := remainingAfterLegacyDrift["next_ready_nodes"].([]any)
	for _, item := range nextReadyAfterLegacyDrift {
		readyNode, _ := item.(map[string]any)
		if stringValue(readyNode["node_id"]) == stringValue(parentLeaf["id"]) {
			t.Fatalf("next_ready_nodes should exclude stale parent leaf with children: %#v", nextReadyAfterLegacyDrift)
		}
	}

	tid4d := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "空分组转回执行节点", "task_key": "RT"})["id"])
	emptyGroup := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4d+"/nodes", map[string]any{"title": "占位分组", "node_key": "1", "kind": "group"})
	retypedLeaf := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4d+"/nodes/"+stringValue(emptyGroup["id"])+"/retype", map[string]any{"message": "改回执行节点"})
	if stringValue(retypedLeaf["kind"]) != "leaf" {
		t.Fatalf("retyped kind = %v", retypedLeaf["kind"])
	}
	if stringValue(retypedLeaf["status"]) != "ready" {
		t.Fatalf("retyped status = %v", retypedLeaf["status"])
	}
	if stringValue(retypedLeaf["result"]) != "" {
		t.Fatalf("retyped result = %v", retypedLeaf["result"])
	}
	retypedProgress := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4d+"/nodes/"+stringValue(emptyGroup["id"])+"/progress", map[string]any{"delta_progress": 0.2, "message": "转型后开始执行"})
	if stringValue(retypedProgress["status"]) != "running" {
		t.Fatalf("retyped progress status = %v", retypedProgress["status"])
	}
	groupWithChild := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4d+"/nodes", map[string]any{"title": "有子节点的分组", "node_key": "2", "kind": "group"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4d+"/nodes", map[string]any{"parent_node_id": groupWithChild["id"], "title": "子节点", "node_key": "1"})
	retypeBlocked := postJSONRaw(t, server.URL+"/v1/tasks/"+tid4d+"/nodes/"+stringValue(groupWithChild["id"])+"/retype", map[string]any{}, http.StatusConflict)
	if !strings.Contains(string(retypeBlocked), "child nodes") {
		t.Fatalf("retype blocked body = %s", string(retypeBlocked))
	}

	tid4c := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "混合关闭态", "task_key": "MX"})["id"])
	mixDone := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4c+"/nodes", map[string]any{"title": "已完成节点", "node_key": "1"})
	mixOpen := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4c+"/nodes", map[string]any{"title": "待取消节点", "node_key": "2"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4c+"/nodes/"+stringValue(mixDone["id"])+"/complete", map[string]any{"message": "完成 mixed 节点，evidence: go test"})
	mixedTask := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4c+"/transition", map[string]any{"action": "cancel"})
	if stringValue(mixedTask["status"]) != "closed" {
		t.Fatalf("mixed task status = %v", mixedTask["status"])
	}
	if stringValue(mixedTask["result"]) != "mixed" {
		t.Fatalf("mixed task result = %v", mixedTask["result"])
	}
	reopenedMixedTask := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4c+"/transition", map[string]any{"action": "reopen"})
	if stringValue(reopenedMixedTask["status"]) != "ready" {
		t.Fatalf("reopened mixed task status = %v", reopenedMixedTask["status"])
	}
	reopenedMixOpen := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(mixOpen["id"]))
	if stringValue(reopenedMixOpen["status"]) != "ready" || stringValue(reopenedMixOpen["result"]) != "" {
		t.Fatalf("reopened canceled leaf = %#v", reopenedMixOpen)
	}
	mixDoneNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(mixDone["id"]))
	if stringValue(mixDoneNode["status"]) != "done" || stringValue(mixDoneNode["result"]) != "done" {
		t.Fatalf("done leaf should remain done after task reopen = %#v", mixDoneNode)
	}

	tid5 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "release 同步"})["id"])
	n51 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes", map[string]any{"title": "零进度释放"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes/"+stringValue(n51["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-release"}, "lease_seconds": 60})
	releasedReady := postNoBodyJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes/"+stringValue(n51["id"])+"/release")
	if stringValue(releasedReady["status"]) != "ready" {
		t.Fatalf("release zero progress status = %v", releasedReady["status"])
	}
	taskAfterRelease := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5)
	if stringValue(taskAfterRelease["status"]) != "ready" {
		t.Fatalf("task status after release = %v", taskAfterRelease["status"])
	}
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes/"+stringValue(n51["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-release"}, "lease_seconds": 60})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes/"+stringValue(n51["id"])+"/progress", map[string]any{"delta_progress": 0.25, "message": "开始执行"})
	releasedRunning := postNoBodyJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid5+"/nodes/"+stringValue(n51["id"])+"/release")
	if stringValue(releasedRunning["status"]) != "running" {
		t.Fatalf("release progressed status = %v", releasedRunning["status"])
	}

	tid6 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "complete 幂等"})["id"])
	n61 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid6+"/nodes", map[string]any{"title": "重复完成"})
	firstComplete := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid6+"/nodes/"+stringValue(n61["id"])+"/complete", map[string]any{"message": "完成并附带测试 evidence", "idempotency_key": "complete-once"})
	time.Sleep(10 * time.Millisecond)
	secondComplete := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid6+"/nodes/"+stringValue(n61["id"])+"/complete", map[string]any{"message": "重复提交", "idempotency_key": "complete-once"})
	if firstComplete["version"].(float64) != secondComplete["version"].(float64) {
		t.Fatalf("complete idempotency version changed: %v -> %v", firstComplete["version"], secondComplete["version"])
	}

	tid7 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "lease 过期"})["id"])
	n71 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid7+"/nodes", map[string]any{"title": "过期归位"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid7+"/nodes/"+stringValue(n71["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-expired"}, "lease_seconds": 0})
	time.Sleep(5 * time.Millisecond)
	cleared, err := app.sweepExpiredLeases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cleared < 1 {
		t.Fatalf("sweep cleared = %d", cleared)
	}
	expiredNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(n71["id"]))
	if stringValue(expiredNode["status"]) != "ready" {
		t.Fatalf("expired lease status = %v", expiredNode["status"])
	}

	tid8 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "claim 清理"})["id"])
	n81 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes", map[string]any{"title": "阻塞时释放占用"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes/"+stringValue(n81["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-sync"}, "lease_seconds": 60})
	blockedReleased := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes/"+stringValue(n81["id"])+"/block", map[string]any{"reason": "等待上游同步"})
	if blockedReleased["claimed_by_id"] != nil {
		t.Fatalf("blocked node should clear claim, got = %v", blockedReleased["claimed_by_id"])
	}
	claimBlockedBody := postJSONRaw(t, server.URL+"/v1/tasks/"+tid8+"/nodes/"+stringValue(n81["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-sync"}}, http.StatusConflict)
	if !strings.Contains(string(claimBlockedBody), "blocked") {
		t.Fatalf("claim blocked body = %s", string(claimBlockedBody))
	}
	n82 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes", map[string]any{"title": "完成时释放占用"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes/"+stringValue(n82["id"])+"/claim", map[string]any{"actor": map[string]any{"tool": "codex", "agent_id": "worker-sync"}, "lease_seconds": 60})
	completedReleased := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid8+"/nodes/"+stringValue(n82["id"])+"/complete", map[string]any{"message": "完成时同步清理 lease，Evidence: smoke test pass"})
	if completedReleased["claimed_by_id"] != nil {
		t.Fatalf("completed node should clear claim, got = %v", completedReleased["claimed_by_id"])
	}
	if stringValue(completedReleased["status"]) != "done" {
		t.Fatalf("completed node status = %v", completedReleased["status"])
	}

	tid9 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "work 页面过滤"})["id"])
	blockedNodeForWork := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid9+"/nodes", map[string]any{"title": "Blocked Hidden"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid9+"/nodes/"+stringValue(blockedNodeForWork["id"])+"/block", map[string]any{"reason": "隐藏在可领取列表外"})

	delTid := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "待删除"})["id"])
	deleteReq(t, server.URL+"/v1/tasks/"+delTid)
	deleteReq(t, server.URL+"/v1/tasks/"+delTid+"/hard")

	resp2, err := http.Get(server.URL + "/")
	if err != nil || resp2.StatusCode != 200 {
		t.Fatalf("ui / status = %v %v", err, resp2.StatusCode)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body2), "Task Tree") || !strings.Contains(string(body2), "id=\"app\"") {
		t.Fatalf("ui / body = %s", string(body2))
	}

	resp3, err := http.Get(server.URL + "/tasks/" + tid)
	if err != nil || resp3.StatusCode != 200 {
		t.Fatalf("ui /tasks/{id} status = %v %v", err, resp3.StatusCode)
	}
	defer resp3.Body.Close()
	body3, _ := io.ReadAll(resp3.Body)
	if !strings.Contains(string(body3), "Task Tree") || !strings.Contains(string(body3), "id=\"app\"") {
		t.Fatalf("ui detail body = %s", string(body3))
	}
	respDefaultSelect, err := http.Get(server.URL + "/tasks/" + tid4o)
	if err != nil || respDefaultSelect.StatusCode != 200 {
		t.Fatalf("ui default selected status = %v %v", err, respDefaultSelect.StatusCode)
	}
	defer respDefaultSelect.Body.Close()
	bodyDefaultSelect, _ := io.ReadAll(respDefaultSelect.Body)
	if !strings.Contains(string(bodyDefaultSelect), "Task Tree") || !strings.Contains(string(bodyDefaultSelect), "id=\"app\"") {
		t.Fatalf("default selected node body = %s", string(bodyDefaultSelect))
	}

	resp3b, err := http.Get(server.URL + "/tasks/" + tid + "?node=" + stringValue(n22["id"]) + "&tab=children")
	if err != nil || resp3b.StatusCode != 200 {
		t.Fatalf("ui detail children status = %v %v", err, resp3b.StatusCode)
	}
	defer resp3b.Body.Close()
	body3b, _ := io.ReadAll(resp3b.Body)
	if !strings.Contains(string(body3b), "Task Tree") || !strings.Contains(string(body3b), "id=\"app\"") {
		t.Fatalf("ui detail children body = %s", string(body3b))
	}

	resp4, err := http.Get(server.URL + "/new-task")
	if err != nil || resp4.StatusCode != 200 {
		t.Fatalf("ui /new-task status = %v %v", err, resp4.StatusCode)
	}
	defer resp4.Body.Close()
	body4, _ := io.ReadAll(resp4.Body)
	if !strings.Contains(string(body4), "Task Tree") || !strings.Contains(string(body4), "id=\"app\"") {
		t.Fatalf("ui new-task body = %s", string(body4))
	}

	formTask := postForm(t, server.URL+"/ui/tasks/create", url.Values{
		"title":    {"UI 创建任务"},
		"goal":     {"通过页面创建任务"},
		"task_key": {"UI"},
	})
	if formTask.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui create task status = %d", formTask.StatusCode)
	}
	if loc := formTask.Header.Get("Location"); !strings.Contains(loc, "/tasks/") {
		t.Fatalf("ui create task location = %s", loc)
	}

	formNode := postForm(t, server.URL+"/ui/tasks/"+tid+"/nodes/create", url.Values{
		"title":       {"UI 创建节点"},
		"instruction": {"在页面里补一个节点"},
		"acceptance":  {"页面创建成功"},
		"estimate":    {"1"},
	})
	if formNode.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui create node status = %d", formNode.StatusCode)
	}

	searchNode := getJSON[map[string]any](t, server.URL+"/v1/search?q=UI%20创建节点")
	nodesFound, _ := searchNode["nodes"].([]any)
	if len(nodesFound) == 0 {
		t.Fatalf("ui created node not found: %#v", searchNode)
	}

	resp5, err := http.Get(server.URL + "/nodes/" + stringValue(nc["id"]) + "/actions")
	if err != nil || resp5.StatusCode != 200 {
		t.Fatalf("ui node actions status = %v %v", err, resp5.StatusCode)
	}
	defer resp5.Body.Close()
	body5, _ := io.ReadAll(resp5.Body)
	if !strings.Contains(string(body5), "Task Tree") || !strings.Contains(string(body5), "id=\"app\"") {
		t.Fatalf("ui node actions body = %s", string(body5))
	}

	respWork, err := http.Get(server.URL + "/work")
	if err != nil || respWork.StatusCode != 200 {
		t.Fatalf("ui /work status = %v %v", err, respWork.StatusCode)
	}
	defer respWork.Body.Close()
	workBody, _ := io.ReadAll(respWork.Body)
	if !strings.Contains(string(workBody), "Task Tree") || !strings.Contains(string(workBody), "id=\"app\"") {
		t.Fatalf("ui /work body: %s", string(workBody))
	}

	uiSaveTask := postForm(t, server.URL+"/ui/tasks/"+tid+"/save", url.Values{
		"title":    {"UI 保存后的任务"},
		"goal":     {"通过工作台保存任务"},
		"task_key": {"UIT"},
		"node_id":  {stringValue(n1["id"])},
		"tab":      {"edit"},
	})
	if uiSaveTask.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui save task status = %d", uiSaveTask.StatusCode)
	}
	uiSaveNode := postForm(t, server.URL+"/ui/nodes/"+stringValue(n1["id"])+"/save", url.Values{
		"title":       {"UI 保存后的节点"},
		"instruction": {"通过工作台直接编辑节点"},
		"acceptance":  {"保存成功"},
		"estimate":    {"2"},
		"tab":         {"edit"},
	})
	if uiSaveNode.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui save node status = %d", uiSaveNode.StatusCode)
	}
	savedNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(n1["id"]))
	if stringValue(savedNode["title"]) != "UI 保存后的节点" {
		t.Fatalf("saved node title = %v", savedNode["title"])
	}

	uiPauseTask := postForm(t, server.URL+"/ui/tasks/"+tid3+"/transition", url.Values{
		"action": {"pause"},
		"tab":    {"edit"},
	})
	if uiPauseTask.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui task pause status = %d", uiPauseTask.StatusCode)
	}
	uiReopenNode := postForm(t, server.URL+"/ui/nodes/"+stringValue(n32["id"])+"/transition", url.Values{
		"action": {"reopen"},
		"tab":    {"edit"},
	})
	if uiReopenNode.StatusCode != http.StatusSeeOther {
		t.Fatalf("ui node reopen status = %d", uiReopenNode.StatusCode)
	}

	tid10 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "父节点事件聚合"})["id"])
	parent10 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid10+"/nodes", map[string]any{"title": "父节点", "node_key": "1"})
	child10 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid10+"/nodes", map[string]any{"parent_node_id": parent10["id"], "title": "子节点", "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid10+"/nodes/"+stringValue(child10["id"])+"/progress", map[string]any{"delta_progress": 0.4, "message": "推进子节点"})
	eventsParentOnly := getJSON[map[string]any](t, server.URL+"/v1/events?task_id="+tid10+"&node_id="+stringValue(parent10["id"]))
	parentOnlyItems, _ := eventsParentOnly["items"].([]any)
	if len(parentOnlyItems) == 0 {
		t.Fatalf("parent events should include at least self events")
	}
	eventsWithDesc := getJSON[map[string]any](t, server.URL+"/v1/events?task_id="+tid10+"&node_id="+stringValue(parent10["id"])+"&include_descendants=true")
	withDescItems, _ := eventsWithDesc["items"].([]any)
	hasChildEvent := false
	for _, raw := range withDescItems {
		item, _ := raw.(map[string]any)
		if stringValue(item["node_id"]) == stringValue(child10["id"]) {
			hasChildEvent = true
			break
		}
	}
	if !hasChildEvent {
		t.Fatalf("expected descendant events to include child node events: %#v", eventsWithDesc)
	}

	tid11 := stringValue(postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "默认项目归属"})["id"])
	task11 := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid11)
	projectID := stringValue(task11["project_id"])
	if projectID == "" {
		t.Fatalf("task should be assigned to default project: %#v", task11)
	}
	project11 := getJSON[map[string]any](t, server.URL+"/v1/projects/"+projectID)
	if stringValue(project11["name"]) == "" {
		t.Fatalf("project should be readable: %#v", project11)
	}
	overview11 := getJSON[map[string]any](t, server.URL+"/v1/projects/"+projectID+"/overview")
	tasksInOverview, _ := overview11["tasks"].([]any)
	if len(tasksInOverview) == 0 {
		t.Fatalf("project overview should include tasks: %#v", overview11)
	}

	project12 := postJSON[map[string]any](t, server.URL+"/v1/projects", map[string]any{
		"name":        "待删除项目",
		"project_key": "DELPROJ",
	})
	project12ID := stringValue(project12["id"])
	task12 := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{
		"title":      "删除项目后的恢复任务",
		"project_id": project12ID,
	})
	task12ID := stringValue(task12["id"])
	deleteJSON(t, server.URL+"/v1/projects/"+project12ID)
	deletedTasks12 := getJSON[[]map[string]any](t, server.URL+"/v1/tasks?include_deleted=true")
	foundDeleted12 := false
	for _, item := range deletedTasks12 {
		if stringValue(item["id"]) == task12ID && item["deleted_at"] != nil {
			foundDeleted12 = true
			break
		}
	}
	if !foundDeleted12 {
		t.Fatalf("task should enter trash after project deletion: %#v", deletedTasks12)
	}
	restored12 := postNoBodyJSON[map[string]any](t, server.URL+"/v1/tasks/"+task12ID+"/restore")
	if stringValue(restored12["deleted_at"]) != "" {
		t.Fatalf("restored task should not stay deleted: %#v", restored12)
	}
	if stringValue(restored12["project_id"]) == "" || stringValue(restored12["project_id"]) == project12ID {
		t.Fatalf("restored task should move to active default project: %#v", restored12)
	}
	defaultProject12 := getJSON[map[string]any](t, server.URL+"/v1/projects/"+stringValue(restored12["project_id"]))
	if stringValue(defaultProject12["name"]) == "" {
		t.Fatalf("restored task target project should be readable: %#v", defaultProject12)
	}
	overview12 := getJSON[map[string]any](t, server.URL+"/v1/projects/"+stringValue(restored12["project_id"])+"/overview")
	tasksInOverview12, _ := overview12["tasks"].([]any)
	foundRestoredInOverview := false
	for _, raw := range tasksInOverview12 {
		item, _ := raw.(map[string]any)
		if stringValue(item["id"]) == task12ID {
			foundRestoredInOverview = true
			break
		}
	}
	if !foundRestoredInOverview {
		t.Fatalf("restored task should reappear in active project overview: %#v", overview12)
	}

	nodesSummary := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes?view_mode=summary&filter_mode=focus&sort_by=path&limit=20")
	summaryItems, _ := nodesSummary["items"].([]any)
	if len(summaryItems) == 0 {
		t.Fatalf("summary nodes should not be empty: %#v", nodesSummary)
	}
	firstSummary, _ := summaryItems[0].(map[string]any)
	if _, ok := firstSummary["next_action"]; !ok {
		t.Fatalf("summary node should include next_action: %#v", firstSummary)
	}
	if _, ok := firstSummary["has_children"]; !ok {
		t.Fatalf("summary node should include has_children: %#v", firstSummary)
	}

	eventsSummary := getJSON[map[string]any](t, server.URL+"/v1/events?task_id="+tid+"&view_mode=summary&type=node_created&limit=10")
	eventItems, _ := eventsSummary["items"].([]any)
	if len(eventItems) == 0 {
		t.Fatalf("summary events should not be empty: %#v", eventsSummary)
	}
	firstEvent, _ := eventItems[0].(map[string]any)
	if _, ok := firstEvent["payload"]; ok {
		t.Fatalf("summary event should not contain payload: %#v", firstEvent)
	}
}

func TestCreateTaskWithInitialNodeTree(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "seed-tree.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{
		"title":    "一次性建树",
		"task_key": "TREE",
		"goal":     "创建任务时直接带复杂节点树",
		"nodes": []map[string]any{
			{
				"title":    "设计接口",
				"node_key": "1",
			},
			{
				"title":    "实现后端",
				"node_key": "2",
				"children": []map[string]any{
					{
						"title":    "设计数据结构",
						"node_key": "1",
					},
					{
						"title":    "实现递归建树",
						"node_key": "2",
						"children": []map[string]any{
							{
								"title":    "补 MCP 入参",
								"node_key": "1",
							},
						},
					},
				},
			},
		},
	})
	tid := stringValue(task["id"])
	nodesWrap := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes")
	nodes := workspaceAsItems(nodesWrap["items"])
	if len(nodes) != 5 {
		t.Fatalf("seed nodes len = %d", len(nodes))
	}

	paths := make(map[string]map[string]any, len(nodes))
	for _, node := range nodes {
		paths[stringValue(node["path"])] = node
	}
	if _, ok := paths["TREE/1"]; !ok {
		t.Fatalf("missing root node TREE/1: %#v", paths)
	}
	rootBackend, ok := paths["TREE/2"]
	if !ok {
		t.Fatalf("missing root node TREE/2: %#v", paths)
	}
	if stringValue(rootBackend["kind"]) != "group" {
		t.Fatalf("TREE/2 kind = %v", rootBackend["kind"])
	}
	if _, ok := paths["TREE/2/1"]; !ok {
		t.Fatalf("missing child node TREE/2/1: %#v", paths)
	}
	if _, ok := paths["TREE/2/2/1"]; !ok {
		t.Fatalf("missing nested child TREE/2/2/1: %#v", paths)
	}

	resume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid+"/resume")
	nextWrap, _ := resume["next_node"].(map[string]any)
	nextNode, _ := nextWrap["node"].(map[string]any)
	if stringValue(nextNode["path"]) != "TREE/1" {
		t.Fatalf("next node path = %v", nextNode["path"])
	}
}

func TestStagesFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "stages.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "阶段任务", "task_key": "STG"})
	taskID := stringValue(task["id"])

	stage1 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "阶段一", "node_key": "1"})
	if stringValue(stage1["role"]) != "stage" {
		t.Fatalf("stage role = %v", stage1["role"])
	}
	taskAfterStage1 := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID)
	if stringValue(taskAfterStage1["current_stage_node_id"]) != stringValue(stage1["id"]) {
		t.Fatalf("current stage = %v", taskAfterStage1["current_stage_node_id"])
	}

	step1 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "阶段内步骤", "node_key": "1"})
	if stringValue(step1["parent_node_id"]) != stringValue(stage1["id"]) {
		t.Fatalf("step parent = %v", step1["parent_node_id"])
	}
	if stringValue(step1["stage_node_id"]) != stringValue(stage1["id"]) {
		t.Fatalf("step stage_node_id = %v", step1["stage_node_id"])
	}

	stage2 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "阶段二", "node_key": "2"})
	stages := getJSON[[]map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages")
	if len(stages) != 2 {
		t.Fatalf("stages len = %d", len(stages))
	}

	activated := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages/"+stringValue(stage2["id"])+"/activate", map[string]any{})
	if stringValue(activated["current_stage_node_id"]) != stringValue(stage2["id"]) {
		t.Fatalf("activated stage = %v", activated["current_stage_node_id"])
	}
	step2 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "第二阶段步骤", "node_key": "1"})
	if stringValue(step2["parent_node_id"]) != stringValue(stage2["id"]) {
		t.Fatalf("step2 parent = %v", step2["parent_node_id"])
	}
	if stringValue(step2["stage_node_id"]) != stringValue(stage2["id"]) {
		t.Fatalf("step2 stage_node_id = %v", step2["stage_node_id"])
	}

	// 即使当前激活阶段是 stage2，显式 stage_node_id 仍应决定默认父节点归属。
	batch := postJSONArray[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/batch", []map[string]any{
		{
			"title":         "显式归属阶段一",
			"node_key":      "2",
			"stage_node_id": stringValue(stage1["id"]),
		},
	})
	created, _ := batch["created"].([]any)
	if len(created) != 1 {
		t.Fatalf("batch created len = %d", len(created))
	}
	createdNode, _ := created[0].(map[string]any)
	if stringValue(createdNode["parent_node_id"]) != stringValue(stage1["id"]) {
		t.Fatalf("batch node parent should follow stage_node_id, got %v", createdNode["parent_node_id"])
	}
	if stringValue(createdNode["stage_node_id"]) != stringValue(stage1["id"]) {
		t.Fatalf("batch node stage_node_id = %v", createdNode["stage_node_id"])
	}
}

func TestNodeRunsFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "runs.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()

	task, err := app.createTask(context.Background(), taskCreate{Title: "执行层测试", TaskKey: strPtr("RUN")})
	if err != nil {
		t.Fatal(err)
	}
	node, err := app.createNode(context.Background(), stringValue(task["id"]), nodeCreate{Title: "执行节点", NodeKey: strPtr("1")})
	if err != nil {
		t.Fatal(err)
	}
	run, err := app.startRun(context.Background(), stringValue(node["id"]), runStart{
		TriggerKind:  strPtr("manual"),
		InputSummary: strPtr("开始执行 run flow"),
		Actor:        &actor{Tool: strPtr("codex"), AgentID: strPtr("runner-1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(run["status"]) != "running" {
		t.Fatalf("run status = %v", run["status"])
	}
	if stringValue(run["trigger_kind"]) != "manual" {
		t.Fatalf("trigger kind = %v", run["trigger_kind"])
	}

	activeNode, err := app.findNode(context.Background(), stringValue(node["id"]), false)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(activeNode["active_run_id"]) != stringValue(run["id"]) {
		t.Fatalf("active run id = %v", activeNode["active_run_id"])
	}
	if stringValue(activeNode["status"]) != "running" {
		t.Fatalf("node status after start = %v", activeNode["status"])
	}
	if asInt(activeNode["run_count"]) != 1 {
		t.Fatalf("run count after start = %v", activeNode["run_count"])
	}

	if _, err := app.startRun(context.Background(), stringValue(node["id"]), runStart{}); err == nil {
		t.Fatal("expected conflict when starting second active run")
	}

	logItem, err := app.addRunLog(context.Background(), stringValue(run["id"]), runLogCreate{
		Kind:    "stdout",
		Content: strPtr("first line"),
		Payload: map[string]any{"chunk": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if asInt(logItem["seq"]) != 1 {
		t.Fatalf("log seq = %v", logItem["seq"])
	}

	runWithLogs, err := app.getRun(context.Background(), stringValue(run["id"]))
	if err != nil {
		t.Fatal(err)
	}
	logs, _ := runWithLogs["logs"].([]jsonMap)
	if len(logs) != 1 {
		t.Fatalf("run logs len = %d", len(logs))
	}

	finished, err := app.finishRun(context.Background(), stringValue(run["id"]), runFinish{
		Result:        strPtr("done"),
		Status:        strPtr("finished"),
		OutputPreview: strPtr("执行完成"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(finished["result"]) != "done" {
		t.Fatalf("finished result = %v", finished["result"])
	}
	if stringValue(finished["status"]) != "finished" {
		t.Fatalf("finished status = %v", finished["status"])
	}

	doneNode, err := app.findNode(context.Background(), stringValue(node["id"]), false)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(doneNode["active_run_id"]) != "" {
		t.Fatalf("active run not cleared = %v", doneNode["active_run_id"])
	}
	if stringValue(doneNode["status"]) != "done" {
		t.Fatalf("node status after finish = %v", doneNode["status"])
	}
	if stringValue(doneNode["result"]) != "done" {
		t.Fatalf("node result after finish = %v", doneNode["result"])
	}

	nodeRuns, err := app.listNodeRuns(context.Background(), stringValue(node["id"]), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodeRuns) != 1 {
		t.Fatalf("node runs len = %d", len(nodeRuns))
	}
}

func TestRunRoutesAndSyntheticCompatibility(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "run-routes.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "执行接口测试", "task_key": "RUNAPI"})
	node := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+stringValue(task["id"])+"/nodes", map[string]any{"title": "执行节点", "node_key": "1"})

	run := postJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/runs", map[string]any{
		"trigger_kind":  "manual",
		"input_summary": "从 HTTP 启动 run",
	})
	if stringValue(run["status"]) != "running" {
		t.Fatalf("http start run status = %v", run["status"])
	}

	logItem := postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/logs", map[string]any{
		"kind":    "stdout",
		"content": "line 1",
		"payload": map[string]any{"index": 1},
	})
	if logItem["seq"].(float64) != 1 {
		t.Fatalf("http run log seq = %v", logItem["seq"])
	}

	runFetched := getJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"?include_logs=true")
	logs, _ := runFetched["logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("http fetched run logs len = %d", len(logs))
	}

	finished := postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/finish", map[string]any{
		"result":         "done",
		"status":         "finished",
		"output_preview": "HTTP 执行完成",
	})
	if stringValue(finished["result"]) != "done" {
		t.Fatalf("http finished result = %v", finished["result"])
	}

	runsWrap := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/runs")
	runs := workspaceAsItems(runsWrap["items"])
	if len(runs) != 1 {
		t.Fatalf("http node runs len = %d", len(runs))
	}

	task2 := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "synthetic 兼容", "task_key": "SYN"})
	node2 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+stringValue(task2["id"])+"/nodes", map[string]any{"title": "旧接口节点", "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+stringValue(task2["id"])+"/nodes/"+stringValue(node2["id"])+"/progress", map[string]any{
		"delta_progress": 0.5,
		"message":        "旧 progress 接口推进 50%",
	})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+stringValue(task2["id"])+"/nodes/"+stringValue(node2["id"])+"/complete", map[string]any{
		"message": "旧 complete 接口完成节点",
	})
	syntheticRunsWrap := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node2["id"])+"/runs")
	syntheticRuns := workspaceAsItems(syntheticRunsWrap["items"])
	if len(syntheticRuns) == 0 {
		t.Fatal("expected synthetic run for legacy node APIs")
	}
	syntheticRun := getJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(syntheticRuns[0]["id"])+"?include_logs=true")
	if stringValue(syntheticRun["result"]) != "done" {
		t.Fatalf("synthetic run result = %v", syntheticRun["result"])
	}
	syntheticLogs, _ := syntheticRun["logs"].([]any)
	if len(syntheticLogs) == 0 {
		t.Fatal("expected synthetic run logs")
	}
}

func TestMemoryAndReadModelFlow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "memory.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "记忆任务", "task_key": "MEM"})
	taskID := stringValue(task["id"])
	stage := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "实施阶段", "node_key": "1"})
	node := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "实现记忆层", "node_key": "1"})

	run := postJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/runs", map[string]any{
		"input_summary": "开始实现记忆层",
	})
	postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/logs", map[string]any{
		"kind":    "stdout",
		"content": "memory merge done",
		"payload": map[string]any{"step": "merge"},
	})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/artifacts", map[string]any{
		"node_id": node["id"],
		"run_id":  run["id"],
		"kind":    "note",
		"title":   "执行产物",
		"uri":     "memory://artifact/1",
	})
	postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/finish", map[string]any{
		"result":         "done",
		"status":         "finished",
		"output_preview": "执行完成，产出 memory 摘要",
		"structured_result": map[string]any{
			"summary_text": "记忆层已完成初版实现",
			"conclusions":  []string{"memory merge 已接通"},
			"risks":        []string{"还需要补 UI 展示"},
			"next_actions": []string{"继续接读取层"},
			"evidence":     []string{"go test ./..."},
		},
	})

	nodeMemory := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/memory")
	if !strings.Contains(stringValue(nodeMemory["summary_text"]), "记忆层已完成初版实现") {
		t.Fatalf("node memory summary = %#v", nodeMemory)
	}
	stageMemory := getJSON[map[string]any](t, server.URL+"/v1/stages/"+stringValue(stage["id"])+"/memory")
	if stageMemory["summary_text"] == nil {
		t.Fatalf("stage memory missing summary: %#v", stageMemory)
	}
	taskMemory := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/memory")
	if taskMemory["summary_text"] == nil {
		t.Fatalf("task memory missing summary: %#v", taskMemory)
	}

	patchedNodeMemory := patchJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/memory", map[string]any{"manual_note_text": "节点人工备注"})
	if stringValue(patchedNodeMemory["manual_note_text"]) != "节点人工备注" {
		t.Fatalf("patched node memory = %#v", patchedNodeMemory)
	}
	patchedStageMemory := patchJSON[map[string]any](t, server.URL+"/v1/stages/"+stringValue(stage["id"])+"/memory", map[string]any{"manual_note_text": "阶段人工备注"})
	if stringValue(patchedStageMemory["manual_note_text"]) != "阶段人工备注" {
		t.Fatalf("patched stage memory = %#v", patchedStageMemory)
	}
	patchedTaskMemory := patchJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/memory", map[string]any{"manual_note_text": "任务人工备注"})
	if stringValue(patchedTaskMemory["manual_note_text"]) != "任务人工备注" {
		t.Fatalf("patched task memory = %#v", patchedTaskMemory)
	}
	nodeMemoryConflict := patchJSONRaw(t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/memory", map[string]any{
		"manual_note_text": "冲突备注",
		"expected_version": 1,
	}, http.StatusConflict)
	if !strings.Contains(string(nodeMemoryConflict), "version mismatch") {
		t.Fatalf("expected node memory conflict body = %s", string(nodeMemoryConflict))
	}
	taskSnapshot := postNoBodyJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/memory/snapshot")
	if stringValue(taskSnapshot["scope_kind"]) != "task" {
		t.Fatalf("task snapshot = %#v", taskSnapshot)
	}

	contextNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/context")
	if contextNode["memory"] == nil || contextNode["stage_summary"] == nil {
		t.Fatalf("node context missing memory/stage: %#v", contextNode)
	}
	recentRuns, _ := contextNode["recent_runs"].([]any)
	if len(recentRuns) == 0 {
		t.Fatalf("node context missing recent runs: %#v", contextNode)
	}
	contextArtifacts, _ := contextNode["artifacts"].([]any)
	if len(contextArtifacts) == 0 {
		t.Fatalf("node context missing artifacts: %#v", contextNode)
	}

	resume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume?debug=1&include=runs")
	if resume["task_memory"] != nil {
		t.Fatalf("resume should not include task_memory by default: %#v", resume)
	}
	if resume["task_memory_summary"] == nil {
		t.Fatalf("resume missing task_memory_summary: %#v", resume)
	}
	if resume["current_stage_memory"] != nil {
		t.Fatalf("resume should not include current_stage_memory by default: %#v", resume)
	}
	debug, _ := resume["debug"].(map[string]any)
	if debug == nil || debug["focus_nodes_count"] == nil {
		t.Fatalf("resume debug missing: %#v", resume)
	}
	recentRunsWrap, _ := resume["recent_runs"].([]any)
	if len(recentRunsWrap) == 0 {
		t.Fatalf("resume missing recent runs: %#v", resume)
	}

	overview := getJSON[map[string]any](t, server.URL+"/v1/projects/"+stringValue(task["project_id"])+"/overview")
	tasksInOverview, _ := overview["tasks"].([]any)
	if len(tasksInOverview) == 0 {
		t.Fatalf("overview missing tasks: %#v", overview)
	}
	firstTask, _ := tasksInOverview[0].(map[string]any)
	if firstTask["memory"] == nil {
		t.Fatalf("overview task missing memory: %#v", firstTask)
	}
	mem, _ := firstTask["memory"].(map[string]any)
	if mem == nil || mem["summary"] == nil {
		t.Fatalf("overview task memory should expose summary alias: %#v", firstTask)
	}
	if firstTask["current_stage"] == nil {
		t.Fatalf("overview task missing current_stage: %#v", firstTask)
	}
	if firstTask["goal"] != nil {
		t.Fatalf("overview task should stay summary-only: %#v", firstTask)
	}
	if firstTask["remaining"] == nil {
		t.Fatalf("overview task should expose remaining summary: %#v", firstTask)
	}
}

func TestMigrationDuplicateVersionAndExpectedVersionConflict(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "conflict.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "版本冲突测试", "task_key": "VER"})
	taskID := stringValue(task["id"])
	conflictBody := patchJSONRaw(t, server.URL+"/v1/tasks/"+taskID, map[string]any{
		"title":            "错误版本更新",
		"expected_version": 999,
	}, http.StatusConflict)
	if !strings.Contains(string(conflictBody), "version mismatch") {
		t.Fatalf("expected version conflict body = %s", string(conflictBody))
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	dupRoot := filepath.Join(tmp, "dup-migrate")
	if err := os.MkdirAll(filepath.Join(dupRoot, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dupRoot, "migrations", "001_first.sql"), []byte("CREATE TABLE first_table(id INTEGER);"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dupRoot, "migrations", "001_second.sql"), []byte("CREATE TABLE second_table(id INTEGER);"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dupRoot); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(tmp, "dup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dupApp := &App{db: db}
	err = dupApp.migrate()
	if err == nil || !strings.Contains(err.Error(), "duplicate migration version") {
		t.Fatalf("expected duplicate migration version error, got %v", err)
	}
}

func TestDryRunAndBatchAtomicDependsOnKeys(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "dryrun-atomic.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	dry := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{
		"title":    "DryRun 任务",
		"task_key": "DRY",
		"dry_run":  true,
		"stages": []map[string]any{
			{"title": "S1", "node_key": "S1", "activate": true},
		},
		"nodes": []map[string]any{
			{"title": "N1", "node_key": "N1"},
			{"title": "N2", "node_key": "N2", "depends_on_keys": []string{"N1"}},
		},
	})
	if dry["dry_run"] != true || dry["validated"] != true {
		t.Fatalf("dry-run validate failed: %#v", dry)
	}
	tasksAfterDry := getJSON[[]map[string]any](t, server.URL+"/v1/tasks")
	if len(tasksAfterDry) != 0 {
		t.Fatalf("dry-run should not persist task, got %d", len(tasksAfterDry))
	}

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "原子批量", "task_key": "ATOMIC"})
	taskID := stringValue(task["id"])
	raw := postJSONArrayRaw(t, server.URL+"/v1/tasks/"+taskID+"/nodes/batch", []map[string]any{
		{"title": "A", "node_key": "A"},
		{"title": "B", "node_key": "B", "depends_on_keys": []string{"NOT_FOUND"}},
	}, http.StatusBadRequest)
	if !strings.Contains(string(raw), "depends_on_keys") {
		t.Fatalf("unexpected batch failure body: %s", string(raw))
	}
	nodesWrap := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes")
	nodes := workspaceAsItems(nodesWrap["items"])
	if len(nodes) != 0 {
		t.Fatalf("batch should rollback on failure, got %d nodes", len(nodes))
	}
}

func TestCheckpointCompletionAndLatestResultPayload(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "checkpoint.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "Checkpoint 任务", "task_key": "CP"})
	taskID := stringValue(task["id"])
	node := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{
		"title":    "构建检查点",
		"node_key": "CK1",
		"role":     "checkpoint",
		"metadata": map[string]any{
			"checkpoint_spec": map[string]any{
				"required_commands": []string{"go build ./..."},
			},
		},
	})
	nodeID := stringValue(node["id"])

	noPayload := postJSONRaw(t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+nodeID+"/complete", map[string]any{"message": "缺少结果"}, http.StatusConflict)
	if !strings.Contains(string(noPayload), "commands_verified") {
		t.Fatalf("checkpoint should require commands_verified, body=%s", string(noPayload))
	}
	wrongPayload := postJSONRaw(t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+nodeID+"/complete", map[string]any{
		"message": "命令不匹配",
		"result_payload": map[string]any{
			"commands_verified": []string{"go test ./..."},
		},
	}, http.StatusConflict)
	if !strings.Contains(string(wrongPayload), "checkpoint 未通过") {
		t.Fatalf("checkpoint mismatch body=%s", string(wrongPayload))
	}

	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+nodeID+"/complete", map[string]any{
		"message": "检查点通过",
		"result_payload": map[string]any{
			"files_modified":    []string{"backend/main.go"},
			"commands_verified": []string{"go build ./..."},
			"notes":             "构建已通过",
		},
	})

	ctxNode := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+nodeID+"/context")
	latest, _ := ctxNode["latest_result_payload"].(map[string]any)
	if latest == nil {
		t.Fatalf("missing latest_result_payload in node context: %#v", ctxNode)
	}
	verified, _ := latest["commands_verified"].([]any)
	if len(verified) != 1 || stringValue(verified[0]) != "go build ./..." {
		t.Fatalf("unexpected latest commands_verified: %#v", latest["commands_verified"])
	}
	runsWrap := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+nodeID+"/runs")
	runs := workspaceAsItems(runsWrap["items"])
	if len(runs) == 0 {
		t.Fatal("expected node runs")
	}
	if runs[0]["latest_result_payload"] == nil {
		t.Fatalf("run missing latest_result_payload: %#v", runs[0])
	}
}

func TestParallelGroupAlternativesAndStageSuggestions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "parallel-stage.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "并行与阶段建议", "task_key": "PAR"})
	taskID := stringValue(task["id"])
	stage1 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "数据层", "node_key": "S1", "activate": true})
	stage2 := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "服务层", "node_key": "S2"})

	group := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{
		"title":         "并行组",
		"node_key":      "PG",
		"kind":          "group",
		"role":          "parallel_group",
		"stage_node_id": stage1["id"],
	})
	childA := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{
		"title":          "并行任务A",
		"parent_node_id": group["id"],
		"node_key":       "1",
	})
	childB := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{
		"title":          "并行任务B",
		"parent_node_id": group["id"],
		"node_key":       "2",
	})

	next := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/next-node")
	rec, _ := next["recommended_action"].(map[string]any)
	alts, _ := rec["alternatives"].([]any)
	if len(alts) == 0 {
		t.Fatalf("parallel group should return alternatives: %#v", next)
	}

	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(childA["id"])+"/complete", map[string]any{"message": "A done"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(childB["id"])+"/complete", map[string]any{"message": "B done"})
	resume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume")
	if resume["pr_suggestion"] == nil {
		t.Fatalf("resume should include pr_suggestion when stage completed: %#v", resume)
	}

	activated := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages/"+stringValue(stage2["id"])+"/activate", map[string]any{})
	gitSuggestion, _ := activated["git_suggestion"].(map[string]any)
	if gitSuggestion == nil || !strings.HasPrefix(stringValue(gitSuggestion["branch_name"]), "feature/") {
		t.Fatalf("activate stage should include git_suggestion branch_name: %#v", activated)
	}

	patchJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/context", map[string]any{
		"architecture_decisions": []string{"使用 Gin 路由，避免重复生成"},
		"reference_files":        []string{"backend/internal/api/router/router.go"},
		"context_doc_text":       "social_token TTL 固定 5 分钟",
	})
	taskCtx := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/context")
	if taskCtx["architecture_decisions"] == nil || taskCtx["reference_files"] == nil {
		t.Fatalf("task context patch/get failed: %#v", taskCtx)
	}
}

func TestDependsOnSummaryAndExecutionOrder(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "depends-summary.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "依赖摘要", "task_key": "DEP"})
	taskID := stringValue(task["id"])
	first := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "先完成", "node_key": "1"})
	second := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{
		"title":      "后执行",
		"node_key":   "2",
		"depends_on": []string{stringValue(first["id"])},
	})

	summary := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes?view_mode=summary&sort_by=path&limit=10")
	items, _ := summary["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("unexpected summary len: %#v", summary)
	}
	secondSummary, _ := items[1].(map[string]any)
	dependsOn, _ := secondSummary["depends_on"].([]any)
	if len(dependsOn) != 1 || stringValue(dependsOn[0]) != stringValue(first["id"]) {
		t.Fatalf("summary depends_on mismatch: %#v", secondSummary)
	}
	if secondSummary["depends_on_count"] != float64(1) {
		t.Fatalf("summary depends_on_count mismatch: %#v", secondSummary)
	}

	resumeBefore := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume")
	recommendedBefore, _ := resumeBefore["recommended_action"].(map[string]any)
	if stringValue(recommendedBefore["node_id"]) != stringValue(first["id"]) {
		t.Fatalf("dependency should keep prerequisite first: %#v", resumeBefore["recommended_action"])
	}

	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(first["id"])+"/complete", map[string]any{"message": "done"})
	resumeAfter := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume")
	recommendedAfter, _ := resumeAfter["recommended_action"].(map[string]any)
	if stringValue(recommendedAfter["node_id"]) != stringValue(second["id"]) {
		t.Fatalf("dependent node should become next after prerequisite done: %#v", resumeAfter["recommended_action"])
	}
}

func TestFocusFilterAppliesBeforePagination(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "focus-before-limit.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "focus limit", "task_key": "FCS"})
	taskID := stringValue(task["id"])
	doneA := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "A", "node_key": "1"})
	doneB := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "B", "node_key": "2"})
	readyC := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "C", "node_key": "3"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(doneA["id"])+"/complete", map[string]any{"message": "done"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(doneB["id"])+"/complete", map[string]any{"message": "done"})

	summary := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes?view_mode=summary&filter_mode=focus&sort_by=path&limit=1")
	items, _ := summary["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("focus summary should keep ready node even when it is beyond the raw first page: %#v", summary)
	}
	firstItem, _ := items[0].(map[string]any)
	if stringValue(firstItem["id"]) != stringValue(readyC["id"]) {
		t.Fatalf("focus summary should page over focused rows, got %#v", firstItem)
	}
}

func TestResumeHonorsTreePagingAndResumeContextSiblings(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "resume-paging.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "resume 分页", "task_key": "RSM"})
	taskID := stringValue(task["id"])
	parent := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "父节点", "node_key": "1", "kind": "group"})
	childA := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "子节点A", "parent_node_id": parent["id"], "node_key": "1"})
	childB := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "子节点B", "parent_node_id": parent["id"], "node_key": "2"})
	childC := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "子节点C", "parent_node_id": parent["id"], "node_key": "3"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "根节点Z", "node_key": "9"})

	resume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume?sort_by=path&sort_order=desc&limit=1")
	tree, _ := resume["tree"].([]any)
	if len(tree) != 1 {
		t.Fatalf("resume tree should honor limit: %#v", resume)
	}
	if resume["tree_cursor"] == nil {
		t.Fatalf("resume tree should expose cursor when paged: %#v", resume)
	}

	ctx := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(childB["id"])+"/resume-context")
	siblings, _ := ctx["siblings"].([]any)
	if len(siblings) != 2 {
		t.Fatalf("resume context should only include same-parent siblings: %#v", ctx)
	}
	ids := map[string]struct{}{}
	for _, raw := range siblings {
		item, _ := raw.(map[string]any)
		ids[stringValue(item["node_id"])] = struct{}{}
	}
	if _, ok := ids[stringValue(childA["id"])]; !ok {
		t.Fatalf("missing sibling A: %#v", siblings)
	}
	if _, ok := ids[stringValue(childC["id"])]; !ok {
		t.Fatalf("missing sibling C: %#v", siblings)
	}

	childrenResume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume?view_mode=summary&parent_node_id="+url.QueryEscape(stringValue(parent["id"])))
	childItems := workspaceAsItems(childrenResume["tree"])
	if len(childItems) != 3 {
		t.Fatalf("resume should honor parent_node_id filtering: %#v", childrenResume)
	}
}

func TestNodesDefaultSummaryAndSubtreeFilters(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "nodes-summary-subtree.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "子树过滤", "task_key": "TREE2"})
	taskID := stringValue(task["id"])
	group := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "分组", "node_key": "1", "kind": "group"})
	child := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "子节点", "parent_node_id": group["id"], "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "孙节点", "parent_node_id": child["id"], "node_key": "1"})

	defaultList := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes")
	items := workspaceAsItems(defaultList["items"])
	if len(items) == 0 || items[0]["next_action"] == nil {
		t.Fatalf("default nodes should return summary wrapper: %#v", defaultList)
	}

	children := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes?view_mode=summary&parent_node_id="+url.QueryEscape(stringValue(group["id"])))
	childItems := workspaceAsItems(children["items"])
	if len(childItems) != 1 || stringValue(childItems[0]["id"]) != stringValue(child["id"]) {
		t.Fatalf("parent_node_id filter mismatch: %#v", children)
	}

	subtree := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes?view_mode=summary&subtree_root_node_id="+url.QueryEscape(stringValue(group["id"]))+"&max_relative_depth=1&sort_by=path")
	subtreeItems := workspaceAsItems(subtree["items"])
	if len(subtreeItems) != 2 {
		t.Fatalf("subtree summary should include root + direct child only: %#v", subtree)
	}
}

func TestResumeIncludesOptInHeavySections(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "resume-includes.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "resume include", "task_key": "RIN"})
	taskID := stringValue(task["id"])
	stage := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/stages", map[string]any{"title": "阶段一", "node_key": "S1", "activate": true})
	node := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "执行节点", "node_key": "1"})
	run := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(node["id"])+"/runs", map[string]any{
		"actor": map[string]any{"tool": "test"},
	})
	postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/logs", map[string]any{"kind": "stdout", "content": "hello"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/artifacts", map[string]any{"node_id": node["id"], "kind": "link", "title": "文档", "uri": "https://example.com"})

	lightResume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume")
	if len(workspaceAsItems(lightResume["recent_events"])) != 0 {
		t.Fatalf("resume should omit events by default: %#v", lightResume)
	}
	if len(workspaceAsItems(lightResume["recent_runs"])) != 0 {
		t.Fatalf("resume should omit runs by default: %#v", lightResume)
	}
	if len(workspaceAsItems(lightResume["artifacts"])) != 0 {
		t.Fatalf("resume should omit artifacts by default: %#v", lightResume)
	}
	if lightResume["next_node"] != nil {
		t.Fatalf("resume should omit next node context by default: %#v", lightResume)
	}
	if lightResume["task_memory"] != nil || lightResume["current_stage_memory"] != nil {
		t.Fatalf("resume should omit heavy memory objects by default: %#v", lightResume)
	}
	if lightResume["task_memory_summary"] == nil {
		t.Fatalf("resume should keep task memory summary by default: %#v", lightResume)
	}
	if lightResume["next_node_summary"] == nil {
		t.Fatalf("resume should keep next_node_summary by default: %#v", lightResume)
	}

	fullResume := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/resume?include=events,runs,artifacts,next_node_context,task_memory,stage_memory")
	if len(workspaceAsItems(fullResume["recent_events"])) == 0 {
		t.Fatalf("resume include should return events: %#v", fullResume)
	}
	if len(workspaceAsItems(fullResume["recent_runs"])) == 0 {
		t.Fatalf("resume include should return runs: %#v", fullResume)
	}
	if len(workspaceAsItems(fullResume["artifacts"])) != 1 {
		t.Fatalf("resume include should return artifacts: %#v", fullResume)
	}
	if fullResume["next_node"] == nil {
		t.Fatalf("resume include should return next node context: %#v", fullResume)
	}
	if fullResume["task_memory"] == nil {
		t.Fatalf("resume include should return task_memory: %#v", fullResume)
	}
	if fullResume["current_stage_memory"] == nil {
		t.Fatalf("resume include should return current_stage_memory: %#v", fullResume)
	}
	if stringValue(stage["id"]) == "" {
		t.Fatalf("stage missing id: %#v", stage)
	}
}

func TestNodeContextPresetRunAndArtifactSummary(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "context-preset-run-artifact.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "上下文预设", "task_key": "CTX"})
	taskID := stringValue(task["id"])
	node := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes", map[string]any{"title": "执行节点", "node_key": "1"})
	runWrap := postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/nodes/"+stringValue(node["id"])+"/claim-and-start-run", map[string]any{
		"actor": map[string]any{"tool": "test", "agent_id": "worker"},
	})
	run, _ := runWrap["run"].(map[string]any)
	if run == nil {
		t.Fatalf("claim-and-start-run should return nested run: %#v", runWrap)
	}
	postJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["id"])+"/logs", map[string]any{"kind": "stdout", "content": "hello"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/artifacts", map[string]any{"node_id": node["id"], "kind": "link", "title": "文档", "uri": "https://example.com"})

	summaryCtx := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/context?preset=summary")
	if summaryCtx["memory"] != nil || summaryCtx["recent_runs"] != nil {
		t.Fatalf("summary preset should stay light: %#v", summaryCtx)
	}
	workCtx := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/context?preset=work")
	if workCtx["memory"] == nil || workCtx["recent_runs"] == nil || workCtx["artifacts"] == nil {
		t.Fatalf("work preset should include execution context: %#v", workCtx)
	}

	runSummary := getJSON[map[string]any](t, server.URL+"/v1/nodes/"+stringValue(node["id"])+"/runs?view_mode=summary&limit=1")
	runItems := workspaceAsItems(runSummary["items"])
	if len(runItems) != 1 || runItems[0]["logs"] != nil {
		t.Fatalf("summary runs should paginate without logs: %#v", runSummary)
	}
	runNoLogs := getJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["run_id"]))
	if runNoLogs["logs"] != nil {
		t.Fatalf("get run default should omit logs: %#v", runNoLogs)
	}
	runWithLogs := getJSON[map[string]any](t, server.URL+"/v1/runs/"+stringValue(run["run_id"])+"?include_logs=true")
	if runWithLogs["logs"] == nil {
		t.Fatalf("get run with include_logs should include logs: %#v", runWithLogs)
	}

	artifacts := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+taskID+"/artifacts?view_mode=summary&limit=10")
	artifactItems := workspaceAsItems(artifacts["items"])
	if len(artifactItems) != 1 {
		t.Fatalf("artifact summary route mismatch: %#v", artifacts)
	}
}

func TestProjectsSummaryWithStats(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "project-summary-stats.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	project := postJSON[map[string]any](t, server.URL+"/v1/projects", map[string]any{"name": "项目汇总"})
	projectID := stringValue(project["id"])
	task := postJSON[map[string]any](t, server.URL+"/v1/tasks", map[string]any{"title": "任务A", "project_id": projectID})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+stringValue(task["id"])+"/nodes", map[string]any{"title": "节点1"})

	projects := getJSON[[]map[string]any](t, server.URL+"/v1/projects?view_mode=summary_with_stats")
	found := false
	for _, item := range projects {
		if stringValue(item["id"]) == projectID {
			found = item["_summary"] != nil
			break
		}
	}
	if !found {
		t.Fatalf("projects summary_with_stats should include _summary: %#v", projects)
	}
}

func TestMCPChildrenAndSubtreeSummaryTools(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "mcp-children-subtree.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()

	task, err := app.createTask(context.Background(), taskCreate{Title: "MCP 子树", TaskKey: nullableOptString("MCP")})
	if err != nil {
		t.Fatal(err)
	}
	taskID := stringValue(task["id"])
	parent, err := app.createNode(context.Background(), taskID, nodeCreate{Title: "父节点", NodeKey: nullableOptString("1"), Kind: "group"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := app.createNode(context.Background(), taskID, nodeCreate{Title: "子节点", ParentNodeID: nullableOptString(stringValue(parent["id"])), NodeKey: nullableOptString("1")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.createNode(context.Background(), taskID, nodeCreate{Title: "孙节点", ParentNodeID: nullableOptString(stringValue(child["id"])), NodeKey: nullableOptString("1")}); err != nil {
		t.Fatal(err)
	}

	s := &mcpServer{app: app}
	childrenWrap := callMCPToolJSON(t, s, "task_tree_list_children", map[string]any{
		"task_id": taskID,
		"node_id": stringValue(parent["id"]),
		"limit":   10,
	})
	childItems := workspaceAsItems(childrenWrap["items"])
	if len(childItems) != 1 || stringValue(childItems[0]["id"]) != stringValue(child["id"]) {
		t.Fatalf("mcp children summary mismatch: %#v", childrenWrap)
	}

	subtreeWrap := callMCPToolJSON(t, s, "task_tree_list_subtree_summary", map[string]any{
		"task_id":            taskID,
		"root_node_id":       stringValue(parent["id"]),
		"max_relative_depth": 1,
		"limit":              10,
	})
	subtreeItems := workspaceAsItems(subtreeWrap["items"])
	if len(subtreeItems) != 2 {
		t.Fatalf("mcp subtree summary should include root + direct child only: %#v", subtreeWrap)
	}
}

func TestAIToolGetTaskRequiresExplicitTree(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TTS_DB_PATH", filepath.Join(tmp, "ai-get-task-summary.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()

	task, err := app.createTask(context.Background(), taskCreate{Title: "AI 摘要", TaskKey: nullableOptString("AIT")})
	if err != nil {
		t.Fatal(err)
	}
	taskID := stringValue(task["id"])
	if _, err := app.createNode(context.Background(), taskID, nodeCreate{Title: "节点一", NodeKey: nullableOptString("1")}); err != nil {
		t.Fatal(err)
	}

	light := app.executeAITool(context.Background(), "get_task", json.RawMessage(`{"task_id":"`+taskID+`"}`))
	if !strings.Contains(light, "节点树：未加载") {
		t.Fatalf("get_task should stay summary-first by default: %s", light)
	}

	full := app.executeAITool(context.Background(), "get_task", json.RawMessage(`{"task_id":"`+taskID+`","include_tree":true}`))
	if strings.Contains(full, "节点树：未加载") || !strings.Contains(full, "AIT/1") {
		t.Fatalf("get_task include_tree should load tree summary: %s", full)
	}
}

func postJSON[T any](t *testing.T, u string, body map[string]any) T {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("POST %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func callMCPToolJSON(t *testing.T, server *mcpServer, name string, args map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.callTool(raw)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := result["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatalf("mcp tool %s returned empty content: %#v", name, result)
	}
	text := stringValue(content[0]["text"])
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("mcp tool %s returned invalid json: %s", name, text)
	}
	return out
}

func postJSONArray[T any](t *testing.T, u string, body any) T {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("POST %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func postJSONArrayRaw(t *testing.T, u string, body any, expectedStatus int) []byte {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		t.Fatalf("POST %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	return raw
}

func postNoBodyJSON[T any](t *testing.T, u string) T {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("POST %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func postJSONRaw(t *testing.T, u string, body map[string]any, expectedStatus int) []byte {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		t.Fatalf("POST %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	return raw
}

func getJSON[T any](t *testing.T, u string) T {
	t.Helper()
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("GET %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func patchJSON[T any](t *testing.T, u string, body map[string]any) T {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, u, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("PATCH %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func patchJSONRaw(t *testing.T, u string, body map[string]any, expectedStatus int) []byte {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, u, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		t.Fatalf("PATCH %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	return raw
}

func deleteJSON(t *testing.T, u string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete %s status=%d body=%s", u, resp.StatusCode, string(data))
	}
}

func deleteReq(t *testing.T, u string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("DELETE %s -> %d %s", u, resp.StatusCode, string(raw))
	}
}

func uploadFile(t *testing.T, u, field, name string, content []byte, fields map[string]string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	_ = writer.Close()
	req, _ := http.NewRequest(http.MethodPost, u, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("UPLOAD %s -> %d %s", u, resp.StatusCode, string(raw))
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func postForm(t *testing.T, u string, values url.Values) *http.Response {
	t.Helper()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.PostForm(u, values)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

package tasktree

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	allNodes := getJSON[[]map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes")
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
	arts := getJSON[[]map[string]any](t, server.URL+"/v1/tasks/"+tidc+"/artifacts")
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
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/nodes", map[string]any{"parent_node_id": orderParent["id"], "title": "第一叶子", "node_key": "1"})
	postJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/nodes", map[string]any{"title": "较晚创建的根节点", "node_key": "3"})
	resumeOrdered := getJSON[map[string]any](t, server.URL+"/v1/tasks/"+tid4o+"/resume")
	nextWrap, _ := resumeOrdered["next_node"].(map[string]any)
	nextOrdered, _ := nextWrap["node"].(map[string]any)
	if stringValue(nextOrdered["path"]) != "O/1/1" {
		t.Fatalf("resume ordered next path = %v", nextOrdered["path"])
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
	if resumeAfterSplit["next_node"] == nil {
		t.Fatal("resume after split missing next node")
	}
	if !strings.Contains(stringValue(childAfterDone["path"]), "RB/1/2") {
		t.Fatalf("child path after split = %v", childAfterDone["path"])
	}
	splitNodes := getJSON[[]map[string]any](t, server.URL+"/v1/tasks/"+tid4b+"/nodes")
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
	nodes := getJSON[[]map[string]any](t, server.URL+"/v1/tasks/"+tid+"/nodes")
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

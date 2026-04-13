package tasktree

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPInitializeAndToolCall(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "mcp.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := &mcpServer{app: app}

	initReq := mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	initResp := server.handle(initReq)
	if initResp == nil || initResp.Error != nil {
		t.Fatalf("initialize failed: %#v", initResp)
	}

	listReq := mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	listResp := server.handle(listReq)
	if listResp == nil || listResp.Error != nil {
		t.Fatalf("tools/list failed: %#v", listResp)
	}
	listResult, _ := listResp.Result.(map[string]any)
	listJSON := mustJSONString(t, listResult)
	if listResult == nil || !strings.Contains(listJSON, "task_tree_delete_task") || !strings.Contains(listJSON, "task_tree_upload_artifact") || !strings.Contains(listJSON, "task_tree_list_nodes_summary") || !strings.Contains(listJSON, "task_tree_focus_nodes") {
		t.Fatalf("tools/list missing parity tools: %#v", listResp.Result)
	}

	callReq := mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "MCP 创建任务",
				"task_key": "MCP",
				"goal":     "验证 MCP tools/call 可创建任务",
			},
		},
	})
	callResp := server.handle(callReq)
	if callResp == nil || callResp.Error != nil {
		t.Fatalf("tools/call failed: %#v", callResp)
	}
	result, _ := callResp.Result.(map[string]any)
	if result == nil {
		t.Fatal("missing tool result")
	}
	content, _ := result["content"].([]map[string]any)
	if content == nil || len(content) == 0 {
		t.Fatal("missing MCP content")
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(content[0]["text"].(string)), &created); err != nil {
		t.Fatal(err)
	}
	taskID := stringValue(created["id"])
	if taskID == "" {
		t.Fatal("missing task id in create result")
	}

	_ = server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      31,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_node",
			"arguments": map[string]any{
				"task_id": taskID,
				"title":   "MCP 节点",
			},
		},
	}))
	summaryReq := mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      32,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_list_nodes_summary",
			"arguments": map[string]any{
				"task_id": taskID,
				"limit":   20,
			},
		},
	})
	summaryResp := server.handle(summaryReq)
	if summaryResp == nil || summaryResp.Error != nil {
		t.Fatalf("summary tool failed: %#v", summaryResp)
	}
	summaryResult, _ := summaryResp.Result.(map[string]any)
	if !strings.Contains(mustJSONString(t, summaryResult), "next_action") {
		t.Fatalf("summary tool payload missing next_action: %#v", summaryResult)
	}

	deleteReq := mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_delete_task",
			"arguments": map[string]any{
				"task_id": taskID,
			},
		},
	})
	deleteResp := server.handle(deleteReq)
	if deleteResp == nil || deleteResp.Error != nil {
		t.Fatalf("delete task failed: %#v", deleteResp)
	}
}

func TestMCPHTTPTransport(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "mcp-http.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	initResp := postRPCJSON(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	}, "", "")
	if initResp.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", initResp.StatusCode)
	}
	if got := initResp.Header.Get("MCP-Protocol-Version"); got != mcpProtocolVersionLatest {
		t.Fatalf("mcp protocol header = %q", got)
	}
	var initBody rpcResponse
	if err := json.NewDecoder(initResp.Body).Decode(&initBody); err != nil {
		t.Fatal(err)
	}
	_ = initResp.Body.Close()
	if initBody.Error != nil {
		t.Fatalf("initialize error = %#v", initBody.Error)
	}

	listResp := postRPCJSON(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}, "", "")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("tools/list status = %d", listResp.StatusCode)
	}
	var listBody rpcResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	_ = listResp.Body.Close()
	listResult, _ := listBody.Result.(map[string]any)
	listJSON := mustJSONString(t, listResult)
	if listResult == nil || !strings.Contains(listJSON, "task_tree_create_task") || !strings.Contains(listJSON, "task_tree_list_events") {
		t.Fatalf("tools/list result = %#v", listBody.Result)
	}

	callResp := postRPCJSON(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "HTTP MCP 创建任务",
				"task_key": "HTTP",
				"goal":     "验证 /mcp 地址可直接创建任务",
			},
		},
	}, "", "")
	if callResp.StatusCode != http.StatusOK {
		t.Fatalf("tools/call status = %d", callResp.StatusCode)
	}
	var callBody rpcResponse
	if err := json.NewDecoder(callResp.Body).Decode(&callBody); err != nil {
		t.Fatal(err)
	}
	_ = callResp.Body.Close()
	callResult, _ := callBody.Result.(map[string]any)
	if callResult == nil || !strings.Contains(mustJSONString(t, callResult), "HTTP MCP 创建任务") {
		t.Fatalf("tools/call result = %#v", callBody.Result)
	}

	sseResp := postRPCSSE(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "initialize",
		"params":  map[string]any{},
	}, "")
	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("sse initialize status = %d", sseResp.StatusCode)
	}
	sessionID := sseResp.Header.Get(mcpSessionHeader)
	if sessionID == "" {
		t.Fatal("missing mcp session header")
	}
	sseEvent := mustReadSSEEvent(t, sseResp.Body)
	_ = sseResp.Body.Close()
	if !strings.Contains(sseEvent.Data, "\"protocolVersion\":\""+mcpProtocolVersionLatest+"\"") {
		t.Fatalf("unexpected sse init payload = %s", sseEvent.Data)
	}

	sseCallResp := postRPCSSE(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "HTTP SSE Task",
				"task_key": "SSE",
			},
		},
	}, sessionID)
	if sseCallResp.StatusCode != http.StatusOK {
		t.Fatalf("sse tools/call status = %d", sseCallResp.StatusCode)
	}
	callEvent := mustReadSSEEvent(t, sseCallResp.Body)
	_ = sseCallResp.Body.Close()
	if callEvent.ID == "" || !strings.Contains(callEvent.Data, "HTTP SSE Task") {
		t.Fatalf("unexpected sse call payload = %#v", callEvent)
	}

	streamReq, err := http.NewRequest(http.MethodGet, server.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	streamReq.Header.Set(mcpSessionHeader, sessionID)
	streamReq.Header.Set(mcpLastEventIDHeader, "1")
	streamResp, err := http.DefaultClient.Do(streamReq)
	if err != nil {
		t.Fatal(err)
	}
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /mcp status = %d", streamResp.StatusCode)
	}
	streamEvent := mustReadSSEEvent(t, streamResp.Body)
	_ = streamResp.Body.Close()
	if streamEvent.ID != callEvent.ID {
		t.Fatalf("replayed event id = %s, want %s", streamEvent.ID, callEvent.ID)
	}

	notifyResp := postRPCJSON(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	}, sessionID, "")
	if notifyResp.StatusCode != http.StatusAccepted {
		t.Fatalf("notification status = %d", notifyResp.StatusCode)
	}
	_ = notifyResp.Body.Close()

	getResp, err := http.Get(server.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /mcp status = %d", getResp.StatusCode)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, server.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	deleteReq.Header.Set(mcpSessionHeader, sessionID)
	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /mcp status = %d", deleteResp.StatusCode)
	}
	_ = deleteResp.Body.Close()

	badOriginResp := postRPCJSON(t, server.URL+"/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "initialize",
		"params":  map[string]any{},
	}, "", "https://example.com")
	if badOriginResp.StatusCode != http.StatusForbidden {
		t.Fatalf("bad origin status = %d", badOriginResp.StatusCode)
	}
	_ = badOriginResp.Body.Close()
}

func TestMCPCreateTaskWithInitialNodeTree(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "mcp-seed.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := &mcpServer{app: app}

	callResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      11,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "MCP 一次性建树",
				"task_key": "MCPTREE",
				"nodes": []map[string]any{
					{
						"title": "分析需求",
					},
					{
						"title": "实现功能",
						"children": []map[string]any{
							{
								"title": "扩展类型",
							},
							{
								"title": "补 MCP",
								"children": []map[string]any{
									{
										"title": "调整 schema",
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	if callResp == nil || callResp.Error != nil {
		t.Fatalf("create task with nodes failed: %#v", callResp)
	}
	result, _ := callResp.Result.(map[string]any)
	content, _ := result["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatal("missing MCP content")
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(content[0]["text"].(string)), &created); err != nil {
		t.Fatal(err)
	}
	taskID := stringValue(created["id"])
	if taskID == "" {
		t.Fatal("missing task id")
	}

	nodesResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      12,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_list_nodes",
			"arguments": map[string]any{
				"task_id":    taskID,
				"sort_by":    "path",
				"sort_order": "asc",
				"limit":      20,
			},
		},
	}))
	if nodesResp == nil || nodesResp.Error != nil {
		t.Fatalf("list nodes failed: %#v", nodesResp)
	}
	nodesResult, _ := nodesResp.Result.(map[string]any)
	nodesJSON := mustJSONString(t, nodesResult)
	for _, path := range []string{"MCPTREE/1", "MCPTREE/2", "MCPTREE/2/1", "MCPTREE/2/2", "MCPTREE/2/2/1"} {
		if !strings.Contains(nodesJSON, path) {
			t.Fatalf("nodes payload missing path %s: %s", path, nodesJSON)
		}
	}
}

func TestMCPArrayArgsStringCompatibility(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "mcp-array-compat.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := &mcpServer{app: app}

	stagesJSON := `[{"title":"阶段一","node_key":"S1"},{"title":"阶段二","node_key":"S2"}]`
	nodesJSON := `[{"title":"种子节点","node_key":"N1"}]`
	createResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      21,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "字符串数组兼容",
				"task_key": "STRARR",
				"stages":   stagesJSON,
				"nodes":    nodesJSON,
			},
		},
	}))
	if createResp == nil || createResp.Error != nil {
		t.Fatalf("create_task with string arrays failed: %#v", createResp)
	}
	createResult, _ := createResp.Result.(map[string]any)
	createContent, _ := createResult["content"].([]map[string]any)
	if len(createContent) == 0 {
		t.Fatal("missing create_task content")
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(createContent[0]["text"].(string)), &created); err != nil {
		t.Fatal(err)
	}
	taskID := stringValue(created["id"])
	if taskID == "" {
		t.Fatal("missing task id")
	}

	stagesResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      22,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_list_stages",
			"arguments": map[string]any{
				"task_id": taskID,
			},
		},
	}))
	if stagesResp == nil || stagesResp.Error != nil {
		t.Fatalf("list_stages failed: %#v", stagesResp)
	}
	stagesResult, _ := stagesResp.Result.(map[string]any)
	stageContent, _ := stagesResult["content"].([]map[string]any)
	var stageItems []map[string]any
	if len(stageContent) == 0 || json.Unmarshal([]byte(stageContent[0]["text"].(string)), &stageItems) != nil {
		t.Fatalf("invalid stages payload: %#v", stagesResult)
	}
	if len(stageItems) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stageItems))
	}
	stage2ID := stringValue(stageItems[1]["id"])
	if stage2ID == "" {
		t.Fatal("missing stage2 id")
	}

	batchNodesJSON := `[{"title":"挂到阶段二","node_key":"G2","stage_node_id":"` + stage2ID + `"}]`
	batchResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      23,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_batch_create_nodes",
			"arguments": map[string]any{
				"task_id": taskID,
				"nodes":   batchNodesJSON,
			},
		},
	}))
	if batchResp == nil || batchResp.Error != nil {
		t.Fatalf("batch_create_nodes with string array failed: %#v", batchResp)
	}
	batchResult, _ := batchResp.Result.(map[string]any)
	batchContent, _ := batchResult["content"].([]map[string]any)
	var batchCreated map[string]any
	if len(batchContent) == 0 || json.Unmarshal([]byte(batchContent[0]["text"].(string)), &batchCreated) != nil {
		t.Fatalf("invalid batch payload: %#v", batchResult)
	}
	createdItems, _ := batchCreated["created"].([]any)
	if len(createdItems) != 1 {
		t.Fatalf("expected 1 created node, got %d", len(createdItems))
	}
	createdNode, _ := createdItems[0].(map[string]any)
	if stringValue(createdNode["parent_node_id"]) != stage2ID {
		t.Fatalf("batch node parent should be stage2, got %v", createdNode["parent_node_id"])
	}
}

func TestMCPExtendedToolsAndAliases(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "mcp-extended.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := &mcpServer{app: app}

	dryResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      31,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "MCP Dry Run",
				"task_key": "MCPDRY",
				"dry_run":  true,
				"stages": []map[string]any{
					{"title": "阶段A", "node_key": "S1"},
				},
				"nodes": []map[string]any{
					{"title": "步骤1", "node_key": "N1"},
					{"title": "步骤2", "node_key": "N2", "depends_on_keys": []string{"N1"}},
				},
			},
		},
	}))
	if dryResp == nil || dryResp.Error != nil {
		t.Fatalf("mcp dry-run failed: %#v", dryResp)
	}
	dryResult, _ := dryResp.Result.(map[string]any)
	dryContentItems, _ := dryResult["content"].([]map[string]any)
	if len(dryContentItems) == 0 {
		t.Fatalf("missing dry-run content: %#v", dryResp.Result)
	}
	var dryPayload map[string]any
	if err := json.Unmarshal([]byte(stringValue(dryContentItems[0]["text"])), &dryPayload); err != nil {
		t.Fatalf("invalid dry-run payload: %v", err)
	}
	if dryPayload["dry_run"] != true || dryPayload["validated"] != true {
		t.Fatalf("unexpected dry-run payload: %#v", dryPayload)
	}

	createResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      32,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_create_task",
			"arguments": map[string]any{
				"title":    "MCP 别名任务",
				"task_key": "MCPALIAS",
				"stages": []map[string]any{
					{"title": "阶段一", "node_key": "S1", "activate": true},
					{"title": "阶段二", "node_key": "S2"},
				},
			},
		},
	}))
	if createResp == nil || createResp.Error != nil {
		t.Fatalf("create task failed: %#v", createResp)
	}
	var created map[string]any
	createResult, _ := createResp.Result.(map[string]any)
	createContent, _ := createResult["content"].([]map[string]any)
	if len(createContent) == 0 || json.Unmarshal([]byte(createContent[0]["text"].(string)), &created) != nil {
		t.Fatalf("invalid create payload: %#v", createResp.Result)
	}
	taskID := stringValue(created["id"])
	if taskID == "" {
		t.Fatal("missing task id")
	}

	stagesResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      33,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_list_stages",
			"arguments": map[string]any{
				"task_id": taskID,
			},
		},
	}))
	if stagesResp == nil || stagesResp.Error != nil {
		t.Fatalf("list stages failed: %#v", stagesResp)
	}
	var stageItems []map[string]any
	stagesResult, _ := stagesResp.Result.(map[string]any)
	stagesContent, _ := stagesResult["content"].([]map[string]any)
	if len(stagesContent) == 0 || json.Unmarshal([]byte(stagesContent[0]["text"].(string)), &stageItems) != nil || len(stageItems) < 2 {
		t.Fatalf("invalid stages payload: %#v", stagesResp.Result)
	}
	stage2ID := stringValue(stageItems[1]["id"])

	aliasActivateResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      34,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree.activate_stage",
			"arguments": map[string]any{
				"task_id":       taskID,
				"stage_node_id": stage2ID,
			},
		},
	}))
	if aliasActivateResp == nil || aliasActivateResp.Error != nil {
		t.Fatalf("alias activate failed: %#v", aliasActivateResp)
	}
	if !strings.Contains(mustJSONString(t, aliasActivateResp.Result), "git_suggestion") {
		t.Fatalf("activate result missing git_suggestion: %#v", aliasActivateResp.Result)
	}

	patchCtxResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      35,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_patch_task_context",
			"arguments": map[string]any{
				"task_id":                taskID,
				"architecture_decisions": []string{"统一走 gin 路由"},
				"reference_files":        []string{"backend/internal/api/router/router.go"},
				"context_doc_text":       "social_token TTL = 5m",
			},
		},
	}))
	if patchCtxResp == nil || patchCtxResp.Error != nil {
		t.Fatalf("patch task context failed: %#v", patchCtxResp)
	}
	getCtxResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      36,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_get_task_context",
			"arguments": map[string]any{
				"task_id": taskID,
			},
		},
	}))
	if getCtxResp == nil || getCtxResp.Error != nil {
		t.Fatalf("get task context failed: %#v", getCtxResp)
	}
	if !strings.Contains(mustJSONString(t, getCtxResp.Result), "architecture_decisions") {
		t.Fatalf("task context payload missing architecture_decisions: %#v", getCtxResp.Result)
	}

	treeResp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      37,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "task_tree_tree_view",
			"arguments": map[string]any{
				"task_id": taskID,
			},
		},
	}))
	if treeResp == nil || treeResp.Error != nil {
		t.Fatalf("tree view failed: %#v", treeResp)
	}
	treeResult, _ := treeResp.Result.(map[string]any)
	treeContent, _ := treeResult["content"].([]map[string]any)
	if len(treeContent) == 0 {
		t.Fatalf("missing tree view content: %#v", treeResp.Result)
	}
	var treePayload map[string]any
	if err := json.Unmarshal([]byte(stringValue(treeContent[0]["text"])), &treePayload); err != nil {
		t.Fatalf("invalid tree view payload: %v", err)
	}
	if treePayload["lines"] == nil {
		t.Fatalf("tree view missing lines field: %#v", treePayload)
	}
}

func mustRPCRequest(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func postRPCJSON(t *testing.T, target string, payload map[string]any, sessionID, origin string) *http.Response {
	return postRPCWithAccept(t, target, payload, sessionID, origin, "application/json")
}

func postRPCSSE(t *testing.T, target string, payload map[string]any, sessionID string) *http.Response {
	return postRPCWithAccept(t, target, payload, sessionID, "", "text/event-stream")
}

func postRPCWithAccept(t *testing.T, target string, payload map[string]any, sessionID, origin, accept string) *http.Response {
	t.Helper()
	body := mustRPCRequest(t, payload)
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", accept)
	if sessionID != "" {
		req.Header.Set(mcpSessionHeader, sessionID)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

type sseEvent struct {
	ID   string
	Name string
	Data string
}

func mustReadSSEEvent(t *testing.T, body io.Reader) sseEvent {
	t.Helper()
	reader := bufio.NewReader(body)
	out := sseEvent{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			out.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			out.Name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if out.Data != "" {
				out.Data += "\n"
			}
			out.Data += strings.TrimPrefix(line, "data: ")
		}
		if err == io.EOF {
			break
		}
	}
	return out
}

func mustJSONString(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

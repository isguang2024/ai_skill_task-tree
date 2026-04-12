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

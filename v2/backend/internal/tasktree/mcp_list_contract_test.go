package tasktree

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestListToolsReturnUnifiedShape guards the MCP list_* contract: every tool
// returning a collection must emit {items, has_more, next_cursor}. Adding a new
// list_* tool without this envelope will fail here. Pagination-less tools set
// has_more=false and next_cursor="".
func TestListToolsReturnUnifiedShape(t *testing.T) {
	t.Setenv("TTS_DB_PATH", filepath.Join(t.TempDir(), "list_contract.db"))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	server := &mcpServer{app: app}

	taskID := callToolForID(t, server, "task_tree_create_task", map[string]any{
		"title":    "契约任务",
		"task_key": "CONTRACT",
		"goal":     "验证 list_* envelope",
	})
	nodeID := callToolForID(t, server, "task_tree_create_node", map[string]any{
		"task_id": taskID,
		"title":   "契约节点",
	})

	cases := []struct {
		name string
		args map[string]any
	}{
		{"task_tree_list_tasks", map[string]any{}},
		{"task_tree_list_projects", map[string]any{}},
		{"task_tree_list_stages", map[string]any{"task_id": taskID}},
		{"task_tree_list_nodes", map[string]any{"task_id": taskID}},
		{"task_tree_list_events", map[string]any{"task_id": taskID}},
		{"task_tree_list_node_runs", map[string]any{"node_id": nodeID}},
		{"task_tree_list_artifacts", map[string]any{"task_id": taskID}},
		{"task_tree_work_items", map[string]any{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := callToolJSON(t, server, tc.name, tc.args)
			var envelope map[string]any
			if err := json.Unmarshal(payload, &envelope); err != nil {
				t.Fatalf("%s: response not JSON object: %s", tc.name, payload)
			}
			items, ok := envelope["items"]
			if !ok {
				t.Fatalf("%s: missing items key: %s", tc.name, payload)
			}
			if _, ok := items.([]any); !ok {
				t.Fatalf("%s: items is not an array: %T", tc.name, items)
			}
			if _, ok := envelope["has_more"]; !ok {
				t.Fatalf("%s: missing has_more key: %s", tc.name, payload)
			}
			if _, ok := envelope["next_cursor"]; !ok {
				t.Fatalf("%s: missing next_cursor key: %s", tc.name, payload)
			}
		})
	}
}

func callToolJSON(t *testing.T, server *mcpServer, tool string, args map[string]any) []byte {
	t.Helper()
	resp := server.handle(mustRPCRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}))
	if resp == nil || resp.Error != nil {
		t.Fatalf("%s failed: %#v", tool, resp)
	}
	result, _ := resp.Result.(map[string]any)
	content, _ := result["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatalf("%s: empty content", tool)
	}
	text, _ := content[0]["text"].(string)
	return []byte(text)
}

func callToolForID(t *testing.T, server *mcpServer, tool string, args map[string]any) string {
	t.Helper()
	payload := callToolJSON(t, server, tool, args)
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	id := stringValue(body["id"])
	if id == "" {
		t.Fatalf("%s: missing id in %s", tool, payload)
	}
	return id
}

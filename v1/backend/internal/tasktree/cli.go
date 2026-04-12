package tasktree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func baseURL() string {
	base := os.Getenv("TTS_BASE")
	if base == "" {
		base = "http://127.0.0.1:8879"
	}
	return strings.TrimRight(base, "/")
}

func runHealthcheck() int {
	resp, err := http.Get(baseURL() + "/healthz")
	if err != nil {
		fmt.Printf("DOWN  %s  (%T)\n", baseURL(), err)
		return 1
	}
	defer resp.Body.Close()
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("DOWN  %s  (%T)\n", baseURL(), err)
		return 1
	}
	if ok, _ := data["ok"].(bool); ok {
		fmt.Printf("UP  %s\n", baseURL())
		return 0
	}
	fmt.Printf("DEGRADED  %s\n", baseURL())
	return 1
}

func runClient(args []string) int {
	if len(args) == 0 {
		fmt.Println(clientUsage)
		return 1
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "resume":
		return clientResume(rest)
	case "next":
		return clientNext(rest)
	case "list":
		return clientList(rest)
	case "search":
		return clientGetJSON("/v1/search?q=" + url.QueryEscape(firstArg(rest)))
	case "work-items":
		status := "ready"
		if len(rest) > 0 {
			status = rest[0]
		}
		return clientGetJSON("/v1/work-items?status=" + url.QueryEscape(status))
	case "claim":
		return clientClaim(rest)
	case "release":
		return clientPostByNode(rest, "/release", nil)
	case "progress":
		return clientProgress(rest)
	case "complete":
		return clientComplete(rest)
	case "block":
		return clientBlock(rest)
	case "artifacts":
		return clientArtifacts(rest)
	case "create-task-stdin":
		return clientCreateFromStdin("/v1/tasks")
	case "create-node-stdin":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "missing task_id")
			return 1
		}
		return clientCreateFromStdin("/v1/tasks/" + rest[0] + "/nodes")
	default:
		fmt.Fprintf(os.Stderr, "unknown client cmd: %s\n", cmd)
		fmt.Println(clientUsage)
		return 1
	}
}

const clientUsage = `Task Tree Go client

task-tree-service.exe client resume <task_id>
task-tree-service.exe client next <task_id>
task-tree-service.exe client list [status]
task-tree-service.exe client search "<q>"
task-tree-service.exe client work-items [status]
task-tree-service.exe client claim <node_id> [lease_seconds]
task-tree-service.exe client release <node_id>
task-tree-service.exe client progress <node_id> <delta> [message] [idempotency_key]
task-tree-service.exe client complete <node_id> [message] [idempotency_key]
task-tree-service.exe client block <node_id> <reason>
task-tree-service.exe client artifacts <task_id> [node_id]
task-tree-service.exe client create-task-stdin
task-tree-service.exe client create-node-stdin <task_id>`

func clientResume(rest []string) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "missing task_id")
		return 1
	}
	var data map[string]any
	if err := clientFetchJSON("/v1/tasks/"+rest[0]+"/resume", &data); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	fmt.Print(renderResumePrompt(data))
	next := ""
	if nn, _ := data["next_node"].(map[string]any); nn != nil {
		if node, _ := nn["node"].(map[string]any); node != nil {
			next = stringValue(node["node_id"])
		}
	}
	fmt.Printf("\n---\n(raw next_node_id: %s)\n", next)
	return 0
}

func clientNext(rest []string) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "missing task_id")
		return 1
	}
	var data map[string]any
	if err := clientFetchJSON("/v1/tasks/"+rest[0]+"/resume", &data); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	nn, _ := data["next_node"].(map[string]any)
	node, _ := nn["node"].(map[string]any)
	if node == nil {
		fmt.Println("no actionable node")
		return 0
	}
	out := map[string]any{
		"task_id":             stringValue(data["task"].(map[string]any)["task_id"]),
		"node_id":             node["node_id"],
		"path":                node["path"],
		"title":               node["title"],
		"instruction":         node["instruction"],
		"acceptance_criteria": node["acceptance_criteria"],
	}
	printJSON(out)
	return 0
}

func clientList(rest []string) int {
	path := "/v1/tasks"
	if len(rest) > 0 {
		path += "?status=" + url.QueryEscape(rest[0])
	}
	var items []map[string]any
	if err := clientFetchJSON(path, &items); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	for _, item := range items {
		fmt.Printf("%s  [%-8s]  %5.0f%%  %s\n", stringValue(item["id"]), stringValue(item["status"]), floatValue(item["summary_percent"]), stringValue(item["title"]))
	}
	return 0
}

func clientClaim(rest []string) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "missing node_id")
		return 1
	}
	nodeID := rest[0]
	lease := 900
	if len(rest) > 1 {
		fmt.Sscanf(rest[1], "%d", &lease)
	}
	node := map[string]any{}
	if err := clientFetchJSON("/v1/nodes/"+nodeID, &node); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	body := map[string]any{"actor": map[string]any{"tool": "go-client", "agent_id": "cli"}, "lease_seconds": lease}
	return postJSONAndPrint(fmt.Sprintf("/v1/tasks/%s/nodes/%s/claim", stringValue(node["task_id"]), nodeID), body)
}

func clientPostByNode(rest []string, suffix string, body map[string]any) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "missing node_id")
		return 1
	}
	nodeID := rest[0]
	node := map[string]any{}
	if err := clientFetchJSON("/v1/nodes/"+nodeID, &node); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	return postJSONAndPrint(fmt.Sprintf("/v1/tasks/%s/nodes/%s%s", stringValue(node["task_id"]), nodeID, suffix), body)
}

func clientProgress(rest []string) int {
	if len(rest) < 2 {
		fmt.Fprintln(os.Stderr, "usage: progress <node_id> <delta> [message] [idempotency_key]")
		return 1
	}
	body := map[string]any{"delta_progress": parseFloat(rest[1]), "actor": map[string]any{"tool": "go-client", "agent_id": "cli"}}
	if len(rest) > 2 {
		body["message"] = rest[2]
	}
	if len(rest) > 3 {
		body["idempotency_key"] = rest[3]
	}
	return clientPostByNode(rest[:1], "/progress", body)
}

func clientComplete(rest []string) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "usage: complete <node_id> [message] [idempotency_key]")
		return 1
	}
	body := map[string]any{"actor": map[string]any{"tool": "go-client", "agent_id": "cli"}}
	if len(rest) > 1 {
		body["message"] = rest[1]
	}
	if len(rest) > 2 {
		body["idempotency_key"] = rest[2]
	}
	return clientPostByNode(rest[:1], "/complete", body)
}

func clientBlock(rest []string) int {
	if len(rest) < 2 {
		fmt.Fprintln(os.Stderr, "usage: block <node_id> <reason>")
		return 1
	}
	body := map[string]any{"reason": rest[1], "actor": map[string]any{"tool": "go-client", "agent_id": "cli"}}
	return clientPostByNode(rest[:1], "/block", body)
}

func clientArtifacts(rest []string) int {
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "usage: artifacts <task_id> [node_id]")
		return 1
	}
	path := "/v1/tasks/" + rest[0] + "/artifacts"
	if len(rest) > 1 {
		path = "/v1/tasks/" + rest[0] + "/nodes/" + rest[1] + "/artifacts"
	}
	return clientGetJSON(path)
}

func clientCreateFromStdin(path string) int {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	resp, err := http.Post(baseURL()+path, "application/json; charset=utf-8", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", string(out))
		return 2
	}
	fmt.Print(string(out))
	return 0
}

func clientGetJSON(path string) int {
	var data any
	if err := clientFetchJSON(path, &data); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	printJSON(data)
	return 0
}

func clientFetchJSON(path string, target any) error {
	resp, err := http.Get(baseURL() + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, target)
}

func postJSONAndPrint(path string, body map[string]any) int {
	data, _ := json.Marshal(body)
	resp, err := http.Post(baseURL()+path, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 2
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Fprintln(os.Stderr, "ERROR:", strings.TrimSpace(string(raw)))
		return 2
	}
	fmt.Print(string(raw))
	return 0
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func floatValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}

func renderResumePrompt(data map[string]any) string {
	task, _ := data["task"].(map[string]any)
	remaining, _ := data["remaining"].(map[string]any)
	lines := []string{
		fmt.Sprintf("# Task %s", stringValue(task["task_id"])),
		fmt.Sprintf("**Title:** %s", stringValue(task["title"])),
	}
	if goal := stringValue(task["goal"]); goal != "" {
		lines = append(lines, fmt.Sprintf("**Goal:** %s", goal))
	}
	lines = append(lines, fmt.Sprintf("**Progress:** %.0f%% · %d remaining", floatValue(task["summary_percent"]), int(floatValue(remaining["remaining_nodes"]))))
	if nn, _ := data["next_node"].(map[string]any); nn != nil {
		if node, _ := nn["node"].(map[string]any); node != nil {
			lines = append(lines, "", fmt.Sprintf("## Next: `%s` — %s", stringValue(node["path"]), stringValue(node["title"])), fmt.Sprintf("Node ID: `%s`", stringValue(node["node_id"])))
			if instruction := stringValue(node["instruction"]); instruction != "" {
				lines = append(lines, "", instruction)
			}
			if criteria, ok := node["acceptance_criteria"].([]any); ok && len(criteria) > 0 {
				lines = append(lines, "", "**Acceptance criteria:**")
				for _, c := range criteria {
					lines = append(lines, "- "+stringValue(c))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}


package tasktree

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type rpcEnvelope struct {
	Method json.RawMessage `json:"method"`
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

func (a *App) handleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !allowMCPOrigin(r) {
		writeJSON(w, http.StatusForbidden, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32600, Message: "invalid origin"},
		})
		return
	}
	switch r.Method {
	case http.MethodPost:
		a.handleMCPHTTPPost(w, r)
	case http.MethodGet:
		w.Header().Set("MCP-Protocol-Version", mcpProtocolVersionLatest)
		a.handleMCPHTTPGet(w, r)
	case http.MethodDelete:
		w.Header().Set("MCP-Protocol-Version", mcpProtocolVersionLatest)
		a.handleMCPHTTPDelete(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *App) handleMCPHTTPPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeRPCError(w, http.StatusBadRequest, -32700, err.Error())
		return
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		writeRPCError(w, http.StatusBadRequest, -32600, "empty request body")
		return
	}

	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		writeRPCError(w, http.StatusBadRequest, -32700, err.Error())
		return
	}

	sessionID := strings.TrimSpace(r.Header.Get(mcpSessionHeader))
	methodName := jsonRawString(env.Method)
	if sessionID == "" && methodName == "initialize" {
		sessionID = a.mcpSessions.create().ID
	}
	if sessionID != "" {
		if _, ok := a.mcpSessions.get(sessionID); !ok {
			writeRPCError(w, http.StatusNotFound, -32001, "session not found")
			return
		}
		w.Header().Set(mcpSessionHeader, sessionID)
	}

	if !hasRPCMethod(env) {
		if hasRPCResult(env) || hasRPCError(env) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeRPCError(w, http.StatusBadRequest, -32600, "missing method")
		return
	}

	server := &mcpServer{app: a}
	if !hasRPCID(env) {
		_ = server.handle(raw)
		w.Header().Set("MCP-Protocol-Version", mcpProtocolVersionLatest)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := server.handle(raw)

	// For initialize, extract negotiated version for the header; otherwise use latest.
	headerVersion := mcpProtocolVersionLatest
	if methodName == "initialize" && resp != nil && resp.Result != nil {
		if m, ok := resp.Result.(map[string]any); ok {
			if v, ok := m["protocolVersion"].(string); ok {
				headerVersion = v
			}
		}
	}
	w.Header().Set("MCP-Protocol-Version", headerVersion)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	stream := acceptsEventStream(r)
	event, stored, err := a.mcpSessions.appendResponse(sessionID, resp)
	if err != nil {
		writeRPCError(w, http.StatusInternalServerError, -32603, err.Error())
		return
	}
	if stream {
		if !stored {
			body, err := json.Marshal(resp)
			if err != nil {
				writeRPCError(w, http.StatusInternalServerError, -32603, err.Error())
				return
			}
			event = mcpStreamEvent{ID: "", Name: "message", Data: body}
		}
		writeMCPSSE(w, event)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handleMCPHTTPGet(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.Header.Get(mcpSessionHeader))
	if sessionID == "" {
		writeRPCError(w, http.StatusBadRequest, -32001, "missing session id")
		return
	}
	events, ok := a.mcpSessions.eventsSince(sessionID, strings.TrimSpace(r.Header.Get(mcpLastEventIDHeader)))
	if !ok {
		writeRPCError(w, http.StatusNotFound, -32001, "session not found")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set(mcpSessionHeader, sessionID)
	flusher, _ := w.(http.Flusher)
	if len(events) == 0 {
		_, _ = io.WriteString(w, ": keepalive\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
	for _, event := range events {
		_, _ = fmt.Fprintf(w, "id: %s\n", event.ID)
		_, _ = fmt.Fprintf(w, "event: %s\n", event.Name)
		for _, line := range strings.Split(string(event.Data), "\n") {
			_, _ = fmt.Fprintf(w, "data: %s\n", line)
		}
		_, _ = io.WriteString(w, "\n")
	}
	if flusher != nil {
		flusher.Flush()
	}
}

func (a *App) handleMCPHTTPDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.Header.Get(mcpSessionHeader))
	if sessionID == "" {
		writeRPCError(w, http.StatusBadRequest, -32001, "missing session id")
		return
	}
	if !a.mcpSessions.delete(sessionID) {
		writeRPCError(w, http.StatusNotFound, -32001, "session not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func acceptsEventStream(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept"), ",") {
		if strings.EqualFold(strings.TrimSpace(part), "text/event-stream") {
			return true
		}
	}
	return false
}

func writeMCPSSE(w http.ResponseWriter, event mcpStreamEvent) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Connection", "keep-alive")
	if event.ID != "" {
		_, _ = fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	name := event.Name
	if name == "" {
		name = "message"
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", name)
	for _, line := range strings.Split(string(event.Data), "\n") {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = io.WriteString(w, "\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeRPCError(w http.ResponseWriter, status, code int, msg string) {
	writeJSON(w, status, rpcResponse{
		JSONRPC: "2.0",
		Error:   &rpcError{Code: code, Message: msg},
	})
}

func allowMCPOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func jsonRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return ""
	}
	return out
}

func hasRPCMethod(env rpcEnvelope) bool {
	return len(env.Method) > 0 && strings.TrimSpace(string(env.Method)) != "null"
}

func hasRPCID(env rpcEnvelope) bool {
	if len(env.ID) == 0 {
		return false
	}
	return strings.TrimSpace(string(env.ID)) != "null"
}

func hasRPCResult(env rpcEnvelope) bool {
	return len(env.Result) > 0 && strings.TrimSpace(string(env.Result)) != "null"
}

func hasRPCError(env rpcEnvelope) bool {
	return len(env.Error) > 0 && strings.TrimSpace(string(env.Error)) != "null"
}


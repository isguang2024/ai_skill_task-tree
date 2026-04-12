package tasktree

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------- SSE helpers ----------

func writeSSE(w http.ResponseWriter, event string, data any) {
	var s string
	switch v := data.(type) {
	case string:
		s = v
	default:
		b, _ := json.Marshal(v)
		s = string(b)
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, s)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ---------- Streaming HTTP helper ----------

func doStream(ctx context.Context, url string, headers map[string]string, body any) (io.ReadCloser, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", aiUserAgent())
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		preview := raw
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(preview))
	}
	return resp.Body, nil
}

// ---------- SSE parser ----------

// parseSSEStream reads an SSE stream and calls onEvent for each complete event.
func parseSSEStream(r io.Reader, onEvent func(eventType, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 128*1024), 128*1024)
	var eventType string
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(dataLines) > 0 || eventType != "" {
				data := strings.Join(dataLines, "\n")
				if err := onEvent(eventType, data); err != nil {
					return err
				}
			}
			eventType = ""
			dataLines = nil
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return scanner.Err()
}

// ---------- OpenAI Responses API streaming call ----------

func callOpenAIResponsesStream(ctx context.Context, input []map[string]any, instructions string, tools []oaiTool, model string, onDelta func(string)) (*oaiResponse, error) {
	url := aiBaseURL() + "/v1/responses"
	headers := map[string]string{
		"authorization": "Bearer " + aiAPIKey(),
	}
	req := oaiRequest{
		Model:           aiModel(model),
		Input:           input,
		Instructions:    instructions,
		Tools:           tools,
		Store:           true,
		Stream:          true,
		MaxOutputTokens: aiMaxTokens,
	}
	if effort := aiReasoningEffort(); effort != "" {
		req.Reasoning = &oaiReasoningConfig{Effort: effort}
	}

	body, err := doStream(ctx, url, headers, req)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// Collect output items from granular events (more reliable than response.completed alone)
	var collectedItems []oaiOutItem
	var textBuf strings.Builder // accumulate full text for current message item
	var finalResp *oaiResponse

	err = parseSSEStream(body, func(eventType, data string) error {
		if data == "[DONE]" || data == "" {
			return nil
		}
		var evt map[string]any
		if jerr := json.Unmarshal([]byte(data), &evt); jerr != nil {
			return nil
		}
		evtType := asString(evt["type"])
		switch evtType {
		case "response.output_text.delta":
			// Stream text chunk to caller
			if delta, ok := evt["delta"].(string); ok {
				textBuf.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}

		case "response.output_item.done":
			// Each complete output item (message or function_call)
			if itemRaw, ok := evt["item"]; ok {
				b, _ := json.Marshal(itemRaw)
				var item oaiOutItem
				if jerr := json.Unmarshal(b, &item); jerr == nil {
					// For message items, fill content from accumulated text
					if item.Type == "message" && len(item.Content) == 0 && textBuf.Len() > 0 {
						item.Content = []oaiContent{{Type: "output_text", Text: textBuf.String()}}
						textBuf.Reset()
					}
					collectedItems = append(collectedItems, item)
				}
			}

		case "response.completed":
			// Use response.completed as authoritative source if it has output
			if respRaw, ok := evt["response"]; ok {
				b, _ := json.Marshal(respRaw)
				var resp oaiResponse
				if jerr := json.Unmarshal(b, &resp); jerr == nil {
					finalResp = &resp
				}
			}

		case "error":
			msg := asString(evt["message"])
			code := asString(evt["code"])
			return fmt.Errorf("stream error [%s]: %s", code, msg)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Prefer response.completed output; fall back to items collected from granular events
	if finalResp != nil && len(finalResp.Output) > 0 {
		return finalResp, nil
	}
	if finalResp != nil && finalResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", finalResp.Error.Message)
	}
	if len(collectedItems) > 0 {
		return &oaiResponse{Output: collectedItems}, nil
	}

	return nil, fmt.Errorf("OpenAI 返回空 output (streaming)")
}

// ---------- OpenAI Responses streaming agent loop ----------

func (a *App) runOpenAIAgentStream(ctx context.Context, w http.ResponseWriter, sessionID, userMsg, taskID, model string) (*aiChatResult, error) {
	history := a.aiSessions.getOAI(sessionID)
	history = append(history, map[string]any{"role": "user", "content": userMsg})

	sidSuffix := sessionID
	if len(sidSuffix) > 8 {
		sidSuffix = sidSuffix[:8]
	}
	instructions := aiSystemPrompt(taskID) + "\n[sid:" + sidSuffix + "]"
	tools := aiOpenAITools()

	var toolCallLog []string
	var roundText strings.Builder

	for i := 0; i < aiMaxIter; i++ {
		roundText.Reset()
		// Append round number to bust proxy cache between tool-call rounds
		roundInstructions := fmt.Sprintf("%s[r:%d]", instructions, i)

		resp, err := callOpenAIResponsesStream(ctx, history, roundInstructions, tools, model, func(delta string) {
			roundText.WriteString(delta)
			writeSSE(w, "delta", jsonMap{"text": delta})
		})
		if err != nil {
			return nil, err
		}

		hasFuncCalls := false
		for _, item := range resp.Output {
			if item.Type == "function_call" {
				hasFuncCalls = true
			}
		}

		// Append output to history
		for _, item := range resp.Output {
			entry := map[string]any{"type": item.Type}
			switch item.Type {
			case "message":
				entry["role"] = item.Role
				var texts []string
				for _, c := range item.Content {
					if c.Type == "output_text" {
						texts = append(texts, c.Text)
					}
				}
				if len(texts) == 1 {
					entry["content"] = texts[0]
				} else {
					entry["content"] = texts
				}
			case "function_call":
				entry["call_id"] = item.CallID
				entry["name"] = item.Name
				entry["arguments"] = item.Arguments
			}
			history = append(history, entry)
		}

		if !hasFuncCalls {
			break
		}

		// Execute tools
		for _, item := range resp.Output {
			if item.Type != "function_call" {
				continue
			}
			label := item.Name + "(" + summarizeRaw(item.Arguments) + ")"
			toolCallLog = append(toolCallLog, label)
			writeSSE(w, "tool", jsonMap{"name": label})

			result := a.executeAITool(ctx, item.Name, json.RawMessage(item.Arguments))
			history = append(history, map[string]any{
				"type":    "function_call_output",
				"call_id": item.CallID,
				"output":  result,
			})
		}
	}

	if len(history) > 60 {
		history = history[len(history)-60:]
	}
	a.aiSessions.setOAI(sessionID, history)

	return &aiChatResult{
		Reply:     roundText.String(),
		ToolCalls: toolCallLog,
		SessionID: sessionID,
	}, nil
}

// ---------- /ai/chat/stream handler ----------

func (a *App) handleAIChatStream(w http.ResponseWriter, r *http.Request) {
	if !aiEnabled() {
		http.Error(w, `{"error":"AI未启用"}`, http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		TaskID    string `json:"task_id"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(r, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		http.Error(w, "message required", http.StatusBadRequest)
		return
	}
	if body.SessionID == "" {
		body.SessionID = newID("ais")
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	var result *aiChatResult
	var err error

	switch aiProvider() {
	case providerOpenAI:
		if aiWireAPI() == "responses" {
			result, err = a.runOpenAIAgentStream(r.Context(), w, body.SessionID, body.Message, body.TaskID, body.Model)
		} else {
			// Chat Completions: run sync, then send as single delta
			result, err = a.runOpenAIChatAgent(r.Context(), body.SessionID, body.Message, body.TaskID, body.Model)
			if err == nil {
				for _, tc := range result.ToolCalls {
					writeSSE(w, "tool", jsonMap{"name": tc})
				}
				writeSSE(w, "delta", jsonMap{"text": result.Reply})
			}
		}
	default:
		// Anthropic: run sync, send as single delta
		result, err = a.runAnthropicAgent(r.Context(), body.SessionID, body.Message, body.TaskID, body.Model)
		if err == nil {
			for _, tc := range result.ToolCalls {
				writeSSE(w, "tool", jsonMap{"name": tc})
			}
			writeSSE(w, "delta", jsonMap{"text": result.Reply})
		}
	}

	if err != nil {
		writeSSE(w, "error", jsonMap{"error": err.Error()})
		return
	}
	writeSSE(w, "done", jsonMap{
		"session_id": result.SessionID,
		"tool_calls": result.ToolCalls,
	})
}


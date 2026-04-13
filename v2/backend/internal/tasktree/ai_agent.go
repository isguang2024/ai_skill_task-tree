package tasktree

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------- Provider detection ----------

const (
	aiMaxIter   = 12
	aiMaxTokens = 4096

	providerAnthropic = "anthropic"
	providerOpenAI    = "openai"
)

func aiProvider() string {
	cfg := loadAIConfig()
	if cfg.Provider != "" {
		return cfg.Provider
	}
	if cfg.OpenAIAPIKey != "" {
		return providerOpenAI
	}
	if cfg.AnthropicAPIKey != "" {
		return providerAnthropic
	}
	return ""
}

func aiEnabled() bool { return aiProvider() != "" }

func aiAPIKey() string {
	cfg := loadAIConfig()
	if aiProvider() == providerOpenAI {
		return cfg.OpenAIAPIKey
	}
	return cfg.AnthropicAPIKey
}

func aiBaseURL() string {
	return loadAIConfig().BaseURL
}

func aiModel(override ...string) string {
	if len(override) > 0 && strings.TrimSpace(override[0]) != "" {
		return strings.TrimSpace(override[0])
	}
	return loadAIConfig().Model
}

func aiReasoningEffort() string { return loadAIConfig().ReasoningEffort }
func aiStoreResponses() bool    { return !loadAIConfig().DisableResponseStorage }
func aiUserAgent() string {
	return loadAIConfig().UserAgent
}

// aiWireAPI: "responses" uses OpenAI Responses API; anything else uses Chat Completions.
func aiWireAPI() string {
	return loadAIConfig().WireAPI
}

// ---------- Anthropic Messages API types ----------

type anthropicRequest struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	System    string      `json:"system"`
	Tools     []aiTool    `json:"tools,omitempty"`
	Messages  []aiMessage `json:"messages"`
}

type anthropicResponse struct {
	ID         string               `json:"id"`
	Role       string               `json:"role"`
	Content    []anthropicRespBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Error      *apiError            `json:"error,omitempty"`
}

type anthropicRespBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ---------- OpenAI Responses API types ----------

type oaiReasoningConfig struct {
	Effort string `json:"effort,omitempty"`
}

type oaiRequest struct {
	Model           string              `json:"model"`
	Input           []map[string]any    `json:"input"`
	Instructions    string              `json:"instructions,omitempty"`
	Tools           []oaiTool           `json:"tools,omitempty"`
	Store           bool                `json:"store"`
	Stream          bool                `json:"stream,omitempty"`
	Reasoning       *oaiReasoningConfig `json:"reasoning,omitempty"`
	MaxOutputTokens int                 `json:"max_output_tokens,omitempty"`
}

type oaiTool struct {
	Type        string         `json:"type"` // "function"
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type oaiResponse struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Output []oaiOutItem `json:"output"`
	Error  *apiError    `json:"error,omitempty"`
}

type oaiOutItem struct {
	Type      string       `json:"type"` // "message" | "function_call"
	ID        string       `json:"id,omitempty"`
	Role      string       `json:"role,omitempty"`
	Content   []oaiContent `json:"content,omitempty"`
	CallID    string       `json:"call_id,omitempty"`
	Name      string       `json:"name,omitempty"`
	Arguments string       `json:"arguments,omitempty"`
}

type oaiContent struct {
	Type string `json:"type"` // "output_text"
	Text string `json:"text,omitempty"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ---------- OpenAI Chat Completions API types ----------

type chatRequest struct {
	Model     string           `json:"model"`
	Messages  []map[string]any `json:"messages"`
	Tools     []chatTool       `json:"tools,omitempty"`
	MaxTokens int              `json:"max_tokens,omitempty"`
}

type chatTool struct {
	Type     string       `json:"type"` // "function"
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message      chatMsg `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type chatMsg struct {
	Role      string         `json:"role"`
	Content   any            `json:"content"` // string or null
	ToolCalls []chatToolCall `json:"tool_calls,omitempty"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatToolCallFunc `json:"function"`
}

type chatToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ---------- HTTP helper ----------

func doPost(ctx context.Context, url string, headers map[string]string, body any) ([]byte, error) {
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
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ---------- Anthropic API call ----------

func callAnthropic(ctx context.Context, msgs []aiMessage, system string, tools []aiTool, model string) (*anthropicResponse, error) {
	url := aiBaseURL() + "/v1/messages"
	headers := map[string]string{
		"x-api-key":         aiAPIKey(),
		"anthropic-version": "2023-06-01",
	}
	req := anthropicRequest{
		Model:     aiModel(model),
		MaxTokens: aiMaxTokens,
		System:    system,
		Tools:     tools,
		Messages:  msgs,
	}
	raw, err := doPost(ctx, url, headers, req)
	if err != nil {
		return nil, err
	}
	var result anthropicResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := raw
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("Anthropic 响应解析失败: %s", preview)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("Anthropic API 错误: %s", result.Error.Message)
	}
	return &result, nil
}

// ---------- OpenAI Responses API call ----------

func callOpenAIResponses(ctx context.Context, input []map[string]any, instructions string, tools []oaiTool, model string) (*oaiResponse, error) {
	url := aiBaseURL() + "/v1/responses"
	headers := map[string]string{
		"authorization": "Bearer " + aiAPIKey(),
	}
	// Note: some proxies require store:true when input is an array.
	// We always set store:true for array input to ensure multi-turn works.
	req := oaiRequest{
		Model:           aiModel(model),
		Input:           input,
		Instructions:    instructions,
		Tools:           tools,
		Store:           true,
		MaxOutputTokens: aiMaxTokens,
	}
	if effort := aiReasoningEffort(); effort != "" {
		req.Reasoning = &oaiReasoningConfig{Effort: effort}
	}

	raw, err := doPost(ctx, url, headers, req)
	if err != nil {
		return nil, err
	}
	var result oaiResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := raw
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("OpenAI 响应解析失败: %s", preview)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("OpenAI API 错误: %s", result.Error.Message)
	}
	// Return the full raw response in error if output is empty and no error
	if len(result.Output) == 0 {
		return nil, fmt.Errorf("OpenAI 返回空 output (input_len=%d). raw_status=%s", len(input), extractField(raw, "status"))
	}
	return &result, nil
}

func extractField(raw []byte, field string) string {
	key := []byte(`"` + field + `":"`)
	idx := bytes.Index(raw, key)
	if idx < 0 {
		return "unknown"
	}
	start := idx + len(key)
	end := bytes.IndexByte(raw[start:], '"')
	if end < 0 {
		return "unknown"
	}
	return string(raw[start : start+end])
}

// ---------- OpenAI Chat Completions call ----------

func callOpenAIChatCompletions(ctx context.Context, msgs []map[string]any, tools []chatTool, model string) (*chatResponse, error) {
	url := aiBaseURL() + "/v1/chat/completions"
	headers := map[string]string{
		"authorization": "Bearer " + aiAPIKey(),
	}
	req := chatRequest{
		Model:     aiModel(model),
		Messages:  msgs,
		Tools:     tools,
		MaxTokens: aiMaxTokens,
	}
	raw, err := doPost(ctx, url, headers, req)
	if err != nil {
		return nil, err
	}
	var result chatResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := raw
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("ChatCompletions 响应解析失败: %s", preview)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("OpenAI API 错误 [%s]: %s", result.Error.Code, result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI 返回空 choices")
	}
	return &result, nil
}

func aiChatTools() []chatTool {
	src := aiToolDefinitions()
	out := make([]chatTool, len(src))
	for i, t := range src {
		out[i] = chatTool{
			Type: "function",
			Function: chatFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}
	return out
}

// ---------- OpenAI Chat Completions agent loop ----------

func (a *App) runOpenAIChatAgent(ctx context.Context, sessionID, userMsg, taskID, model string) (*aiChatResult, error) {
	history := a.aiSessions.getOAI(sessionID)
	if len(history) == 0 {
		history = []map[string]any{
			{"role": "system", "content": aiSystemPrompt(taskID)},
		}
	}
	history = append(history, map[string]any{"role": "user", "content": userMsg})

	tools := aiChatTools()

	var toolCallLog []string
	var finalReply string

	for i := 0; i < aiMaxIter; i++ {
		resp, err := callOpenAIChatCompletions(ctx, history, tools, model)
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// Build history entry for assistant turn
		assistantEntry := map[string]any{"role": "assistant"}
		if msg.Content != nil && msg.Content != "" {
			assistantEntry["content"] = msg.Content
			if s, ok := msg.Content.(string); ok && s != "" {
				finalReply = s
			}
		} else {
			assistantEntry["content"] = nil
		}
		if len(msg.ToolCalls) > 0 {
			tcs := make([]map[string]any, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				tcs[i] = map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			assistantEntry["tool_calls"] = tcs
		}
		history = append(history, assistantEntry)

		if choice.FinishReason != "tool_calls" {
			break
		}

		// Execute tools
		for _, tc := range msg.ToolCalls {
			toolCallLog = append(toolCallLog, tc.Function.Name+"("+summarizeRaw(tc.Function.Arguments)+")")
			result := a.executeAITool(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			history = append(history, map[string]any{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      result,
			})
		}
	}

	if len(history) > 60 {
		history = history[len(history)-60:]
	}
	a.aiSessions.setOAI(sessionID, history)
	return &aiChatResult{Reply: finalReply, ToolCalls: toolCallLog, SessionID: sessionID}, nil
}

// ---------- OpenAI tool definitions (Responses API format) ----------

func aiOpenAITools() []oaiTool {
	src := aiToolDefinitions()
	out := make([]oaiTool, len(src))
	for i, t := range src {
		out[i] = oaiTool{
			Type:        "function",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		}
	}
	return out
}

// ---------- Agent result ----------

type aiChatResult struct {
	Reply     string   `json:"reply"`
	ToolCalls []string `json:"tool_calls"`
	SessionID string   `json:"session_id"`
}

// ---------- Anthropic agent loop ----------

func (a *App) runAnthropicAgent(ctx context.Context, sessionID, userMsg, taskID, model string) (*aiChatResult, error) {
	msgs := a.aiSessions.get(sessionID)
	msgs = append(msgs, aiMessage{Role: "user", Content: userMsg})

	system := aiSystemPrompt(taskID)
	tools := aiToolDefinitions()

	var toolCallLog []string
	var finalReply string

	for i := 0; i < aiMaxIter; i++ {
		resp, err := callAnthropic(ctx, msgs, system, tools, model)
		if err != nil {
			return nil, err
		}

		// Build assistant content blocks
		assistantBlocks := make([]aiContentBlock, 0, len(resp.Content))
		for _, block := range resp.Content {
			cb := aiContentBlock{Type: block.Type}
			switch block.Type {
			case "text":
				cb.Text = block.Text
				finalReply = block.Text
			case "tool_use":
				cb.ID = block.ID
				cb.Name = block.Name
				var inp any
				_ = json.Unmarshal(block.Input, &inp)
				cb.Input = inp
			}
			assistantBlocks = append(assistantBlocks, cb)
		}
		msgs = append(msgs, aiMessage{Role: "assistant", Content: assistantBlocks})

		if resp.StopReason != "tool_use" {
			break
		}

		// Execute tools and gather results
		resultBlocks := make([]aiContentBlock, 0)
		for _, block := range resp.Content {
			if block.Type != "tool_use" {
				continue
			}
			toolCallLog = append(toolCallLog, block.Name+"("+summarizeInput(block.Input)+")")
			result := a.executeAITool(ctx, block.Name, block.Input)
			resultBlocks = append(resultBlocks, aiContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   result,
			})
		}
		msgs = append(msgs, aiMessage{Role: "user", Content: resultBlocks})
	}

	if len(msgs) > 40 {
		msgs = msgs[len(msgs)-40:]
	}
	a.aiSessions.set(sessionID, msgs)

	return &aiChatResult{Reply: finalReply, ToolCalls: toolCallLog, SessionID: sessionID}, nil
}

// ---------- OpenAI Responses agent loop ----------

func (a *App) runOpenAIAgent(ctx context.Context, sessionID, userMsg, taskID, model string) (*aiChatResult, error) {
	// Load OpenAI input history (flat array of input items)
	history := a.aiSessions.getOAI(sessionID)
	history = append(history, map[string]any{"role": "user", "content": userMsg})

	// Append session ID to instructions so each session produces a unique cache key
	// on proxy servers that cache based on (instructions + tools) only.
	sidSuffix := sessionID
	if len(sidSuffix) > 8 {
		sidSuffix = sidSuffix[:8]
	}
	instructions := aiSystemPrompt(taskID) + "\n[sid:" + sidSuffix + "]"
	tools := aiOpenAITools()

	var toolCallLog []string
	var finalReply string

	for i := 0; i < aiMaxIter; i++ {
		resp, err := callOpenAIResponses(ctx, history, instructions, tools, model)
		if err != nil {
			return nil, err
		}

		hasFuncCalls := false
		for _, item := range resp.Output {
			switch item.Type {
			case "message":
				for _, c := range item.Content {
					if c.Type == "output_text" {
						finalReply = c.Text
					}
				}
			case "function_call":
				hasFuncCalls = true
			}
		}

		// Add all output items to history
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

		// Execute function calls and add results
		for _, item := range resp.Output {
			if item.Type != "function_call" {
				continue
			}
			toolCallLog = append(toolCallLog, item.Name+"("+summarizeRaw(item.Arguments)+")")
			result := a.executeAITool(ctx, item.Name, json.RawMessage(item.Arguments))
			history = append(history, map[string]any{
				"type":    "function_call_output",
				"call_id": item.CallID,
				"output":  result,
			})
		}
	}

	// Limit history length
	if len(history) > 60 {
		history = history[len(history)-60:]
	}
	a.aiSessions.setOAI(sessionID, history)

	return &aiChatResult{Reply: finalReply, ToolCalls: toolCallLog, SessionID: sessionID}, nil
}

// ---------- Unified agent dispatch ----------

func (a *App) runAIAgent(ctx context.Context, sessionID, userMsg, taskID, model string) (*aiChatResult, error) {
	switch aiProvider() {
	case providerOpenAI:
		if aiWireAPI() == "responses" {
			return a.runOpenAIAgent(ctx, sessionID, userMsg, taskID, model)
		}
		return a.runOpenAIChatAgent(ctx, sessionID, userMsg, taskID, model)
	default:
		return a.runAnthropicAgent(ctx, sessionID, userMsg, taskID, model)
	}
}

// ---------- Helpers ----------

func summarizeInput(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return summarizeMap(m)
}

func summarizeRaw(s string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return s
	}
	return summarizeMap(m)
}

func summarizeMap(m map[string]any) string {
	parts := make([]string, 0, len(m))
	for k, v := range m {
		s := asString(v)
		if len(s) > 28 {
			s = s[:28] + "…"
		}
		parts = append(parts, k+"="+s)
	}
	return strings.Join(parts, ", ")
}

func aiSystemPrompt(taskID string) string {
	var sb strings.Builder
	sb.WriteString(`你是 Task Tree 任务管理 AI 助手。当前时间：` + time.Now().Format("2006-01-02 15:04") + `

## 最重要的规则：立即调用工具，不要先用文字描述你要做什么
- 用户让你查任务 → 直接调用 get_task，不要说"我先去获取..."
- 用户让你列任务 → 直接调用 list_tasks
- 用户让你操作节点 → 直接调用工具执行
- 拿到工具结果后，再用中文汇报给用户

## 节点操作规则
- 操作节点前必须先调用 get_task，从结果中读取真实 node_id（格式 nd_xxx）
- 路径（proj/1/2）≠ node_id，绝对不能混用
- 节点记录格式：node_id:nd_xxx  path:xxx  [status] 标题  进度%

## 其他规则
- 删除任务前必须用户明确说"确认删除"，再调用 delete_task(confirm="yes")
- 创建任务：create_task → 逐个 create_node（根节点→子节点）
- 风格：简洁中文，结果导向

## 工具
list_tasks / get_task / create_task / create_node / update_node / claim_node / progress_node / complete_node / transition_node(block|unblock|pause|reopen|cancel) / delete_task`)

	if taskID != "" {
		fmt.Fprintf(&sb, "\n\n当前任务 ID：%s（直接对此任务操作，无需询问）", taskID)
	}
	return sb.String()
}

// ---------- HTTP handlers ----------

func (a *App) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if !aiEnabled() {
		writeJSON(w, http.StatusServiceUnavailable, jsonMap{
			"error": "AI 未启用。Anthropic: 设置 ANTHROPIC_API_KEY；OpenAI: 设置 OPENAI_API_KEY 和 AI_BASE_URL。",
		})
		return
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, jsonMap{"error": "method not allowed"})
		return
	}

	var body struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		TaskID    string `json:"task_id"`
		Model     string `json:"model"` // optional override, e.g. "gpt-5.2"
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		writeJSON(w, http.StatusBadRequest, jsonMap{"error": "message 不能为空"})
		return
	}
	if body.SessionID == "" {
		body.SessionID = newID("ais")
	}

	result, err := a.runAIAgent(r.Context(), body.SessionID, body.Message, body.TaskID, body.Model)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, jsonMap{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleAIClearSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, jsonMap{"error": "method not allowed"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeErr(w, err)
			return
		}
	}
	a.aiSessions.clear(body.SessionID)
	writeJSON(w, http.StatusOK, jsonMap{"ok": true})
}

// handleAIStatus returns current AI configuration (no secrets).
func (a *App) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	provider := aiProvider()
	if provider == "" {
		writeJSON(w, http.StatusOK, jsonMap{"enabled": false})
		return
	}
	info := jsonMap{
		"enabled":  true,
		"provider": provider,
		"model":    aiModel(),
		"base_url": aiBaseURL(),
	}
	if provider == providerOpenAI {
		info["wire_api"] = aiWireAPI()
	}
	writeJSON(w, http.StatusOK, info)
}


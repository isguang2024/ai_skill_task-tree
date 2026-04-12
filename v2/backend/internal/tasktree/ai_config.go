package tasktree

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

type aiConfig struct {
	Provider               string
	OpenAIAPIKey           string
	AnthropicAPIKey        string
	BaseURL                string
	Model                  string
	WireAPI                string
	ReasoningEffort        string
	DisableResponseStorage bool
	UserAgent              string
}

var (
	aiConfigOnce  sync.Once
	aiConfigCache aiConfig
)

func loadAIConfig() aiConfig {
	aiConfigOnce.Do(func() {
		aiConfigCache = readAIConfigFile()
	})
	return aiConfigCache
}

func readAIConfigFile() aiConfig {
	cfg := aiConfig{
		BaseURL:   "https://api.anthropic.com",
		Model:     "claude-opus-4-5",
		WireAPI:   "chat_completions",
		UserAgent: "Codex Desktop/0.119.0-alpha.11 (Windows 10.0.19042; x86_64) unknown (Codex Desktop; 26.406.31014)",
	}
	file, err := os.Open(resolveUpwardPath("ai.env.txt"))
	if err != nil {
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "AI_PROVIDER":
			cfg.Provider = strings.ToLower(val)
		case "OPENAI_API_KEY":
			cfg.OpenAIAPIKey = val
		case "ANTHROPIC_API_KEY":
			cfg.AnthropicAPIKey = val
		case "AI_BASE_URL":
			if val != "" {
				cfg.BaseURL = strings.TrimRight(val, "/")
			}
		case "AI_MODEL":
			if val != "" {
				cfg.Model = val
			}
		case "AI_WIRE_API":
			if val != "" {
				cfg.WireAPI = strings.ToLower(val)
			}
		case "AI_REASONING_EFFORT":
			cfg.ReasoningEffort = val
		case "AI_DISABLE_RESPONSE_STORAGE":
			cfg.DisableResponseStorage = strings.EqualFold(val, "true")
		case "AI_USER_AGENT":
			if val != "" {
				cfg.UserAgent = val
			}
		}
	}
	return cfg
}


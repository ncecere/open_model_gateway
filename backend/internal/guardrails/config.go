package guardrails

import "encoding/json"

// Config represents the structured guardrail policy used by the evaluator.
type Config struct {
	Enabled    bool             `json:"enabled"`
	Prompt     PromptConfig     `json:"prompt"`
	Response   ResponseConfig   `json:"response"`
	Moderation ModerationConfig `json:"moderation"`
}

type PromptConfig struct {
	BlockedKeywords []string `json:"blocked_keywords"`
}

type ResponseConfig struct {
	BlockedKeywords []string `json:"blocked_keywords"`
}

type ModerationConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Action   string `json:"action"`
	WebhookURL string `json:"webhook_url"`
	WebhookAuthHeader string `json:"webhook_auth_header"`
	WebhookAuthValue string `json:"webhook_auth_value"`
	TimeoutSeconds int `json:"timeout_seconds"`
}

func DefaultConfig() Config {
	return Config{}
}

func ParseConfig(raw map[string]any) Config {
	if raw == nil {
		return DefaultConfig()
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return DefaultConfig()
	}
	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

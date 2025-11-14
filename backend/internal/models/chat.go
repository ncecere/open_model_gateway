package models

import "time"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float32      `json:"temperature,omitempty"`
	TopP        *float32      `json:"top_p,omitempty"`
	MaxTokens   *int32        `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Created time.Time    `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type ChatChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Created time.Time    `json:"created"`
	Choices []ChunkDelta `json:"choices"`
	Usage   *Usage       `json:"-"`
}

func (c ChatChunk) IsUsageOnly() bool {
	return len(c.Choices) == 0 && c.Usage != nil && (c.Usage.PromptTokens > 0 || c.Usage.CompletionTokens > 0 || c.Usage.TotalTokens > 0)
}

type ChunkDelta struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

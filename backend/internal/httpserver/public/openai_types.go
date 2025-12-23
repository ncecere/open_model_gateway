package public

import "encoding/json"

type openAIChatMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Name      string          `json:"name,omitempty"`
	Reasoning string          `json:"reasoning,omitempty"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	ReasoningTokens  int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens"`
}

type openAIStreamDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Reasoning string          `json:"reasoning,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
}

package batchworker

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

type batchItem struct {
	ID       uuid.UUID
	Index    int64
	CustomID string
	Input    []byte
}

type batchRequest struct {
	CustomID string          `json:"custom_id"`
	Method   string          `json:"method"`
	URL      string          `json:"url"`
	Body     json.RawMessage `json:"body"`
	Headers  json.RawMessage `json:"headers"`
}

type chatBody struct {
	Model       string          `json:"model"`
	Messages    []chatMessage   `json:"messages"`
	Temperature *float32        `json:"temperature"`
	TopP        *float32        `json:"top_p"`
	MaxTokens   *int32          `json:"max_tokens"`
	Stream      bool            `json:"stream"`
	Stop        json.RawMessage `json:"stop"`
}

type responsesBody struct {
	Model              string            `json:"model"`
	Input              json.RawMessage   `json:"input"`
	Instructions       string            `json:"instructions"`
	Temperature        *float32          `json:"temperature,omitempty"`
	TopP               *float32          `json:"top_p,omitempty"`
	MaxOutputTokens    *int32            `json:"max_output_tokens,omitempty"`
	Stream             bool              `json:"stream,omitempty"`
	Metadata           map[string]string `json:"metadata"`
	Tools              json.RawMessage   `json:"tools"`
	ToolChoice         json.RawMessage   `json:"tool_choice"`
	ResponseFormat     json.RawMessage   `json:"response_format"`
	Conversation       json.RawMessage   `json:"conversation"`
	PreviousResponseID string            `json:"previous_response_id"`
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`
}

type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Name    string          `json:"name,omitempty"`
}

type openAIEmbeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n,omitempty"`
	User           string `json:"user,omitempty"`
	Background     string `json:"background,omitempty"`
	Style          string `json:"style,omitempty"`
}

type openAIImageEditBatchRequest struct {
	Model          string          `json:"model"`
	Prompt         string          `json:"prompt"`
	Size           string          `json:"size,omitempty"`
	ResponseFormat string          `json:"response_format,omitempty"`
	Quality        string          `json:"quality,omitempty"`
	Background     string          `json:"background,omitempty"`
	Style          string          `json:"style,omitempty"`
	N              int             `json:"n,omitempty"`
	User           string          `json:"user,omitempty"`
	Image          json.RawMessage `json:"image"`
	Images         json.RawMessage `json:"image[]"`
	Mask           json.RawMessage `json:"mask"`
	Masks          json.RawMessage `json:"mask[]"`
}

type openAIImageVariationBatchRequest struct {
	Model          string          `json:"model"`
	Size           string          `json:"size,omitempty"`
	ResponseFormat string          `json:"response_format,omitempty"`
	Quality        string          `json:"quality,omitempty"`
	Background     string          `json:"background,omitempty"`
	Style          string          `json:"style,omitempty"`
	N              int             `json:"n,omitempty"`
	User           string          `json:"user,omitempty"`
	Image          json.RawMessage `json:"image"`
	Images         json.RawMessage `json:"image[]"`
}

type itemOutcome struct {
	response   []byte
	errPayload []byte
	statusCode int
	requestID  string
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
}

type openAIChatChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIChatMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Reasoning string          `json:"reasoning,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	ReasoningTokens  int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens"`
}

type openAIEmbedding struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
	Object    string    `json:"object"`
}

type openAIEmbeddingResponse struct {
	Object string            `json:"object"`
	Model  string            `json:"model"`
	Data   []openAIEmbedding `json:"data"`
	Usage  openAIUsage       `json:"usage"`
}

type openAIModerationRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type openAIModerationResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Results []openAIModerationResult `json:"results"`
}

type openAIModerationResult struct {
	Categories                models.ModerationCategories                `json:"categories"`
	CategoryAppliedInputTypes models.ModerationCategoryAppliedInputTypes `json:"category_applied_input_types"`
	CategoryScores            models.ModerationCategoryScores            `json:"category_scores"`
	Flagged                   bool                                       `json:"flagged"`
}

type openAIImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
}

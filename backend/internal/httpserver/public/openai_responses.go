package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type openAIResponsesRequest struct {
	Model             string            `json:"model"`
	Input             json.RawMessage   `json:"input"`
	Instructions      string            `json:"instructions"`
	Temperature       *float32          `json:"temperature,omitempty"`
	TopP              *float32          `json:"top_p,omitempty"`
	MaxOutputTokens   *int32            `json:"max_output_tokens,omitempty"`
	Metadata          map[string]string `json:"metadata"`
	Stream            bool              `json:"stream,omitempty"`
	Tools             json.RawMessage   `json:"tools"`
	ToolChoice        json.RawMessage   `json:"tool_choice"`
	ResponseFormat    json.RawMessage   `json:"response_format"`
	Conversation      json.RawMessage   `json:"conversation"`
	PreviousResponse  string            `json:"previous_response_id"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
}

type openAIResponsesPayload struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	CreatedAt         int64                   `json:"created_at"`
	Status            string                  `json:"status"`
	Model             string                  `json:"model"`
	Output            []openAIResponsesOutput `json:"output"`
	ParallelToolCalls bool                    `json:"parallel_tool_calls"`
	ToolChoice        string                  `json:"tool_choice"`
	Tools             []interface{}           `json:"tools"`
	Usage             openAIResponsesUsage    `json:"usage"`
	Instructions      string                  `json:"instructions,omitempty"`
	Metadata          map[string]string       `json:"metadata,omitempty"`
}

type openAIResponsesOutput struct {
	Type    string                         `json:"type"`
	ID      string                         `json:"id"`
	Status  string                         `json:"status"`
	Role    string                         `json:"role"`
	Content []openAIResponsesOutputContent `json:"content"`
}

type openAIResponsesOutputContent struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Annotations []interface{} `json:"annotations"`
}

type openAIResponsesUsage struct {
	InputTokens         int32                             `json:"input_tokens"`
	OutputTokens        int32                             `json:"output_tokens"`
	TotalTokens         int32                             `json:"total_tokens"`
	OutputTokensDetails openAIResponsesUsageOutputDetails `json:"output_tokens_details"`
}

type openAIResponsesUsageOutputDetails struct {
	ReasoningTokens int32 `json:"reasoning_tokens"`
}

type openAIResponseOptions struct {
	Instructions      string
	Metadata          map[string]string
	ParallelToolCalls bool
}

func (h *openAIHandler) responses(c *fiber.Ctx) error {
	var req openAIResponsesRequest
	if err := c.BodyParser(&req); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, "invalid request body")
	}
	alias := strings.TrimSpace(req.Model)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if len(req.Input) == 0 {
		return httputil.WriteError(c, fiber.StatusBadRequest, "input is required")
	}
	// Conversations and response_format are not yet supported
	if len(req.ResponseFormat) > 0 || len(req.Conversation) > 0 || req.PreviousResponse != "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "conversations and response_format are not supported")
	}
	if err := validateResponseMetadata(req.Metadata); err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	messages, err := buildResponseMessages(req.Instructions, req.Input)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusBadRequest, err.Error())
	}

	// Parse tools
	var tools []models.Tool
	if len(req.Tools) > 0 {
		var parsedTools []openAITool
		if err := json.Unmarshal(req.Tools, &parsedTools); err != nil {
			return httputil.WriteError(c, fiber.StatusBadRequest, "invalid tools format")
		}
		tools = make([]models.Tool, 0, len(parsedTools))
		for _, t := range parsedTools {
			tools = append(tools, models.Tool{
				Type: t.Type,
				Function: models.ToolFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
					Strict:      t.Function.Strict,
				},
			})
		}
	}

	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	if !h.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	// Validate that the model supports tools if tools are provided
	if len(tools) > 0 {
		model, err := h.container.Queries.GetModelByAlias(ctx, alias)
		if err == nil && !model.SupportsTools {
			return httputil.WriteError(c, fiber.StatusBadRequest, "model does not support tool calling")
		}
	}

	traceID := traceIDFromContext(c)
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))

	modelReq := models.ChatRequest{
		Messages:          messages,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		MaxTokens:         req.MaxOutputTokens,
		Tools:             tools,
		ToolChoice:        req.ToolChoice,
		ParallelToolCalls: req.ParallelToolCalls,
	}

	parallel := true
	if req.ParallelToolCalls != nil {
		parallel = *req.ParallelToolCalls
	}
	options := openAIResponseOptions{
		Instructions:      strings.TrimSpace(req.Instructions),
		Metadata:          req.Metadata,
		ParallelToolCalls: parallel,
	}

	if req.Stream {
		return h.chatStreamPipeline.StreamResponses(c, rc, alias, traceID, idempotencyKey, modelReq, options)
	}

	return h.chatPipeline.ExecuteWithConverter(c, rc, alias, traceID, idempotencyKey, modelReq, func(resp models.ChatResponse, alias string) (interface{}, error) {
		return convertResponsesResponse(resp, alias, options), nil
	})
}

func validateResponseMetadata(md map[string]string) error {
	if len(md) == 0 {
		return nil
	}
	if len(md) > 16 {
		return fmt.Errorf("metadata supports at most 16 key/value pairs")
	}
	for key, value := range md {
		if len(key) > 64 {
			return fmt.Errorf("metadata key %q exceeds 64 characters", key)
		}
		if len(value) > 512 {
			return fmt.Errorf("metadata value for %q exceeds 512 characters", key)
		}
	}
	return nil
}

func buildResponseMessages(instructions string, input json.RawMessage) ([]models.ChatMessage, error) {
	messages := make([]models.ChatMessage, 0, 1)
	if instr := strings.TrimSpace(instructions); instr != "" {
		messages = append(messages, models.ChatMessage{Role: "system", Content: instr})
	}
	inputMessages, err := parseResponseInputItems(input)
	if err != nil {
		return nil, err
	}
	messages = append(messages, inputMessages...)
	return messages, nil
}

func parseResponseInputItems(raw json.RawMessage) ([]models.ChatMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("input is required")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []models.ChatMessage{{Role: "user", Content: text}}, nil
	}
	var items []openAIChatMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		if len(items) == 0 {
			return nil, errors.New("input array must contain at least one item")
		}
		return convertOpenAIMessages(items, "input item")
	}
	return nil, errors.New("input must be a string or array of message objects")
}

func convertResponsesResponse(resp models.ChatResponse, alias string, opts openAIResponseOptions) openAIResponsesPayload {
	return buildResponsesPayload(resp, alias, opts, "")
}

func buildResponsesPayload(resp models.ChatResponse, alias string, opts openAIResponseOptions, statusOverride string) openAIResponsesPayload {
	outputs := make([]openAIResponsesOutput, 0, len(resp.Choices))
	overallStatus := statusOverride
	if overallStatus == "" {
		overallStatus = "completed"
	}
	for _, choice := range resp.Choices {
		status := mapResponseStatus(choice.FinishReason)
		if status != "completed" && statusOverride == "" {
			overallStatus = status
		}
		outputs = append(outputs, openAIResponsesOutput{
			Type:   "message",
			ID:     fmt.Sprintf("%s-%d", resp.ID, choice.Index),
			Status: status,
			Role:   choice.Message.Role,
			Content: []openAIResponsesOutputContent{
				buildResponseOutputContent(choice.Message),
			},
		})
	}
	usage := openAIResponsesUsage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		OutputTokensDetails: openAIResponsesUsageOutputDetails{
			ReasoningTokens: resp.Usage.ReasoningTokens,
		},
	}
	payload := openAIResponsesPayload{
		ID:                resp.ID,
		Object:            "response",
		CreatedAt:         resp.Created.Unix(),
		Status:            overallStatus,
		Model:             alias,
		Output:            outputs,
		ParallelToolCalls: opts.ParallelToolCalls,
		ToolChoice:        "auto",
		Tools:             []interface{}{},
		Usage:             usage,
	}
	if instr := strings.TrimSpace(opts.Instructions); instr != "" {
		payload.Instructions = instr
	}
	if len(opts.Metadata) > 0 {
		payload.Metadata = opts.Metadata
	}
	return payload
}

func mapResponseStatus(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "length", "content_filter":
		return "incomplete"
	case "":
		return "completed"
	default:
		return "completed"
	}
}

func buildResponseOutputContent(msg models.ChatMessage) openAIResponsesOutputContent {
	return openAIResponsesOutputContent{
		Type:        "output_text",
		Text:        msg.Text(),
		Annotations: []interface{}{},
	}
}

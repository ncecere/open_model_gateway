package batchworker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"strings"
)

func (w *Worker) runResponsesItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/responses")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body responsesBody
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid responses body: %v", err)),
		}
	}
	alias := strings.TrimSpace(body.Model)
	if alias == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "model is required"),
		}
	}
	if body.Stream {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "streaming is not supported in batches"),
		}
	}
	if len(body.Input) == 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "input is required"),
		}
	}
	if len(body.Conversation) > 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "conversations are not supported in batches"),
		}
	}
	if strings.TrimSpace(body.PreviousResponseID) != "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "previous_response_id is not supported in batches"),
		}
	}
	if err := validateResponsesMetadata(body.Metadata); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", err.Error()),
		}
	}

	messages, err := buildResponsesMessages(body.Instructions, body.Input)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", err.Error()),
		}
	}

	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return itemOutcome{
			statusCode: fiber.StatusForbidden,
			requestID:  traceID,
			errPayload: encodeErrorPayload("permission_error", "model not enabled for tenant"),
		}
	}

	// Parse tools for batch
	var tools []models.Tool
	if len(body.Tools) > 0 {
		var rawTools []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
				Strict      *bool           `json:"strict"`
			} `json:"function"`
		}
		if err := json.Unmarshal(body.Tools, &rawTools); err == nil {
			tools = make([]models.Tool, 0, len(rawTools))
			for _, t := range rawTools {
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
	}

	req := models.ChatRequest{
		Messages:          messages,
		Temperature:       body.Temperature,
		TopP:              body.TopP,
		MaxTokens:         body.MaxOutputTokens,
		Tools:             tools,
		ToolChoice:        body.ToolChoice,
		ParallelToolCalls: body.ParallelToolCalls,
	}

	options := responsesOptions{
		Instructions:      strings.TrimSpace(body.Instructions),
		Metadata:          body.Metadata,
		ParallelToolCalls: true,
	}
	if body.ParallelToolCalls != nil {
		options.ParallelToolCalls = *body.ParallelToolCalls
	}

	callCtx := requestctx.WithContext(ctx, rc)
	result, err := w.executor.Chat(callCtx, rc, alias, req, traceID, "")
	if err != nil {
		return mapExecutorError(traceID, err)
	}

	payload := convertResponsesPayload(result.Response, alias, options)
	data, err := json.Marshal(payload)
	if err != nil {
		return newSerializationOutcome(traceID, err)
	}
	requestID := payload.ID
	if requestID == "" {
		requestID = traceID
	}
	return itemOutcome{statusCode: fiber.StatusOK, requestID: requestID, response: data}
}

func convertResponsesPayload(resp models.ChatResponse, alias string, opts responsesOptions) openAIResponsesPayload {
	outputs := make([]openAIResponsesOutput, 0, len(resp.Choices))
	status := "completed"
	outputIdx := 0
	for _, choice := range resp.Choices {
		finish := mapResponseStatus(choice.FinishReason)
		if finish != "completed" {
			status = finish
		}
		// Emit function_call items for tool calls
		for _, tc := range choice.Message.ToolCalls {
			outputs = append(outputs, openAIResponsesOutput{
				Type:      "function_call",
				ID:        fmt.Sprintf("%s-fc-%d", resp.ID, outputIdx),
				Status:    "completed",
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
			outputIdx++
		}
		// Emit message item if there's text content
		text := choice.Message.Text()
		if text != "" || len(choice.Message.ToolCalls) == 0 {
			outputs = append(outputs, openAIResponsesOutput{
				Type:   "message",
				ID:     fmt.Sprintf("%s-%d", resp.ID, outputIdx),
				Status: finish,
				Role:   choice.Message.Role,
				Content: []openAIResponsesOutputContent{
					{
						Type:        "output_text",
						Text:        text,
						Annotations: []interface{}{},
					},
				},
			})
			outputIdx++
		}
	}
	completedAt := resp.Created.Unix()
	usage := openAIResponsesUsage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		OutputTokensDetails: openAIResponsesUsageOutputDetails{
			ReasoningTokens: resp.Usage.ReasoningTokens,
		},
	}

	var instructions *string
	if inst := strings.TrimSpace(opts.Instructions); inst != "" {
		instructions = &inst
	}
	metadata := opts.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	var incomplete *incompleteDetails
	if status == "incomplete" {
		incomplete = &incompleteDetails{Reason: "max_output_tokens"}
	}

	payload := openAIResponsesPayload{
		ID:                resp.ID,
		Object:            "response",
		CreatedAt:         resp.Created.Unix(),
		CompletedAt:       &completedAt,
		Status:            status,
		IncompleteDetails: incomplete,
		Model:             alias,
		Instructions:      instructions,
		Output:            outputs,
		ParallelToolCalls: opts.ParallelToolCalls,
		ToolChoice:        "auto",
		Truncation:        "disabled",
		Tools:             []interface{}{},
		Usage:             usage,
		Metadata:          metadata,
		ServiceTier:       "default",
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

type responsesOptions struct {
	Instructions      string
	Metadata          map[string]string
	ParallelToolCalls bool
}

type openAIResponsesPayload struct {
	ID                 string                  `json:"id"`
	Object             string                  `json:"object"`
	CreatedAt          int64                   `json:"created_at"`
	CompletedAt        *int64                  `json:"completed_at"`
	Status             string                  `json:"status"`
	IncompleteDetails  *incompleteDetails      `json:"incomplete_details"`
	Model              string                  `json:"model"`
	PreviousResponseID *string                 `json:"previous_response_id"`
	Instructions       *string                 `json:"instructions"`
	Output             []openAIResponsesOutput `json:"output"`
	Error              *responsesErrorObj      `json:"error"`
	ParallelToolCalls  bool                    `json:"parallel_tool_calls"`
	ToolChoice         string                  `json:"tool_choice"`
	Truncation         string                  `json:"truncation"`
	Tools              []interface{}           `json:"tools"`
	Temperature        *float32                `json:"temperature"`
	TopP               *float32                `json:"top_p"`
	MaxOutputTokens    *int32                  `json:"max_output_tokens"`
	Usage              openAIResponsesUsage    `json:"usage"`
	Metadata           map[string]string       `json:"metadata"`
	Store              bool                    `json:"store"`
	ServiceTier        string                  `json:"service_tier"`
}

type incompleteDetails struct {
	Reason string `json:"reason"`
}

type responsesErrorObj struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type openAIResponsesOutput struct {
	Type    string                         `json:"type"`
	ID      string                         `json:"id"`
	Status  string                         `json:"status"`
	Role    string                         `json:"role,omitempty"`
	Content []openAIResponsesOutputContent `json:"content,omitempty"`

	// function_call fields
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
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
	InputTokensDetails  openAIResponsesUsageInputDetails  `json:"input_tokens_details"`
	OutputTokensDetails openAIResponsesUsageOutputDetails `json:"output_tokens_details"`
}

type openAIResponsesUsageInputDetails struct {
	CachedTokens int32 `json:"cached_tokens"`
}

type openAIResponsesUsageOutputDetails struct {
	ReasoningTokens int32 `json:"reasoning_tokens"`
}

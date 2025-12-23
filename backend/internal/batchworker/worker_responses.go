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
	if len(body.Tools) > 0 || len(body.ToolChoice) > 0 || len(body.ResponseFormat) > 0 || len(body.Conversation) > 0 || strings.TrimSpace(body.PreviousResponseID) != "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "tools, conversations, and response_format are not supported in batches"),
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

	req := models.ChatRequest{
		Messages:    messages,
		Temperature: body.Temperature,
		TopP:        body.TopP,
		MaxTokens:   body.MaxOutputTokens,
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
	for _, choice := range resp.Choices {
		finish := mapResponseStatus(choice.FinishReason)
		if finish != "completed" {
			status = finish
		}
		outputs = append(outputs, openAIResponsesOutput{
			Type:   "message",
			ID:     fmt.Sprintf("%s-%d", resp.ID, choice.Index),
			Status: finish,
			Role:   choice.Message.Role,
			Content: []openAIResponsesOutputContent{
				{
					Type: "output_text",
					Text: choice.Message.Text(),
				},
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
		Status:            status,
		Model:             alias,
		Output:            outputs,
		ParallelToolCalls: opts.ParallelToolCalls,
		ToolChoice:        "auto",
		Tools:             []interface{}{},
		Usage:             usage,
	}
	if strings.TrimSpace(opts.Instructions) != "" {
		payload.Instructions = opts.Instructions
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

type responsesOptions struct {
	Instructions      string
	Metadata          map[string]string
	ParallelToolCalls bool
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

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

func (w *Worker) runChatItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/chat/completions")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body chatBody
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid chat body: %v", err)),
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
	if !w.container.IsModelAllowed(rc.TenantID, alias) {
		return itemOutcome{
			statusCode: fiber.StatusForbidden,
			requestID:  traceID,
			errPayload: encodeErrorPayload("permission_error", "model not enabled for tenant"),
		}
	}
	if body.Stream {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "streaming is not supported in batches"),
		}
	}

	stop, err := parseStop(body.Stop)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "invalid stop value"),
		}
	}

	messages := make([]models.ChatMessage, 0, len(body.Messages))
	for idx, msg := range body.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "user"
		}
		textContent, parts, err := models.ParseMessageContent(msg.Content)
		if err != nil {
			return itemOutcome{
				statusCode: fiber.StatusBadRequest,
				requestID:  traceID,
				errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid content for message %d: %v", idx, err)),
			}
		}
		messages = append(messages, models.ChatMessage{
			Role:         role,
			Content:      textContent,
			ContentParts: parts,
			Name:         msg.Name,
		})
	}
	if len(messages) == 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "messages are required"),
		}
	}

	req := models.ChatRequest{
		Messages:    messages,
		Temperature: body.Temperature,
		TopP:        body.TopP,
		MaxTokens:   body.MaxTokens,
		Stop:        stop,
	}

	return w.executeChatItem(ctx, rc, traceID, alias, req)
}

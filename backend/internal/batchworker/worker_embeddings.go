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

func (w *Worker) runEmbeddingItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/embeddings")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body openAIEmbeddingRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid embeddings body: %v", err)),
		}
	}
	body.Model = strings.TrimSpace(body.Model)
	if body.Model == "" {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "model is required"),
		}
	}
	values, err := parseEmbeddingInput(body.Input)
	if err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "invalid input field"),
		}
	}
	if len(values) == 0 {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", "input is required"),
		}
	}

	if !w.container.IsModelAllowed(rc.TenantID, body.Model) {
		return itemOutcome{
			statusCode: fiber.StatusForbidden,
			requestID:  traceID,
			errPayload: encodeErrorPayload("permission_error", "model not enabled for tenant"),
		}
	}

	req := models.EmbeddingsRequest{Model: body.Model, Input: values}
	return w.executeEmbeddingItem(ctx, rc, traceID, body.Model, req)
}

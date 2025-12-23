package batchworker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"strings"
)

func (w *Worker) runModerationItem(ctx context.Context, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	input, errPayload := decodeBatchRequest(item, "/v1/moderations")
	if errPayload != nil {
		return itemOutcome{statusCode: fiber.StatusBadRequest, requestID: traceID, errPayload: errPayload}
	}

	var body openAIModerationRequest
	if err := json.Unmarshal(input.Body, &body); err != nil {
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: encodeErrorPayload("invalid_request_error", fmt.Sprintf("invalid moderation body: %v", err)),
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
	values, err := parseEmbeddingInput(body.Input)
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

	return w.executeModerationItem(ctx, rc, traceID, alias, values)
}

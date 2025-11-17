package batchworker

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

func (w *Worker) executeChatItem(ctx context.Context, rc *requestctx.Context, traceID string, alias string, req models.ChatRequest) itemOutcome {
    callCtx := requestctx.WithContext(ctx, rc)
    result, err := w.executor.Chat(callCtx, rc, alias, req, traceID, "")
    if err != nil {
        return mapExecutorError(traceID, err)
    }
    response := convertChatResponse(result.Response, alias)
    data, err := json.Marshal(response)
    if err != nil {
        return newSerializationOutcome(traceID, err)
    }
    requestID := response.ID
    if requestID == "" {
        requestID = traceID
    }
    return itemOutcome{statusCode: fiber.StatusOK, requestID: requestID, response: data}
}

func (w *Worker) executeEmbeddingItem(ctx context.Context, rc *requestctx.Context, traceID string, alias string, req models.EmbeddingsRequest) itemOutcome {
    callCtx := requestctx.WithContext(ctx, rc)
    result, err := w.executor.Embed(callCtx, rc, alias, req, traceID)
    if err != nil {
        return mapExecutorError(traceID, err)
    }
    response := convertEmbeddingResponse(result.Response, alias)
    data, err := json.Marshal(response)
    if err != nil {
        return newSerializationOutcome(traceID, err)
    }
    return itemOutcome{statusCode: fiber.StatusOK, requestID: traceID, response: data}
}

func (w *Worker) executeModerationItem(ctx context.Context, rc *requestctx.Context, traceID string, alias string, inputs []string) itemOutcome {
    callCtx := requestctx.WithContext(ctx, rc)
    result, err := w.executor.Moderate(callCtx, rc, alias, inputs, traceID)
    if err != nil {
        return mapExecutorError(traceID, err)
    }
	response := convertModerationResponse(result.Response, alias)
    data, err := json.Marshal(response)
    if err != nil {
        return newSerializationOutcome(traceID, err)
    }
    return itemOutcome{statusCode: fiber.StatusOK, requestID: traceID, response: data}
}

func (w *Worker) executeImageOperation(ctx context.Context, rc *requestctx.Context, traceID, alias string, cfg executor.ImageOperationConfig) itemOutcome {
    callCtx := requestctx.WithContext(ctx, rc)
    cfg.Alias = alias
    result, err := w.executor.Image(callCtx, rc, traceID, cfg)
    if err != nil {
        return mapExecutorError(traceID, err)
    }
    payload := convertImageResponse(result.Response)
    data, err := json.Marshal(payload)
    if err != nil {
        return newSerializationOutcome(traceID, err)
    }
    return itemOutcome{statusCode: fiber.StatusOK, requestID: traceID, response: data}
}

func mapExecutorError(traceID string, err error) itemOutcome {
	if status, msg, ok := executor.AsAPIError(err); ok {
		return itemOutcome{
			statusCode: status,
			requestID:  traceID,
			errPayload: encodeErrorPayload(mapStatusToCode(status), msg),
		}
	}
	return itemOutcome{
		statusCode: fiber.StatusBadGateway,
		requestID:  traceID,
		errPayload: encodeErrorPayload("provider_error", err.Error()),
	}
}

func newSerializationOutcome(traceID string, err error) itemOutcome {
	return itemOutcome{
		statusCode: fiber.StatusInternalServerError,
		requestID:  traceID,
		errPayload: encodeErrorPayload("serialization_error", err.Error()),
	}
}

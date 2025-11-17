package public

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// chatPipeline encapsulates the shared flow for synchronous chat completions.
type chatPipeline struct {
	container *app.Container
	executor  *executor.Executor
}

func newChatPipeline(container *app.Container, exec *executor.Executor) *chatPipeline {
	return &chatPipeline{
		container: container,
		executor:  exec,
	}
}

func (p *chatPipeline) Execute(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
) error {
	ctx := c.UserContext()
	if idempotencyKey != "" {
		if data, ok := p.container.Idempotency.Get(ctx, idempotencyKey); ok {
			c.Set("Content-Type", "application/json")
			return c.Send(data)
		}
	}

	result, err := p.executor.Chat(ctx, rc, alias, req, traceID, idempotencyKey)
	if err != nil {
		if status, msg, ok := executor.AsAPIError(err); ok {
			return httputil.WriteError(c, status, msg)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	httputil.ApplyBudgetHeaders(c, result.BudgetStatus)

	resp := convertChatResponse(result.Response, alias)
	payload, err := json.Marshal(resp)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
	}

	if idempotencyKey != "" {
		p.container.Idempotency.Set(ctx, idempotencyKey, payload)
	}

	c.Set("Content-Type", "application/json")
	return c.Send(payload)
}

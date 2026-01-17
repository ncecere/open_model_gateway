package public

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/pipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// chatPipeline encapsulates the shared flow for synchronous chat completions.
type chatPipeline struct {
	*pipeline.Base
}

func newChatPipeline(container *app.Container, exec *executor.Executor) *chatPipeline {
	return &chatPipeline{Base: pipeline.NewBase(container, exec)}
}

func (p *chatPipeline) Execute(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
) error {
	return p.ExecuteWithConverter(c, rc, alias, traceID, idempotencyKey, req, func(resp models.ChatResponse, alias string) (interface{}, error) {
		return convertChatResponse(resp, alias), nil
	})
}

func (p *chatPipeline) ExecuteWithConverter(
	c *fiber.Ctx,
	rc *requestctx.Context,
	alias string,
	traceID string,
	idempotencyKey string,
	req models.ChatRequest,
	convert func(models.ChatResponse, string) (interface{}, error),
) error {
	ctx := c.UserContext()

	// Check idempotency cache
	if found, err := p.CheckIdempotency(c, ctx, idempotencyKey); found || err != nil {
		return err
	}

	result, err := p.Executor.Chat(ctx, rc, alias, req, traceID, idempotencyKey)
	if err != nil {
		return p.HandleExecutorError(c, err)
	}
	httputil.ApplyBudgetHeaders(c, result.BudgetStatus)

	// Enrich wide event with execution metrics
	if event, ok := logging.WideEventFromContext(ctx); ok {
		enrichWideEventFromChatResult(event, alias, &result)
	}

	respBody, err := convert(result.Response, alias)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	payload, err := json.Marshal(respBody)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
	}

	p.CacheResponse(ctx, idempotencyKey, payload)

	c.Set("Content-Type", "application/json")
	return c.Send(payload)
}

// enrichWideEventFromChatResult populates wide event fields from a chat result.
func enrichWideEventFromChatResult(event *logging.WideEvent, alias string, result *executor.ChatResult) {
	event.ModelAlias = alias
	event.SetModelContext(
		alias,
		result.Metrics.Provider,
		result.Metrics.ProviderModel,
		result.Metrics.Deployment,
	)
	event.SetExecutionMetrics(
		result.Metrics.LatencyMs,
		result.Metrics.RetryCount,
		result.Metrics.RouteCount,
	)
	event.SetUsageMetrics(
		int64(result.Response.Usage.PromptTokens),
		int64(result.Response.Usage.CompletionTokens),
		int64(result.Response.Usage.TotalTokens),
		result.Metrics.CostMicroUSD,
	)
	event.SetBudgetStatus(
		result.BudgetStatus.TotalCostCents,
		result.BudgetStatus.LimitCents-result.BudgetStatus.TotalCostCents,
		result.BudgetStatus.Exceeded,
	)
}

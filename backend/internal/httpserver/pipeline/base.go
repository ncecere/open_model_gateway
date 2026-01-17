// Package pipeline provides common utilities for HTTP request pipelines.
//
// # Pipeline Pattern
//
// The pipeline pattern encapsulates the common flow for handling HTTP requests that
// interact with the executor layer. Each pipeline embeds the Base struct to gain
// access to shared functionality:
//
//   - ValidateAlias: Model alias validation and tenant access checks
//   - CheckIdempotency/CacheResponse: Idempotency key handling
//   - HandleExecutorError: Consistent error response formatting
//   - SendJSONResponse/SendJSONWithIdempotency: Response serialization with budget headers
//   - CheckRoutes: Route availability verification
//   - GetRequestContext: Request context extraction
//
// # Creating a New Pipeline
//
// To create a new pipeline, embed the Base struct and use its methods:
//
//	type myPipeline struct {
//	    *pipeline.Base
//	}
//
//	func newMyPipeline(container *app.Container, exec *executor.Executor) *myPipeline {
//	    return &myPipeline{Base: pipeline.NewBase(container, exec)}
//	}
//
//	func (p *myPipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, req MyRequest) error {
//	    ctx := c.UserContext()
//
//	    // Validate and check idempotency
//	    alias, err := p.ValidateAlias(c, rc.TenantID, req.Model)
//	    if err != nil {
//	        return err
//	    }
//	    if found, err := p.CheckIdempotency(c, ctx, req.IdempotencyKey); found || err != nil {
//	        return err
//	    }
//
//	    // Execute operation via executor
//	    result, err := p.Executor.MyOperation(ctx, rc, alias, req)
//	    if err != nil {
//	        return p.HandleExecutorError(c, err)
//	    }
//
//	    // Return response with idempotency caching
//	    return p.SendJSONWithIdempotency(c, ctx, result.BudgetStatus, result.Response, req.IdempotencyKey)
//	}
//
// # Extension Points
//
// Pipelines can extend the Base functionality by:
//   - Adding custom validation before calling executor methods
//   - Using custom response converters (see chatPipeline.ExecuteWithConverter)
//   - Adding streaming support (see chatStreamPipeline)
//   - Adding operation-specific logic (see imagePipeline with OverrideCost)
package pipeline

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/cache"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

// Base provides common pipeline functionality for HTTP handlers.
type Base struct {
	Container   *app.Container
	Executor    *executor.Executor
	Idempotency *cache.IdempotencyCache
}

// NewBase creates a new Base pipeline with the given container and executor.
func NewBase(container *app.Container, exec *executor.Executor) *Base {
	var idem *cache.IdempotencyCache
	if container.Telemetry != nil {
		idem = container.Telemetry.Idempotency
	}
	return &Base{
		Container:   container,
		Executor:    exec,
		Idempotency: idem,
	}
}

// ValidateAlias checks if the alias is valid and the tenant is allowed to use it.
// Returns an error response if validation fails, nil if successful.
func (b *Base) ValidateAlias(c *fiber.Ctx, tenantID uuid.UUID, alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if !b.Container.IsModelAllowed(tenantID, alias) {
		return "", httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}
	return alias, nil
}

// CheckIdempotency checks if a response for this idempotency key is cached.
// Returns true and sends the cached response if found.
func (b *Base) CheckIdempotency(c *fiber.Ctx, ctx context.Context, key string) (bool, error) {
	if key == "" || b.Idempotency == nil {
		return false, nil
	}
	if data, ok := b.Idempotency.Get(ctx, key); ok {
		c.Set("Content-Type", "application/json")
		return true, c.Send(data)
	}
	return false, nil
}

// CacheResponse stores the response for idempotency.
func (b *Base) CacheResponse(ctx context.Context, key string, payload []byte) {
	if key == "" || b.Idempotency == nil {
		return
	}
	b.Idempotency.Set(ctx, key, payload)
}

// HandleExecutorError handles errors returned from executor methods.
// Returns an appropriate HTTP error response.
func (b *Base) HandleExecutorError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	if status, msg, ok := executor.AsAPIError(err); ok {
		return httputil.WriteError(c, status, msg)
	}
	return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
}

// SendJSONResponse sends a JSON response with budget headers.
func (b *Base) SendJSONResponse(c *fiber.Ctx, budgetStatus usagepipeline.BudgetStatus, body interface{}) error {
	httputil.ApplyBudgetHeaders(c, budgetStatus)
	return c.JSON(body)
}

// SendJSONWithIdempotency sends a JSON response, caching it for idempotency if a key is provided.
func (b *Base) SendJSONWithIdempotency(c *fiber.Ctx, ctx context.Context, budgetStatus usagepipeline.BudgetStatus, body interface{}, idempotencyKey string) error {
	httputil.ApplyBudgetHeaders(c, budgetStatus)

	payload, err := json.Marshal(body)
	if err != nil {
		return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
	}

	b.CacheResponse(ctx, idempotencyKey, payload)

	c.Set("Content-Type", "application/json")
	return c.Send(payload)
}

// GetRequestContext extracts the request context from the fiber context.
func (b *Base) GetRequestContext(c *fiber.Ctx) (*requestctx.Context, error) {
	ctx := c.UserContext()
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return nil, httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	return rc, nil
}

// CheckRoutes verifies routes are available for the alias.
func (b *Base) CheckRoutes(c *fiber.Ctx, alias string) error {
	if b.Container.Routing == nil || b.Container.Routing.Engine == nil {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "routing not configured")
	}
	routes := b.Container.Routing.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}
	return nil
}

// EnrichWideEvent enriches the wide event from executor metrics and budget status.
// This is a helper for pipelines that use the executor pattern.
func (b *Base) EnrichWideEvent(c *fiber.Ctx, alias string, metrics executor.ExecutionMetrics, budgetStatus usagepipeline.BudgetStatus, usage models.Usage) {
	ctx := c.UserContext()
	event, ok := logging.WideEventFromContext(ctx)
	if !ok {
		return
	}

	event.SetModelContext(alias, metrics.Provider, metrics.ProviderModel, metrics.Deployment)
	event.SetExecutionMetrics(metrics.LatencyMs, metrics.RetryCount, metrics.RouteCount)
	event.SetUsageMetrics(
		int64(usage.PromptTokens),
		int64(usage.CompletionTokens),
		int64(usage.TotalTokens),
		metrics.CostMicroUSD,
	)

	remainingCents := budgetStatus.LimitCents - budgetStatus.TotalCostCents
	if remainingCents < 0 {
		remainingCents = 0
	}
	event.SetBudgetStatus(budgetStatus.TotalCostCents, remainingCents, budgetStatus.Exceeded)
}

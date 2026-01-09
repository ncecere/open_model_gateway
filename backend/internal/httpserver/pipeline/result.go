package pipeline

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

// Result represents the result of an executor operation.
type Result[T any] struct {
	Response     T
	BudgetStatus usagepipeline.BudgetStatus
}

// ResultHandler handles executor results with common patterns.
type ResultHandler[T any] struct {
	base *Base
}

// NewResultHandler creates a new ResultHandler.
func NewResultHandler[T any](base *Base) *ResultHandler[T] {
	return &ResultHandler[T]{base: base}
}

// Handle processes an executor result with a converter function.
func (h *ResultHandler[T]) Handle(
	c *fiber.Ctx,
	ctx context.Context,
	result Result[T],
	idempotencyKey string,
	convert func(T) interface{},
) error {
	body := convert(result.Response)
	return h.base.SendJSONWithIdempotency(c, ctx, result.BudgetStatus, body, idempotencyKey)
}

// HandleSimple processes an executor result without idempotency.
func (h *ResultHandler[T]) HandleSimple(
	c *fiber.Ctx,
	result Result[T],
	convert func(T) interface{},
) error {
	body := convert(result.Response)
	return h.base.SendJSONResponse(c, result.BudgetStatus, body)
}

// ExecuteAndHandle runs an executor function and handles the result.
func ExecuteAndHandle[T any](
	c *fiber.Ctx,
	base *Base,
	idempotencyKey string,
	execute func() (T, usagepipeline.BudgetStatus, error),
	convert func(T) interface{},
) error {
	ctx := c.UserContext()

	// Check idempotency cache first
	if found, err := base.CheckIdempotency(c, ctx, idempotencyKey); found || err != nil {
		return err
	}

	// Execute the operation
	response, budgetStatus, err := execute()
	if err != nil {
		return base.HandleExecutorError(c, err)
	}

	// Convert and send response
	body := convert(response)
	httputil.ApplyBudgetHeaders(c, budgetStatus)

	if idempotencyKey != "" {
		payload, err := json.Marshal(body)
		if err != nil {
			return httputil.WriteError(c, fiber.StatusInternalServerError, "failed to encode response")
		}
		base.CacheResponse(ctx, idempotencyKey, payload)
		c.Set("Content-Type", "application/json")
		return c.Send(payload)
	}

	return c.JSON(body)
}

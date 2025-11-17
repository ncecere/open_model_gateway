package httputil

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

const budgetHeaderLimit = "X-Budget-Limit-Cents"
const budgetHeaderTotal = "X-Budget-Total-Cents"
const budgetHeaderRemaining = "X-Budget-Remaining-Cents"
const budgetHeaderWarning = "X-Budget-Warning"
const budgetHeaderExceeded = "X-Budget-Exceeded"

// ApplyBudgetHeaders sets the standard budget headers for OpenAI-compatible responses.
func ApplyBudgetHeaders(c *fiber.Ctx, status usagepipeline.BudgetStatus) {
	if c == nil {
		return
	}
	c.Set(budgetHeaderLimit, strconv.FormatInt(status.LimitCents, 10))
	c.Set(budgetHeaderTotal, strconv.FormatInt(status.TotalCostCents, 10))
	remaining := status.LimitCents - status.TotalCostCents
	if remaining < 0 {
		remaining = 0
	}
	c.Set(budgetHeaderRemaining, strconv.FormatInt(remaining, 10))
	if status.Warning {
		c.Set(budgetHeaderWarning, "true")
	}
	if status.Exceeded {
		c.Set(budgetHeaderExceeded, "true")
	}
	c.Locals(budgetHeaderLimit, status)
}

// BudgetHeaderMiddleware automatically injects budget headers after request completion.
func BudgetHeaderMiddleware(container *app.Container) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if container == nil || container.UsageLogger == nil {
			return c.Next()
		}
		err := c.Next()
		if hasBudgetHeaders(c) {
			return err
		}
		ctx := c.UserContext()
		rc, ok := requestctx.FromContext(ctx)
		if !ok || rc == nil {
			return err
		}
		status, budgetErr := container.UsageLogger.CheckBudget(ctx, rc, time.Now().UTC())
		if budgetErr != nil {
			return err
		}
		ApplyBudgetHeaders(c, status)
		return err
	}
}

func hasBudgetHeaders(c *fiber.Ctx) bool {
	if c == nil {
		return true
	}
	return len(c.Response().Header.Peek(budgetHeaderLimit)) > 0
}

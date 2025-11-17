package public

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type moderationPipeline struct {
	container *app.Container
	executor  *executor.Executor
}

func newModerationPipeline(container *app.Container, exec *executor.Executor) *moderationPipeline {
	return &moderationPipeline{container: container, executor: exec}
}

func (p *moderationPipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, alias string, inputs []string) error {
	ctx := c.UserContext()
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if !p.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	routes := p.container.Engine.SelectRoutes(alias)
	if len(routes) == 0 {
		return httputil.WriteError(c, fiber.StatusServiceUnavailable, "no backend available for model")
	}

	traceID := traceIDFromContext(c)
	result, err := p.executor.Moderate(ctx, rc, alias, inputs, traceID)
	if err != nil {
		if status, msg, ok := executor.AsAPIError(err); ok {
			return httputil.WriteError(c, status, msg)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	httputil.ApplyBudgetHeaders(c, result.BudgetStatus)
	return c.JSON(convertModerationHTTPResponse(result.Response, alias))
}

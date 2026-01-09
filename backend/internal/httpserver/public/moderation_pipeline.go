package public

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/pipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type moderationPipeline struct {
	*pipeline.Base
}

func newModerationPipeline(container *app.Container, exec *executor.Executor) *moderationPipeline {
	return &moderationPipeline{Base: pipeline.NewBase(container, exec)}
}

func (p *moderationPipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, alias string, inputs []string) error {
	ctx := c.UserContext()

	// Validate alias and tenant access
	alias, err := p.ValidateAlias(c, rc.TenantID, alias)
	if err != nil {
		return err
	}

	// Check routes are available
	if err := p.CheckRoutes(c, alias); err != nil {
		return err
	}

	traceID := traceIDFromContext(c)
	result, err := p.Executor.Moderate(ctx, rc, alias, inputs, traceID)
	if err != nil {
		return p.HandleExecutorError(c, err)
	}

	return p.SendJSONResponse(c, result.BudgetStatus, convertModerationHTTPResponse(result.Response, alias))
}

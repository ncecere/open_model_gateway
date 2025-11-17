package public

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type embeddingPipeline struct {
	container *app.Container
	executor  *executor.Executor
}

func newEmbeddingPipeline(container *app.Container, exec *executor.Executor) *embeddingPipeline {
	return &embeddingPipeline{container: container, executor: exec}
}

func (p *embeddingPipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, alias string, req models.EmbeddingsRequest) error {
	ctx := c.UserContext()
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return httputil.WriteError(c, fiber.StatusBadRequest, "model is required")
	}
	if !p.container.IsModelAllowed(rc.TenantID, alias) {
		return httputil.WriteError(c, fiber.StatusForbidden, "model not enabled for tenant")
	}

	traceID := traceIDFromContext(c)
	result, err := p.executor.Embed(ctx, rc, alias, req, traceID)
	if err != nil {
		if status, msg, ok := executor.AsAPIError(err); ok {
			return httputil.WriteError(c, status, msg)
		}
		return httputil.WriteError(c, fiber.StatusInternalServerError, err.Error())
	}
	httputil.ApplyBudgetHeaders(c, result.BudgetStatus)
	return c.JSON(convertEmbeddingResponse(result.Response, alias))
}

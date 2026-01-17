package public

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/pipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

type embeddingPipeline struct {
	*pipeline.Base
}

func newEmbeddingPipeline(container *app.Container, exec *executor.Executor) *embeddingPipeline {
	return &embeddingPipeline{Base: pipeline.NewBase(container, exec)}
}

func (p *embeddingPipeline) Execute(c *fiber.Ctx, rc *requestctx.Context, alias string, req models.EmbeddingsRequest) error {
	ctx := c.UserContext()

	// Validate alias and tenant access
	alias, err := p.ValidateAlias(c, rc.TenantID, alias)
	if err != nil {
		return err
	}

	traceID := traceIDFromContext(c)
	result, err := p.Executor.Embed(ctx, rc, alias, req, traceID)
	if err != nil {
		return p.HandleExecutorError(c, err)
	}

	// Enrich wide event with execution metrics
	p.EnrichWideEvent(c, alias, result.Metrics, result.BudgetStatus, result.Response.Usage)

	return p.SendJSONResponse(c, result.BudgetStatus, convertEmbeddingResponse(result.Response, alias))
}

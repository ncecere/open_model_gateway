package public

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
)

// batchPipeline centralizes batching budget checks + service access.
type batchPipeline struct {
	container *app.Container
}

func newBatchPipeline(container *app.Container) *batchPipeline {
	return &batchPipeline{container: container}
}

func (p *batchPipeline) create(c *fiber.Ctx, rc *requestctx.Context, params batchsvc.CreateParams) (batchsvc.Batch, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return batchsvc.Batch{}, err
	}
	return svc.Create(c.UserContext(), params)
}

func (p *batchPipeline) list(c *fiber.Ctx, rc *requestctx.Context, limit int32, afterID *uuid.UUID) ([]batchsvc.Batch, bool, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return nil, false, err
	}
	return svc.ListCursor(c.UserContext(), rc.TenantID, limit, afterID)
}

func (p *batchPipeline) get(c *fiber.Ctx, rc *requestctx.Context, id uuid.UUID) (batchsvc.Batch, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return batchsvc.Batch{}, err
	}
	return svc.Get(c.UserContext(), rc.TenantID, id)
}

func (p *batchPipeline) cancel(c *fiber.Ctx, rc *requestctx.Context, id uuid.UUID) (batchsvc.Batch, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return batchsvc.Batch{}, err
	}
	return svc.Cancel(c.UserContext(), rc.TenantID, id)
}

func (p *batchPipeline) service(c *fiber.Ctx, rc *requestctx.Context) (*batchsvc.Service, error) {
	if rc == nil {
		return nil, httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	svc := p.container.Batches
	if svc == nil {
		return nil, httputil.WriteError(c, fiber.StatusNotImplemented, "batches not enabled")
	}
	if p.container.UsageLogger != nil {
		status, err := p.container.UsageLogger.CheckBudget(c.UserContext(), rc, time.Now().UTC())
		if err != nil {
			return nil, httputil.WriteError(c, fiber.StatusInternalServerError, "failed to evaluate budget")
		}
		httputil.ApplyBudgetHeaders(c, status)
		if status.Exceeded {
			return nil, httputil.WriteError(c, fiber.StatusForbidden, "tenant budget exceeded")
		}
	}
	return svc, nil
}

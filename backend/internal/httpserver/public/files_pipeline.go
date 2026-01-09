package public

import (
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

// filesPipeline centralizes common validation + budget checks for file operations.
type filesPipeline struct {
	container *app.Container
}

func newFilesPipeline(container *app.Container) *filesPipeline {
	return &filesPipeline{container: container}
}

func (p *filesPipeline) list(c *fiber.Ctx, rc *requestctx.Context, opts filesvc.ListOptions) (filesvc.ListResult, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return filesvc.ListResult{}, err
	}
	return svc.List(c.UserContext(), rc.TenantID, opts)
}

func (p *filesPipeline) open(c *fiber.Ctx, rc *requestctx.Context, id uuid.UUID) (io.ReadCloser, filesvc.FileRecord, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return nil, filesvc.FileRecord{}, err
	}
	return svc.Open(c.UserContext(), rc.TenantID, id)
}

func (p *filesPipeline) delete(c *fiber.Ctx, rc *requestctx.Context, id uuid.UUID) error {
	svc, err := p.service(c, rc)
	if err != nil {
		return err
	}
	return svc.Delete(c.UserContext(), rc.TenantID, id)
}

func (p *filesPipeline) upload(c *fiber.Ctx, rc *requestctx.Context, params filesvc.UploadParams) (filesvc.FileRecord, error) {
	svc, err := p.service(c, rc)
	if err != nil {
		return filesvc.FileRecord{}, err
	}
	return svc.Upload(c.UserContext(), params)
}

func (p *filesPipeline) service(c *fiber.Ctx, rc *requestctx.Context) (*filesvc.Service, error) {
	if rc == nil {
		return nil, httputil.WriteError(c, fiber.StatusInternalServerError, "request context missing")
	}
	svc := p.container.Services.Files
	if svc == nil {
		return nil, httputil.WriteError(c, fiber.StatusNotImplemented, "files service disabled")
	}
	if p.container.Telemetry.UsageLogger != nil {
		status, err := p.container.Telemetry.UsageLogger.CheckBudget(c.UserContext(), rc, time.Now().UTC())
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

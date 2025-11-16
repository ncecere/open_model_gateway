package app

import (
	"context"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

// UsageLogger defines the subset of methods handlers depend on for budget checks.
type UsageLogger interface {
	LoadCatalog(entries []config.ModelCatalogEntry)
	SetConfig(cfg config.BudgetConfig)
	CheckBudget(ctx context.Context, rc *requestctx.Context, now time.Time) (usagepipeline.BudgetStatus, error)
	Record(ctx context.Context, rec usagepipeline.Record) (usagepipeline.BudgetStatus, error)
}

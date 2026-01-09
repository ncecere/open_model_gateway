package containers

import (
	"context"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/cache"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

// UsageLogger defines the subset of methods handlers depend on for budget checks.
type UsageLogger interface {
	LoadCatalog(entries []config.ModelCatalogEntry)
	SetConfig(cfg config.BudgetConfig)
	CheckBudget(ctx context.Context, rc *requestctx.Context, now time.Time) (usagepipeline.BudgetStatus, error)
	Record(ctx context.Context, rec usagepipeline.Record) (usagepipeline.BudgetStatus, error)
}

// TelemetryContainer holds observability, usage logging, and metrics services.
type TelemetryContainer struct {
	Observability        *observability.Provider
	UsageLogger          UsageLogger
	ProviderTelemetry    *providermetrics.Recorder
	ProviderTelemetrySvc *providermetrics.Service
	Idempotency          *cache.IdempotencyCache
	TelemetryCancel      context.CancelFunc
}

// NewTelemetryContainer creates a new TelemetryContainer.
func NewTelemetryContainer(
	obs *observability.Provider,
	usageLogger UsageLogger,
	providerRec *providermetrics.Recorder,
	providerSvc *providermetrics.Service,
	idempotency *cache.IdempotencyCache,
	cancel context.CancelFunc,
) *TelemetryContainer {
	return &TelemetryContainer{
		Observability:        obs,
		UsageLogger:          usageLogger,
		ProviderTelemetry:    providerRec,
		ProviderTelemetrySvc: providerSvc,
		Idempotency:          idempotency,
		TelemetryCancel:      cancel,
	}
}

// RecordProviderTelemetry fans out provider samples to metrics and Redis windows.
func (t *TelemetryContainer) RecordProviderTelemetry(ctx context.Context, sample providermetrics.Sample) {
	if t == nil {
		return
	}
	if t.ProviderTelemetry != nil {
		_ = t.ProviderTelemetry.RecordSample(ctx, sample)
	}
	if t.Observability != nil {
		t.Observability.RecordProviderMetrics(sample)
	}
}

package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

// Telemetry bundles observability, usage logging, and supporting services.
type Telemetry struct {
	Observability *observability.Provider
	UsageLogger   UsageLogger
	Files         *filesvc.Service
	Batches       *batchsvc.Service
	AdminConfig   *adminconfigsvc.Service
}

// BuildTelemetry initializes observability, usage logging, and resource services.
func BuildTelemetry(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, queries *db.Queries, entries []config.ModelCatalogEntry) (*Telemetry, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if pool == nil || queries == nil {
		return nil, fmt.Errorf("database primitives required")
	}
	obsProvider, err := observability.Setup(ctx, cfg.Observability)
	if err != nil {
		return nil, fmt.Errorf("setup observability: %w", err)
	}
	alertSink := usagepipeline.NewCompositeSink(
		usagepipeline.NewEmailSink(cfg.Budgets.Alert.SMTP, slog.Default()),
		usagepipeline.NewWebhookSink(cfg.Budgets.Alert.Webhook, slog.Default()),
		usagepipeline.NewLogAlertSink(slog.Default()),
	)
	usageLogger := usagepipeline.NewLogger(ctx, pool, queries, cfg.Budgets, alertSink, obsProvider)
	usageLogger.LoadCatalog(entries)

	blobStore, err := blob.New(ctx, cfg.Files)
	if err != nil {
		return nil, fmt.Errorf("init blob store: %w", err)
	}
	filesService := filesvc.NewService(queries, blobStore, &cfg.Files)
	batchesService := batchsvc.NewService(pool, queries, filesService, &cfg.Batches)
	adminConfigService := adminconfigsvc.NewService(queries, cfg, filesService, batchesService)
	return &Telemetry{
		Observability: obsProvider,
		UsageLogger:   usageLogger,
		Files:         filesService,
		Batches:       batchesService,
		AdminConfig:   adminConfigService,
	}, nil
}

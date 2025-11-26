package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

// Telemetry bundles observability, usage logging, and supporting services.
type Telemetry struct {
	Observability *observability.Provider
	UsageLogger   UsageLogger
	Files         *filesvc.Service
	Batches       *batchsvc.Service
	AdminConfig   *adminconfigsvc.Service
	ProviderRec   *providermetrics.Recorder
	ProviderSvc   *providermetrics.Service
}

// BuildTelemetry initializes observability, usage logging, and resource services.
// Returns telemetry bundle and an optional cancel func for background workers.
func BuildTelemetry(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, queries *db.Queries, entries []config.ModelCatalogEntry, redisClient *redis.Client) (*Telemetry, context.CancelFunc, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is required")
	}
	if pool == nil || queries == nil {
		return nil, nil, fmt.Errorf("database primitives required")
	}
	if redisClient == nil {
		return nil, nil, fmt.Errorf("redis client required for telemetry")
	}
	obsProvider, err := observability.Setup(ctx, cfg.Observability)
	if err != nil {
		return nil, nil, fmt.Errorf("setup observability: %w", err)
	}
	alertSink := usagepipeline.NewCompositeSink(
		usagepipeline.NewEmailSink(cfg.Budgets.Alert.SMTP, cfg.Public.BaseURL, slog.Default()),
		usagepipeline.NewWebhookSink(cfg.Budgets.Alert.Webhook, slog.Default()),
		usagepipeline.NewLogAlertSink(slog.Default()),
	)
	usageLogger := usagepipeline.NewLogger(ctx, pool, queries, cfg.Budgets, alertSink, obsProvider)
	usageLogger.LoadCatalog(entries)

	blobStore, err := blob.New(ctx, cfg.Files)
	if err != nil {
		return nil, nil, fmt.Errorf("init blob store: %w", err)
	}
	filesService := filesvc.NewService(queries, blobStore, &cfg.Files)
	batchesService := batchsvc.NewService(pool, queries, filesService, &cfg.Batches)
	adminConfigService := adminconfigsvc.NewService(queries, cfg, filesService, batchesService)
	providerRec := providermetrics.NewRecorder(redisClient, cfg.Telemetry.Provider.WindowSize)
	providerAlertSink := providermetrics.NewCompositeAlertSink(
		providermetrics.NewLogAlertSink(slog.Default()),
		providermetrics.NewEmailAlertSink(cfg.Budgets.Alert.SMTP, cfg.Public.BaseURL, cfg.Budgets.Alert.Emails, slog.Default()),
		providermetrics.NewWebhookAlertSink(cfg.Budgets.Alert.Webhook, cfg.Budgets.Alert.Webhooks, slog.Default()),
	)
	providerSvc := providermetrics.NewService(providerRec, queries, cfg.Telemetry, cfg.Budgets, providerAlertSink, slog.Default())

	telemetry := &Telemetry{
		Observability: obsProvider,
		UsageLogger:   usageLogger,
		Files:         filesService,
		Batches:       batchesService,
		AdminConfig:   adminConfigService,
		ProviderRec:   providerRec,
		ProviderSvc:   providerSvc,
	}

	var cancel context.CancelFunc
	if cfg.Telemetry.Provider.Enabled && cfg.Telemetry.Provider.EvaluationInterval > 0 {
		ctx, c := context.WithCancel(ctx)
		cancel = c
		go func() {
			ticker := time.NewTicker(cfg.Telemetry.Provider.EvaluationInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if providerSvc != nil {
						if _, err := providerSvc.Tick(context.Background()); err != nil {
							slog.Default().Warn("provider telemetry tick failed", slog.String("err", err.Error()))
						}
					}
				}
			}
		}()
	}

	return telemetry, cancel, nil
}

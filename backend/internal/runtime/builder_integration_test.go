package runtime

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/redisclient"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

var (
	_, runtimeTestFile, _, _ = runtime.Caller(0)
	backendRoot              = filepath.Clean(filepath.Join(filepath.Dir(runtimeTestFile), "..", ".."))
	migrationsDir            = filepath.Join(backendRoot, "migrations")
	schemaDir                = filepath.Join(backendRoot, "sql", "schema")
)

func TestRuntimeBuilderWiresCoreDependencies(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pgStop, dsn := startTestPostgres(t)
	defer pgStop()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	filesDir := t.TempDir()
	cfg := testRuntimeConfig(dsn, fmt.Sprintf("redis://%s", mr.Addr()), filesDir)

	if err := applySchema(ctx, cfg.Database.URL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	pool, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	redisClient := redisclient.New(cfg.Redis)
	t.Cleanup(func() { _ = redisClient.Close() })

	builder := NewBuilder(Options{})
	builder.datastores = &Datastores{Config: cfg, DB: pool}
	builder.config = cfg
	builder.redis = redisClient
	builder.health = NewHealthChecks()
	builder.health.Register("postgres", pool.Ping)
	builder.health.Register("redis", func(ctx context.Context) error {
		return redisClient.Ping(ctx).Err()
	})

	runtime, err := builder.Build(ctx)
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })

	container := runtime.Container
	if container == nil {
		t.Fatalf("runtime container missing")
	}
	if container.Observability == nil {
		t.Fatalf("expected observability provider wired")
	}
	if container.UsageLogger == nil {
		t.Fatalf("expected usage logger wired")
	}
	if container.RateLimiter == nil {
		t.Fatalf("expected rate limiter wired")
	}
	if err := runBudgetScenario(ctx, container); err != nil {
		t.Fatalf("budget scenario failed: %v", err)
	}

	reporter := runtime.HealthReporter()
	if reporter == nil {
		t.Fatalf("expected health reporter")
	}
	results := reporter.Run(ctx)
	if len(results) == 0 {
		t.Fatalf("expected health results")
	}
	seen := map[string]bool{}
	for _, res := range results {
		if res.Err != nil {
			t.Fatalf("check %s failed: %v", res.Name, res.Err)
		}
		seen[res.Name] = true
	}
	for _, name := range []string{"postgres", "redis", "router"} {
		if !seen[name] {
			t.Fatalf("missing %s health check", name)
		}
	}
}

func startTestPostgres(t *testing.T) (func(), string) {
	t.Helper()
	baseDir := t.TempDir()
	port := freePort(t)
	const (
		username     = "postgres"
		password     = "postgres"
		databaseName = "omg_test"
	)
	cfg := embeddedpostgres.DefaultConfig().
		DataPath(filepath.Join(baseDir, "data")).
		RuntimePath(filepath.Join(baseDir, "runtime")).
		BinariesPath(filepath.Join(baseDir, "bin")).
		Username(username).
		Password(password).
		Database(databaseName).
		Port(uint32(port))

	db := embeddedpostgres.NewDatabase(cfg)
	if err := db.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	return func() {
		_ = db.Stop()
	}, fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", username, password, port, databaseName)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	return addr.Port
}

func testRuntimeConfig(pgURL, redisURL, filesDir string) *config.Config {
	encKey := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	trueVal := true
	return &config.Config{
		Server: config.ServerConfig{
			ListenAddr:        ":0",
			BodyLimitMB:       16,
			SyncTimeout:       5 * time.Second,
			StreamIdleTimeout: 5 * time.Second,
			StreamMaxDuration: 30 * time.Second,
		},
		Database: config.DatabaseConfig{
			URL:           pgURL,
			RunMigrations: true,
			MigrationsDir: migrationsDir,
		},
		Redis: config.RedisConfig{
			URL: redisURL,
		},
		RateLimits: config.RateLimitConfig{
			DefaultTokensPerMinute:        10000,
			DefaultRequestsPerMinute:      100,
			DefaultParallelRequestsKey:    5,
			DefaultParallelRequestsTenant: 5,
		},
		Budgets: config.BudgetConfig{
			DefaultUSD:           1000,
			WarningThresholdPerc: 0.8,
			RefreshSchedule:      "monthly",
			Alert: config.BudgetAlertConfig{
				Enabled: false,
				SMTP:    config.SMTPConfig{},
				Webhook: config.WebhookConfig{},
			},
		},
		Reporting: config.ReportingConfig{Timezone: "UTC"},
		Providers: config.ProviderConfig{
			OpenAI: config.OpenAIProviderConfig{APIKey: "test-openai-key"},
		},
		Files: config.FilesConfig{
			Storage:        "local",
			MaxSizeMB:      25,
			DefaultTTL:     time.Hour,
			MaxTTL:         24 * time.Hour,
			SweepInterval:  time.Minute,
			SweepBatchSize: 10,
			EncryptionKey:  encKey,
			Local:          config.FilesLocalConfig{Directory: filesDir},
		},
		Audio: config.AudioConfig{MaxUploadMB: 25},
		Batches: config.BatchesConfig{
			MaxRequests:    50,
			MaxConcurrency: 5,
			DefaultTTL:     30 * time.Minute,
			MaxTTL:         2 * time.Hour,
		},
		Observability: config.ObservabilityConfig{EnableMetrics: true},
		Health:        config.HealthConfig{CheckInterval: time.Minute, RollingWindow: 5, Cooldown: time.Minute},
		Admin: config.AdminConfig{
			Session: config.AdminSessionConfig{
				JWTSecret:       "test-secret",
				AccessTokenTTL:  time.Hour,
				RefreshTokenTTL: 24 * time.Hour,
				CookieName:      "router",
			},
			Local: config.LocalAuthConfig{Enabled: true},
			OIDC:  config.OIDCConfig{},
		},
		ModelCatalog: []config.ModelCatalogEntry{
			{
				Alias:         "test-chat",
				Provider:      "openai",
				ProviderModel: "gpt-4o-mini",
				ModelType:     "chat",
				PriceInput:    0.000001,
				PriceOutput:   0.000002,
				Currency:      "usd",
			},
		},
		Bootstrap: config.BootstrapConfig{
			Tenants: []config.BootstrapTenant{{Name: "Acme"}},
			AdminUsers: []config.BootstrapAdminUser{{
				Email:      "admin@example.com",
				Name:       "Admin",
				Password:   "password123",
				SuperAdmin: &trueVal,
			}},
		},
	}
}

func applySchema(ctx context.Context, dbURL string) error {
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("read schema dir: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open schema database: %w", err)
	}
	defer sqlDB.Close()
	for _, name := range files {
		path := filepath.Join(schemaDir, name)
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		contentStr := string(contents)
		if name == "000_init.sql" {
			if err := ensureTriggerFunction(ctx, sqlDB, contentStr); err != nil {
				return err
			}
		}
		if _, err := sqlDB.ExecContext(ctx, contentStr); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}

func runBudgetScenario(ctx context.Context, container *app.Container) error {
	tenant, err := container.Queries.GetTenantByName(ctx, "Acme")
	if err != nil {
		return fmt.Errorf("load bootstrap tenant: %w", err)
	}
	tenantID, err := uuid.FromBytes(tenant.ID.Bytes[:])
	if err != nil {
		return fmt.Errorf("tenant id: %w", err)
	}
	reqCtx := &requestctx.Context{TenantID: tenantID}
	budget, err := container.UsageLogger.CheckBudget(ctx, reqCtx, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("initial budget check: %w", err)
	}
	rec := usagepipeline.Record{
		Context:   reqCtx,
		Alias:     "test-chat",
		Provider:  "openai",
		Usage:     models.Usage{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500},
		Latency:   10 * time.Millisecond,
		Status:    fiber.StatusOK,
		Timestamp: time.Now().UTC(),
		Success:   true,
	}
	override := int64(250)
	rec.OverrideCostCents = &override
	if _, err := container.UsageLogger.Record(ctx, rec); err != nil {
		return fmt.Errorf("async record: %w", err)
	}
	if closer, ok := container.UsageLogger.(interface{ Shutdown(context.Context) error }); ok {
		if err := closer.Shutdown(ctx); err != nil {
			return fmt.Errorf("flush usage logger: %w", err)
		}
	}
	budgetAfter, err := container.UsageLogger.CheckBudget(ctx, reqCtx, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("post-record budget check: %w", err)
	}
	if budgetAfter.TotalCostCents <= budget.TotalCostCents {
		return fmt.Errorf("expected budget cost to increase (before=%d after=%d)", budget.TotalCostCents, budgetAfter.TotalCostCents)
	}
	return nil
}

func ensureTriggerFunction(ctx context.Context, db *sql.DB, content string) error {
	const marker = "CREATE OR REPLACE FUNCTION trigger_set_timestamp()"
	idx := strings.Index(content, marker)
	if idx == -1 {
		return nil
	}
	rest := content[idx:]
	endMarker := "$func$ LANGUAGE plpgsql;"
	endIdx := strings.Index(rest, endMarker)
	if endIdx == -1 {
		return nil
	}
	endIdx += len(endMarker)
	blk := rest[:endIdx]
	if _, err := db.ExecContext(ctx, blk); err != nil {
		return fmt.Errorf("init trigger function: %w", err)
	}
	return nil
}

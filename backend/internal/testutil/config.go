package testutil

import (
	"encoding/base64"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// TestConfigOptions allows customizing test configuration.
type TestConfigOptions struct {
	PostgresURL string
	RedisURL    string
	FilesDir    string
}

// NewTestConfig creates a minimal test configuration.
func NewTestConfig(opts TestConfigOptions) *config.Config {
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
			URL:           opts.PostgresURL,
			RunMigrations: true,
			MigrationsDir: MigrationsDir,
		},
		Redis: config.RedisConfig{
			URL: opts.RedisURL,
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
			Local:          config.FilesLocalConfig{Directory: opts.FilesDir},
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
				JWTSecret:       "test-secret-key-for-testing-only",
				AccessTokenTTL:  time.Hour,
				RefreshTokenTTL: 24 * time.Hour,
				CookieName:      "router_test",
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
			Tenants: []config.BootstrapTenant{{Name: "TestTenant"}},
			AdminUsers: []config.BootstrapAdminUser{{
				Email:      "admin@test.local",
				Name:       "Test Admin",
				Password:   "testpassword123",
				SuperAdmin: &trueVal,
			}},
		},
	}
}

// NewMinimalTestConfig creates a minimal config without bootstrap data.
func NewMinimalTestConfig(opts TestConfigOptions) *config.Config {
	cfg := NewTestConfig(opts)
	cfg.Bootstrap = config.BootstrapConfig{}
	return cfg
}

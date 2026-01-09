package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "router.yaml")

	configContent := `
server:
  listen_addr: ":9090"
  body_limit_mb: 50

database:
  url: "postgres://test:test@localhost:5432/testdb"
  run_migrations: false

redis:
  url: "redis://localhost:6379"

rate_limits:
  default_tokens_per_minute: 500000
  default_requests_per_minute: 500
  default_parallel_requests_key: 5
  default_parallel_requests_tenant: 50

budgets:
  default_usd: 50.0
  warning_threshold_perc: 0.7
  alert:
    enabled: false

admin:
  session:
    jwt_secret: "test-secret-key-for-testing"
    access_token_ttl: "30m"
    refresh_token_ttl: "48h"
    cookie_name: "test_session"
  local:
    enabled: true

files:
  max_size_mb: 100

batches:
  max_requests: 1000
  max_concurrency: 10

telemetry:
  provider:
    enabled: true
    evaluation_interval: "2m"
    window_size: "10m"
    incident_retention_days: 14
    defaults:
      latency_p95_ms: 3000
      error_rate_threshold: 0.05
      timeout_rate_threshold: 0.02
      min_samples: 100
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(Options{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify server config
	if cfg.Server.ListenAddr != ":9090" {
		t.Errorf("expected listen_addr :9090, got %s", cfg.Server.ListenAddr)
	}
	if cfg.Server.BodyLimitMB != 50 {
		t.Errorf("expected body_limit_mb 50, got %d", cfg.Server.BodyLimitMB)
	}

	// Verify database config
	if cfg.Database.URL != "postgres://test:test@localhost:5432/testdb" {
		t.Errorf("unexpected database URL: %s", cfg.Database.URL)
	}
	if cfg.Database.RunMigrations != false {
		t.Error("expected run_migrations false")
	}

	// Verify rate limits
	if cfg.RateLimits.DefaultTokensPerMinute != 500000 {
		t.Errorf("expected default_tokens_per_minute 500000, got %d", cfg.RateLimits.DefaultTokensPerMinute)
	}

	// Verify budgets
	if cfg.Budgets.DefaultUSD != 50.0 {
		t.Errorf("expected default_usd 50.0, got %f", cfg.Budgets.DefaultUSD)
	}
	if cfg.Budgets.WarningThresholdPerc != 0.7 {
		t.Errorf("expected warning_threshold_perc 0.7, got %f", cfg.Budgets.WarningThresholdPerc)
	}

	// Verify admin config
	if cfg.Admin.Session.JWTSecret != "test-secret-key-for-testing" {
		t.Errorf("unexpected jwt_secret: %s", cfg.Admin.Session.JWTSecret)
	}
	if cfg.Admin.Session.AccessTokenTTL != 30*time.Minute {
		t.Errorf("expected access_token_ttl 30m, got %v", cfg.Admin.Session.AccessTokenTTL)
	}

	// Verify telemetry config
	if cfg.Telemetry.Provider.EvaluationInterval != 2*time.Minute {
		t.Errorf("expected evaluation_interval 2m, got %v", cfg.Telemetry.Provider.EvaluationInterval)
	}
	if cfg.Telemetry.Provider.Defaults.LatencyP95Ms != 3000 {
		t.Errorf("expected latency_p95_ms 3000, got %d", cfg.Telemetry.Provider.Defaults.LatencyP95Ms)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Create minimal config that relies on defaults
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "router.yaml")

	configContent := `
database:
  url: "postgres://test:test@localhost:5432/testdb"

redis:
  url: "redis://localhost:6379"

admin:
  session:
    jwt_secret: "test-secret"
  local:
    enabled: true

budgets:
  alert:
    enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(Options{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify defaults are applied
	if cfg.Server.ListenAddr != ":8080" {
		t.Errorf("expected default listen_addr :8080, got %s", cfg.Server.ListenAddr)
	}
	if cfg.Server.BodyLimitMB != 20 {
		t.Errorf("expected default body_limit_mb 20, got %d", cfg.Server.BodyLimitMB)
	}
	if cfg.RateLimits.DefaultTokensPerMinute != 1_000_000 {
		t.Errorf("expected default tokens_per_minute 1000000, got %d", cfg.RateLimits.DefaultTokensPerMinute)
	}
	if cfg.Budgets.DefaultUSD != 100.0 {
		t.Errorf("expected default budget_usd 100.0, got %f", cfg.Budgets.DefaultUSD)
	}
	if cfg.Files.MaxSizeMB != 200 {
		t.Errorf("expected default files.max_size_mb 200, got %d", cfg.Files.MaxSizeMB)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		errContains string
	}{
		{
			name: "missing database url",
			config: `
redis:
  url: "redis://localhost:6379"
admin:
  session:
    jwt_secret: "test"
  local:
    enabled: true
budgets:
  alert:
    enabled: false
`,
			errContains: "ROUTER_DB_URL",
		},
		{
			name: "missing redis url",
			config: `
database:
  url: "postgres://test@localhost/db"
admin:
  session:
    jwt_secret: "test"
  local:
    enabled: true
budgets:
  alert:
    enabled: false
`,
			errContains: "ROUTER_REDIS_URL",
		},
		{
			name: "invalid budget warning threshold",
			config: `
database:
  url: "postgres://test@localhost/db"
redis:
  url: "redis://localhost:6379"
admin:
  session:
    jwt_secret: "test"
  local:
    enabled: true
budgets:
  warning_threshold_perc: 1.5
  alert:
    enabled: false
`,
			errContains: "warning_threshold_perc",
		},
		{
			name: "missing jwt secret",
			config: `
database:
  url: "postgres://test@localhost/db"
redis:
  url: "redis://localhost:6379"
admin:
  local:
    enabled: true
budgets:
  alert:
    enabled: false
`,
			errContains: "jwt_secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "router.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := Load(Options{ConfigFile: configPath})
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestNormalizeBudgetRefreshSchedule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "calendar_month"},
		{"calendar_month", "calendar_month"},
		{"CALENDAR_MONTH", "calendar_month"},
		{"weekly", "weekly"},
		{"WEEKLY", "weekly"},
		{"rolling_30d", "rolling_30d"},
		{"rolling_7d", "rolling_7d"},
		{"rolling_90d", "rolling_90d"},
		{"invalid", "calendar_month"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeBudgetRefreshSchedule(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBudgetRefreshSchedule(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBudgetRollingWindowDays(t *testing.T) {
	tests := []struct {
		input    string
		days     int
		ok       bool
	}{
		{"rolling_30d", 30, true},
		{"rolling_7d", 7, true},
		{"rolling_90d", 90, true},
		{"rolling_d", 30, true}, // defaults to 30
		{"calendar_month", 0, false},
		{"weekly", 0, false},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			days, ok := BudgetRollingWindowDays(tt.input)
			if ok != tt.ok {
				t.Errorf("BudgetRollingWindowDays(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if days != tt.days {
				t.Errorf("BudgetRollingWindowDays(%q) days = %d, want %d", tt.input, days, tt.days)
			}
		})
	}
}

func TestModelCatalogEntryIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ModelCatalogEntry{Enabled: tt.enabled}
			if entry.IsEnabled() != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", entry.IsEnabled(), tt.expected)
			}
		})
	}
}

func TestBootstrapAdminUserIsSuperAdmin(t *testing.T) {
	tests := []struct {
		name       string
		superAdmin *bool
		expected   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := BootstrapAdminUser{SuperAdmin: tt.superAdmin}
			if user.IsSuperAdmin() != tt.expected {
				t.Errorf("IsSuperAdmin() = %v, want %v", user.IsSuperAdmin(), tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

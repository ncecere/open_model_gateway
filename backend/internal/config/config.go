package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

// Config captures the runtime configuration for the router service.
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	RateLimits    RateLimitConfig     `mapstructure:"rate_limits"`
	Budgets       BudgetConfig        `mapstructure:"budgets"`
	Reporting     ReportingConfig     `mapstructure:"reporting"`
	Providers     ProviderConfig      `mapstructure:"providers"`
	Files         FilesConfig         `mapstructure:"files"`
	Audio         AudioConfig         `mapstructure:"audio"`
	Batches       BatchesConfig       `mapstructure:"batches"`
	Retention     RetentionConfig     `mapstructure:"retention"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Health        HealthConfig        `mapstructure:"health"`
	Admin         AdminConfig         `mapstructure:"admin"`
	ModelCatalog  []ModelCatalogEntry `mapstructure:"model_catalog"`
	Bootstrap     BootstrapConfig     `mapstructure:"bootstrap"`
}

type ServerConfig struct {
	ListenAddr            string        `mapstructure:"listen_addr"`
	BodyLimitMB           int           `mapstructure:"body_limit_mb"`
	SyncTimeout           time.Duration `mapstructure:"sync_timeout"`
	StreamIdleTimeout     time.Duration `mapstructure:"stream_idle_timeout"`
	StreamMaxDuration     time.Duration `mapstructure:"stream_max_duration"`
	ProviderTimeout       time.Duration `mapstructure:"provider_timeout"`
	ReadHeaderTimeout     time.Duration `mapstructure:"read_header_timeout"`
	GracefulShutdownDelay time.Duration `mapstructure:"graceful_shutdown_delay"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	RunMigrations   bool          `mapstructure:"run_migrations"`
	MigrationsDir   string        `mapstructure:"migrations_dir"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MinConns        int32         `mapstructure:"min_conns"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type AdminConfig struct {
	Session AdminSessionConfig `mapstructure:"session"`
	Local   LocalAuthConfig    `mapstructure:"local"`
	OIDC    OIDCConfig         `mapstructure:"oidc"`
}

type AdminSessionConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	CookieName      string        `mapstructure:"cookie_name"`
}

type LocalAuthConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type OIDCConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	Issuer         string        `mapstructure:"issuer"`
	ClientID       string        `mapstructure:"client_id"`
	ClientSecret   string        `mapstructure:"client_secret"`
	RedirectURL    string        `mapstructure:"redirect_url"`
	Scopes         []string      `mapstructure:"scopes"`
	AllowedDomains []string      `mapstructure:"allowed_domains"`
	HTTPTimeout    time.Duration `mapstructure:"http_timeout"`
	RolesClaim     string        `mapstructure:"roles_claim"`
	AllowedRoles   []string      `mapstructure:"allowed_roles"`
	AdminRoles     []string      `mapstructure:"admin_roles"`
}

type RateLimitConfig struct {
	DefaultTokensPerMinute        int `mapstructure:"default_tokens_per_minute"`
	DefaultRequestsPerMinute      int `mapstructure:"default_requests_per_minute"`
	DefaultParallelRequestsKey    int `mapstructure:"default_parallel_requests_key"`
	DefaultParallelRequestsTenant int `mapstructure:"default_parallel_requests_tenant"`
}

type BudgetConfig struct {
	DefaultUSD           float64           `mapstructure:"default_usd"`
	WarningThresholdPerc float64           `mapstructure:"warning_threshold_perc"`
	RefreshSchedule      string            `mapstructure:"refresh_schedule"`
	Alert                BudgetAlertConfig `mapstructure:"alert"`
}

type BudgetAlertConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Emails   []string      `mapstructure:"emails"`
	Webhooks []string      `mapstructure:"webhooks"`
	Cooldown time.Duration `mapstructure:"cooldown"`
	SMTP     SMTPConfig    `mapstructure:"smtp"`
	Webhook  WebhookConfig `mapstructure:"webhook"`
}

type SMTPConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	From           string        `mapstructure:"from"`
	UseTLS         bool          `mapstructure:"use_tls"`
	SkipTLSVerify  bool          `mapstructure:"skip_tls_verify"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type WebhookConfig struct {
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

type ReportingConfig struct {
	Timezone string `mapstructure:"timezone"`
}

type ProviderConfig struct {
	OpenAIKey           string `mapstructure:"openai_key"`
	AnthropicKey        string `mapstructure:"anthropic_key"`
	AzureOpenAIKey      string `mapstructure:"azure_openai_key"`
	AzureOpenAIEndpoint string `mapstructure:"azure_openai_endpoint"`
	AzureOpenAIVersion  string `mapstructure:"azure_openai_version"`
	AWSAccessKeyID      string `mapstructure:"aws_access_key_id"`
	AWSSecretAccessKey  string `mapstructure:"aws_secret_access_key"`
	AWSRegion           string `mapstructure:"aws_region"`
	GCPProjectID        string `mapstructure:"gcp_project_id"`
	GCPJSONCredentials  string `mapstructure:"gcp_json_credentials"`
	HuggingFaceToken    string `mapstructure:"hugging_face_token"`
}

type FilesConfig struct {
	Storage       string           `mapstructure:"storage"`
	MaxSizeMB     int              `mapstructure:"max_size_mb"`
	DefaultTTL    time.Duration    `mapstructure:"default_ttl"`
	MaxTTL        time.Duration    `mapstructure:"max_ttl"`
	EncryptionKey string           `mapstructure:"encryption_key"`
	S3            FilesS3Config    `mapstructure:"s3"`
	Local         FilesLocalConfig `mapstructure:"local"`
}

type FilesS3Config struct {
	Bucket       string `mapstructure:"bucket"`
	Prefix       string `mapstructure:"prefix"`
	Region       string `mapstructure:"region"`
	Endpoint     string `mapstructure:"endpoint"`
	UsePathStyle bool   `mapstructure:"use_path_style"`
}

type FilesLocalConfig struct {
	Directory string `mapstructure:"directory"`
}

type AudioConfig struct {
	MaxUploadMB int `mapstructure:"max_upload_mb"`
}

type BatchesConfig struct {
	MaxRequests    int           `mapstructure:"max_requests"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	DefaultTTL     time.Duration `mapstructure:"default_ttl"`
	MaxTTL         time.Duration `mapstructure:"max_ttl"`
}

type ModelCatalogEntry struct {
	Alias             string            `mapstructure:"alias"`
	Provider          string            `mapstructure:"provider"`
	ProviderModel     string            `mapstructure:"provider_model"`
	ContextWindow     int32             `mapstructure:"context_window"`
	MaxOutputTokens   int32             `mapstructure:"max_output_tokens"`
	Modalities        []string          `mapstructure:"modalities"`
	SupportsTools     bool              `mapstructure:"supports_tools"`
	Enabled           *bool             `mapstructure:"enabled"`
	Deployment        string            `mapstructure:"deployment"`
	Endpoint          string            `mapstructure:"endpoint"`
	APIKey            string            `mapstructure:"api_key"`
	APIVersion        string            `mapstructure:"api_version"`
	Region            string            `mapstructure:"region"`
	Weight            int               `mapstructure:"weight"`
	Metadata          map[string]string `mapstructure:"metadata"`
	ProviderOverrides `mapstructure:",squash"`
	PriceInput        float64 `mapstructure:"price_input"`
	PriceOutput       float64 `mapstructure:"price_output"`
	Currency          string  `mapstructure:"currency"`
}

func (e ModelCatalogEntry) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

type RetentionConfig struct {
	MetadataDays  int  `mapstructure:"metadata_days"`
	ZeroRetention bool `mapstructure:"zero_retention"`
}

type ObservabilityConfig struct {
	OTLPEndpoint  string `mapstructure:"otlp_endpoint"`
	EnableOTLP    bool   `mapstructure:"enable_otlp"`
	EnableMetrics bool   `mapstructure:"enable_metrics"`
}

type HealthConfig struct {
	CheckInterval time.Duration `mapstructure:"check_interval"`
	RollingWindow int           `mapstructure:"rolling_window"`
	Cooldown      time.Duration `mapstructure:"cooldown"`
}

type BootstrapConfig struct {
	Tenants       []BootstrapTenant       `mapstructure:"tenants"`
	AdminUsers    []BootstrapAdminUser    `mapstructure:"admin_users"`
	APIKeys       []BootstrapAPIKey       `mapstructure:"api_keys"`
	Memberships   []BootstrapMembership   `mapstructure:"memberships"`
	TenantLimits  []BootstrapTenantLimit  `mapstructure:"tenant_limits"`
	TenantBudgets []BootstrapTenantBudget `mapstructure:"tenant_budgets"`
}

type BootstrapTenant struct {
	Name   string `mapstructure:"name"`
	Status string `mapstructure:"status"`
}

type BootstrapAdminUser struct {
	Email      string `mapstructure:"email"`
	Name       string `mapstructure:"name"`
	Password   string `mapstructure:"password"`
	SuperAdmin *bool  `mapstructure:"super_admin"`
}

func (u BootstrapAdminUser) IsSuperAdmin() bool {
	if u.SuperAdmin == nil {
		return true
	}
	return *u.SuperAdmin
}

type BootstrapAPIKey struct {
	Tenant    string             `mapstructure:"tenant"`
	Prefix    string             `mapstructure:"prefix"`
	Secret    string             `mapstructure:"secret"`
	Name      string             `mapstructure:"name"`
	RateLimit BootstrapRateLimit `mapstructure:"rate_limit"`
}

type BootstrapMembership struct {
	Tenant string `mapstructure:"tenant"`
	Email  string `mapstructure:"email"`
	Role   string `mapstructure:"role"`
}

type BootstrapTenantLimit struct {
	Tenant string             `mapstructure:"tenant"`
	Limits BootstrapRateLimit `mapstructure:"limits"`
}

type BootstrapTenantBudget struct {
	Tenant           string        `mapstructure:"tenant"`
	BudgetUSD        *float64      `mapstructure:"budget_usd"`
	WarningThreshold *float64      `mapstructure:"warning_threshold"`
	RefreshSchedule  string        `mapstructure:"refresh_schedule"`
	AlertEmails      []string      `mapstructure:"alert_emails"`
	AlertWebhooks    []string      `mapstructure:"alert_webhooks"`
	AlertCooldown    time.Duration `mapstructure:"alert_cooldown"`
}

type BootstrapRateLimit struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	TokensPerMinute   int `mapstructure:"tokens_per_minute"`
	ParallelRequests  int `mapstructure:"parallel_requests"`
}

// Options controls the config loader behavior.
type Options struct {
	ConfigFile string
	EnvFile    string
}

// Load returns the merged configuration sourced from YAML and environment variables.
func Load(opts Options) (*Config, error) {
	if opts.EnvFile != "" {
		_ = godotenv.Load(opts.EnvFile)
	} else {
		_ = godotenv.Load()
	}

	v := viper.New()
	setDefaults(v)

	explicitFile := false
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
		explicitFile = true
	} else {
		if cfg := os.Getenv("ROUTER_CONFIG_FILE"); cfg != "" {
			v.SetConfigFile(cfg)
			explicitFile = true
		}
	}

	if !explicitFile {
		// Allow standard lookup locations when no explicit file is provided.
		v.SetConfigName("router")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	v.SetEnvPrefix("ROUTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(timeStringToDurationHook())); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate ensures required values are set.
func (c *Config) Validate() error {
	var missing []string

	if c.Database.URL == "" {
		missing = append(missing, "ROUTER_DB_URL")
	}
	if c.Redis.URL == "" {
		missing = append(missing, "ROUTER_REDIS_URL")
	}

	if c.RateLimits.DefaultTokensPerMinute <= 0 {
		missing = append(missing, "DEFAULT_TPM")
	}
	if c.RateLimits.DefaultRequestsPerMinute <= 0 {
		missing = append(missing, "DEFAULT_RPM")
	}
	if c.RateLimits.DefaultParallelRequestsKey <= 0 {
		missing = append(missing, "DEFAULT_PARALLEL_KEY")
	}
	if c.RateLimits.DefaultParallelRequestsTenant <= 0 {
		missing = append(missing, "DEFAULT_PARALLEL_TENANT")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	if c.Budgets.DefaultUSD <= 0 {
		return fmt.Errorf("budgets.default_usd must be > 0")
	}
	if c.Budgets.WarningThresholdPerc <= 0 || c.Budgets.WarningThresholdPerc >= 1 {
		return fmt.Errorf("budgets.warning_threshold_perc must be between 0 and 1 exclusive")
	}
	c.Budgets.RefreshSchedule = NormalizeBudgetRefreshSchedule(c.Budgets.RefreshSchedule)
	c.Budgets.Alert.Emails = normalizeStringSlice(c.Budgets.Alert.Emails)
	c.Budgets.Alert.Webhooks = normalizeStringSlice(c.Budgets.Alert.Webhooks)
	if c.Budgets.Alert.Cooldown <= 0 {
		if c.Budgets.Alert.Enabled {
			return fmt.Errorf("budgets.alert.cooldown must be > 0 when alerting is enabled")
		}
		c.Budgets.Alert.Cooldown = time.Hour
	}
	if c.Budgets.Alert.Enabled && len(c.Budgets.Alert.Emails) == 0 && len(c.Budgets.Alert.Webhooks) == 0 {
		return fmt.Errorf("budgets.alert requires at least one email or webhook when enabled")
	}
	smtp := &c.Budgets.Alert.SMTP
	if strings.TrimSpace(smtp.Host) != "" {
		if smtp.Port <= 0 {
			smtp.Port = 587
		}
		if strings.TrimSpace(smtp.From) == "" {
			return fmt.Errorf("budgets.alert.smtp.from must be provided when smtp.host is set")
		}
		if smtp.ConnectTimeout <= 0 {
			smtp.ConnectTimeout = 5 * time.Second
		}
	}
	if c.Budgets.Alert.Webhook.Timeout <= 0 {
		c.Budgets.Alert.Webhook.Timeout = 5 * time.Second
	}
	if c.Budgets.Alert.Webhook.MaxRetries <= 0 {
		c.Budgets.Alert.Webhook.MaxRetries = 3
	}
	if c.Database.RunMigrations && c.Database.MigrationsDir == "" {
		return fmt.Errorf("database.migrations_dir must be provided when run_migrations is true")
	}

	reportingTZ := strings.TrimSpace(c.Reporting.Timezone)
	if reportingTZ == "" {
		reportingTZ = "UTC"
	}
	if _, err := time.LoadLocation(reportingTZ); err != nil {
		return fmt.Errorf("invalid reporting.timezone: %w", err)
	}
	c.Reporting.Timezone = reportingTZ
	if c.Database.MaxConns < 0 {
		return fmt.Errorf("database.max_conns must be >= 0")
	}
	if c.Redis.PoolSize < 0 {
		return fmt.Errorf("redis.pool_size must be >= 0")
	}

	if err := c.Files.validate(); err != nil {
		return err
	}
	if err := c.Audio.validate(); err != nil {
		return err
	}
	if err := c.Batches.validate(); err != nil {
		return err
	}

	if err := c.Admin.validate(); err != nil {
		return err
	}

	for i, entry := range c.ModelCatalog {
		if entry.Alias == "" {
			return fmt.Errorf("model_catalog[%d].alias must be provided", i)
		}
		if entry.Provider == "" {
			return fmt.Errorf("model_catalog[%d].provider must be provided", i)
		}
		if entry.ProviderModel == "" {
			return fmt.Errorf("model_catalog[%d].provider_model must be provided", i)
		}
		if entry.Deployment == "" {
			return fmt.Errorf("model_catalog[%d].deployment must be provided", i)
		}
		if entry.Weight == 0 {
			c.ModelCatalog[i].Weight = 100
		}
		if entry.PriceInput < 0 || entry.PriceOutput < 0 {
			return fmt.Errorf("model_catalog[%d] price_input and price_output must be >= 0", i)
		}
		if entry.Currency == "" {
			c.ModelCatalog[i].Currency = "USD"
		}
	}

	if err := c.Bootstrap.validate(); err != nil {
		return err
	}

	return nil
}

func (a *AdminConfig) validate() error {
	if a.Session.JWTSecret == "" {
		return fmt.Errorf("admin.session.jwt_secret must be provided")
	}
	if a.Session.AccessTokenTTL <= 0 {
		return fmt.Errorf("admin.session.access_token_ttl must be > 0")
	}
	if a.Session.RefreshTokenTTL <= 0 {
		return fmt.Errorf("admin.session.refresh_token_ttl must be > 0")
	}
	if a.Session.CookieName == "" {
		return fmt.Errorf("admin.session.cookie_name must be provided")
	}

	localEnabled := a.Local.Enabled
	oidcEnabled := a.OIDC.Enabled
	if !localEnabled && !oidcEnabled {
		return fmt.Errorf("at least one admin authentication method must be enabled (local or oidc)")
	}

	if oidcEnabled {
		if a.OIDC.Issuer == "" {
			return fmt.Errorf("admin.oidc.issuer must be provided when OIDC is enabled")
		}
		if a.OIDC.ClientID == "" {
			return fmt.Errorf("admin.oidc.client_id must be provided when OIDC is enabled")
		}
		if a.OIDC.ClientSecret == "" {
			return fmt.Errorf("admin.oidc.client_secret must be provided when OIDC is enabled")
		}
		if a.OIDC.RedirectURL == "" {
			return fmt.Errorf("admin.oidc.redirect_url must be provided when OIDC is enabled")
		}
		if a.OIDC.HTTPTimeout <= 0 {
			return fmt.Errorf("admin.oidc.http_timeout must be > 0")
		}
	}

	return nil
}

func (f *FilesConfig) validate() error {
	if f.MaxSizeMB <= 0 {
		return fmt.Errorf("files.max_size_mb must be > 0")
	}
	if f.DefaultTTL <= 0 {
		f.DefaultTTL = 168 * time.Hour
	}
	if f.MaxTTL <= 0 {
		f.MaxTTL = 720 * time.Hour
	}
	if f.DefaultTTL > f.MaxTTL {
		return fmt.Errorf("files.default_ttl cannot exceed files.max_ttl")
	}
	if strings.TrimSpace(f.Storage) == "" {
		f.Storage = "local"
	}
	return nil
}

func (a *AudioConfig) validate() error {
	if a.MaxUploadMB <= 0 {
		a.MaxUploadMB = 50
	}
	return nil
}

func (b *BatchesConfig) validate() error {
	if b.MaxRequests <= 0 {
		return fmt.Errorf("batches.max_requests must be > 0")
	}
	if b.MaxConcurrency <= 0 {
		return fmt.Errorf("batches.max_concurrency must be > 0")
	}
	if b.DefaultTTL <= 0 {
		b.DefaultTTL = 168 * time.Hour
	}
	if b.MaxTTL <= 0 {
		b.MaxTTL = 720 * time.Hour
	}
	if b.DefaultTTL > b.MaxTTL {
		return fmt.Errorf("batches.default_ttl cannot exceed batches.max_ttl")
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.listen_addr", ":8080")
	v.SetDefault("server.body_limit_mb", 20)
	v.SetDefault("server.sync_timeout", "300s")
	v.SetDefault("server.stream_idle_timeout", "30s")
	v.SetDefault("server.stream_max_duration", "300s")
	v.SetDefault("server.provider_timeout", "280s")
	v.SetDefault("server.read_header_timeout", "5s")
	v.SetDefault("server.graceful_shutdown_delay", "5s")

	v.SetDefault("rate_limits.default_tokens_per_minute", 1_000_000)
	v.SetDefault("rate_limits.default_requests_per_minute", 1_000)
	v.SetDefault("rate_limits.default_parallel_requests_key", 10)
	v.SetDefault("rate_limits.default_parallel_requests_tenant", 100)

	v.SetDefault("budgets.default_usd", 100.0)
	v.SetDefault("budgets.warning_threshold_perc", 0.8)
	v.SetDefault("budgets.refresh_schedule", "calendar_month")
	v.SetDefault("budgets.alert.enabled", true)
	v.SetDefault("budgets.alert.emails", []string{})
	v.SetDefault("budgets.alert.webhooks", []string{})
	v.SetDefault("budgets.alert.cooldown", "1h")
	v.SetDefault("budgets.alert.smtp.port", 587)
	v.SetDefault("budgets.alert.smtp.use_tls", true)
	v.SetDefault("budgets.alert.smtp.skip_tls_verify", false)
	v.SetDefault("budgets.alert.smtp.connect_timeout", "5s")
	v.SetDefault("budgets.alert.webhook.timeout", "5s")
	v.SetDefault("budgets.alert.webhook.max_retries", 3)

	v.SetDefault("retention.metadata_days", 30)
	v.SetDefault("retention.zero_retention", false)

	v.SetDefault("observability.enable_otlp", true)
	v.SetDefault("observability.enable_metrics", true)
	v.SetDefault("observability.otlp_endpoint", "http://localhost:4317")

	v.SetDefault("reporting.timezone", "UTC")

	v.SetDefault("health.check_interval", "60s")
	v.SetDefault("health.rolling_window", 5)
	v.SetDefault("health.cooldown", "5m")

	v.SetDefault("database.run_migrations", true)
	v.SetDefault("database.migrations_dir", "./migrations")
	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.max_conn_idle_time", "10m")
	v.SetDefault("database.max_conn_lifetime", "1h")

	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)

	v.SetDefault("files.storage", "local")
	v.SetDefault("files.max_size_mb", 200)
	v.SetDefault("files.default_ttl", "168h")
	v.SetDefault("files.max_ttl", "720h")
	v.SetDefault("files.local.directory", "./data/files")

	v.SetDefault("audio.max_upload_mb", 50)

	v.SetDefault("batches.max_requests", 5000)
	v.SetDefault("batches.max_concurrency", 50)
	v.SetDefault("batches.default_ttl", "168h")
	v.SetDefault("batches.max_ttl", "720h")

	v.SetDefault("admin.session.access_token_ttl", "15m")
	v.SetDefault("admin.session.refresh_token_ttl", "24h")
	v.SetDefault("admin.session.cookie_name", "og_admin_session")
	v.SetDefault("admin.local.enabled", true)
	v.SetDefault("admin.oidc.enabled", false)
	v.SetDefault("admin.oidc.scopes", []string{"openid", "email", "profile"})
	v.SetDefault("admin.oidc.http_timeout", "5s")
	v.SetDefault("providers.azure_openai_version", "2024-07-01-preview")
}

func (b *BootstrapConfig) validate() error {
	for i, tenant := range b.Tenants {
		if strings.TrimSpace(tenant.Name) == "" {
			return fmt.Errorf("bootstrap.tenants[%d].name must be provided", i)
		}
		if tenant.Status != "" {
			switch strings.ToLower(tenant.Status) {
			case "active", "suspended":
			default:
				return fmt.Errorf("bootstrap.tenants[%d].status must be active or suspended", i)
			}
		}
	}
	for i, user := range b.AdminUsers {
		if strings.TrimSpace(user.Email) == "" {
			return fmt.Errorf("bootstrap.admin_users[%d].email must be provided", i)
		}
		if strings.TrimSpace(user.Name) == "" {
			return fmt.Errorf("bootstrap.admin_users[%d].name must be provided", i)
		}
		if strings.TrimSpace(user.Password) == "" {
			return fmt.Errorf("bootstrap.admin_users[%d].password must be provided", i)
		}
	}
	for i, key := range b.APIKeys {
		if strings.TrimSpace(key.Tenant) == "" {
			return fmt.Errorf("bootstrap.api_keys[%d].tenant must be provided", i)
		}
		if strings.TrimSpace(key.Prefix) == "" {
			return fmt.Errorf("bootstrap.api_keys[%d].prefix must be provided", i)
		}
		if strings.TrimSpace(key.Secret) == "" {
			return fmt.Errorf("bootstrap.api_keys[%d].secret must be provided", i)
		}
		if strings.TrimSpace(key.Name) == "" {
			return fmt.Errorf("bootstrap.api_keys[%d].name must be provided", i)
		}
		if err := validateRateLimit(key.RateLimit); err != nil {
			return fmt.Errorf("bootstrap.api_keys[%d].rate_limit: %w", i, err)
		}
	}
	for i, member := range b.Memberships {
		if strings.TrimSpace(member.Tenant) == "" {
			return fmt.Errorf("bootstrap.memberships[%d].tenant must be provided", i)
		}
		if strings.TrimSpace(member.Email) == "" {
			return fmt.Errorf("bootstrap.memberships[%d].email must be provided", i)
		}
		if _, ok := rbac.ParseRole(member.Role); !ok {
			return fmt.Errorf("bootstrap.memberships[%d].role must be owner/admin/viewer/user", i)
		}
	}
	for i, limit := range b.TenantLimits {
		if strings.TrimSpace(limit.Tenant) == "" {
			return fmt.Errorf("bootstrap.tenant_limits[%d].tenant must be provided", i)
		}
		if err := validateRateLimit(limit.Limits); err != nil {
			return fmt.Errorf("bootstrap.tenant_limits[%d].limits: %w", i, err)
		}
	}
	for i := range b.TenantBudgets {
		budget := &b.TenantBudgets[i]
		if strings.TrimSpace(budget.Tenant) == "" {
			return fmt.Errorf("bootstrap.tenant_budgets[%d].tenant must be provided", i)
		}
		if budget.BudgetUSD != nil && *budget.BudgetUSD < 0 {
			return fmt.Errorf("bootstrap.tenant_budgets[%d].budget_usd must be >= 0", i)
		}
		if budget.WarningThreshold != nil {
			if *budget.WarningThreshold <= 0 || *budget.WarningThreshold > 1 {
				return fmt.Errorf("bootstrap.tenant_budgets[%d].warning_threshold must be between 0 and 1", i)
			}
		}
		if strings.TrimSpace(budget.RefreshSchedule) != "" {
			budget.RefreshSchedule = NormalizeBudgetRefreshSchedule(budget.RefreshSchedule)
		}
		budget.AlertEmails = normalizeStringSlice(budget.AlertEmails)
		budget.AlertWebhooks = normalizeStringSlice(budget.AlertWebhooks)
		if budget.AlertCooldown < 0 {
			return fmt.Errorf("bootstrap.tenant_budgets[%d].alert_cooldown must be >= 0", i)
		}
	}
	return nil
}

func validateRateLimit(limit BootstrapRateLimit) error {
	if limit.RequestsPerMinute < 0 {
		return fmt.Errorf("requests_per_minute must be >= 0")
	}
	if limit.TokensPerMinute < 0 {
		return fmt.Errorf("tokens_per_minute must be >= 0")
	}
	if limit.ParallelRequests < 0 {
		return fmt.Errorf("parallel_requests must be >= 0")
	}
	return nil
}

func NormalizeBudgetRefreshSchedule(schedule string) string {
	schedule = strings.ToLower(strings.TrimSpace(schedule))
	if schedule == "" {
		return "calendar_month"
	}
	switch schedule {
	case "calendar_month", "weekly":
		return schedule
	default:
		if days, ok := BudgetRollingWindowDays(schedule); ok && days > 0 {
			return fmt.Sprintf("rolling_%dd", days)
		}
	}
	return "calendar_month"
}

func BudgetRollingWindowDays(schedule string) (int, bool) {
	schedule = strings.ToLower(strings.TrimSpace(schedule))
	if !strings.HasPrefix(schedule, "rolling_") {
		return 0, false
	}
	rest := strings.TrimPrefix(schedule, "rolling_")
	rest = strings.TrimSuffix(rest, "d")
	if rest == "" {
		rest = "30"
	}
	days, err := strconv.Atoi(rest)
	if err != nil || days <= 0 {
		return 0, false
	}
	return days, true
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	clean := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}

func timeStringToDurationHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != reflect.TypeOf(time.Duration(0)) {
			return data, nil
		}

		switch v := data.(type) {
		case time.Duration:
			return v, nil
		case string:
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, err
			}
			return d, nil
		default:
			return nil, fmt.Errorf("cannot decode %T into time.Duration", data)
		}
	}
}

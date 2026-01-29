package config

import "time"

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	ListenAddr            string        `mapstructure:"listen_addr"`
	BodyLimitMB           int           `mapstructure:"body_limit_mb"`
	SyncTimeout           time.Duration `mapstructure:"sync_timeout"`
	StreamIdleTimeout     time.Duration `mapstructure:"stream_idle_timeout"`
	StreamMaxDuration     time.Duration `mapstructure:"stream_max_duration"`
	ProviderTimeout       time.Duration `mapstructure:"provider_timeout"`
	ReadHeaderTimeout     time.Duration `mapstructure:"read_header_timeout"`
	GracefulShutdownDelay time.Duration `mapstructure:"graceful_shutdown_delay"`

	// CORS configures Cross-Origin Resource Sharing.
	CORS CORSConfig `mapstructure:"cors"`

	// Security configures HTTP security headers and protections.
	Security SecurityConfig `mapstructure:"security"`

	// AuthRateLimit configures brute-force protection on auth endpoints.
	AuthRateLimit AuthRateLimitConfig `mapstructure:"auth_rate_limit"`
}

// CORSConfig holds CORS middleware settings.
type CORSConfig struct {
	// Enabled enables CORS middleware. Defaults to true.
	Enabled bool `mapstructure:"enabled"`

	// AllowOrigins is a comma-separated list of allowed origins. Use "*" for all.
	AllowOrigins string `mapstructure:"allow_origins"`

	// AllowMethods is a comma-separated list of allowed HTTP methods.
	AllowMethods string `mapstructure:"allow_methods"`

	// AllowHeaders is a comma-separated list of allowed request headers.
	AllowHeaders string `mapstructure:"allow_headers"`

	// AllowCredentials indicates whether the response can include credentials.
	AllowCredentials bool `mapstructure:"allow_credentials"`

	// MaxAge is the max duration (in seconds) that preflight results can be cached.
	MaxAge int `mapstructure:"max_age"`
}

// SecurityConfig holds HTTP security header settings.
type SecurityConfig struct {
	// HSTS enables Strict-Transport-Security header.
	HSTS bool `mapstructure:"hsts"`

	// HSTSMaxAge is the max-age value in seconds for HSTS. Defaults to 31536000 (1 year).
	HSTSMaxAge int `mapstructure:"hsts_max_age"`

	// FrameOptions sets X-Frame-Options header. Common values: DENY, SAMEORIGIN.
	FrameOptions string `mapstructure:"frame_options"`

	// ContentTypeNosniff enables X-Content-Type-Options: nosniff header.
	ContentTypeNosniff bool `mapstructure:"content_type_nosniff"`

	// CSP sets the Content-Security-Policy header value. Empty disables.
	CSP string `mapstructure:"csp"`
}

// AuthRateLimitConfig configures rate limiting on authentication endpoints.
type AuthRateLimitConfig struct {
	// Enabled enables rate limiting on auth endpoints. Defaults to true.
	Enabled bool `mapstructure:"enabled"`

	// MaxAttempts is the maximum number of attempts per window. Defaults to 10.
	MaxAttempts int `mapstructure:"max_attempts"`

	// Window is the sliding window duration. Defaults to 15m.
	Window time.Duration `mapstructure:"window"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	RunMigrations   bool          `mapstructure:"run_migrations"`
	MigrationsDir   string        `mapstructure:"migrations_dir"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MinConns        int32         `mapstructure:"min_conns"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL      string `mapstructure:"url"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// PublicConfig holds public-facing configuration.
type PublicConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// ReportingConfig holds reporting settings.
type ReportingConfig struct {
	Timezone string `mapstructure:"timezone"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error).
	Level string `mapstructure:"level"`

	// Format is the output format (json, text).
	Format string `mapstructure:"format"`

	// AddSource adds source file information to log entries.
	AddSource bool `mapstructure:"add_source"`

	// WideEvent configures wide event (canonical log line) logging.
	WideEvent WideEventConfig `mapstructure:"wide_event"`
}

// WideEventConfig holds wide event logging settings.
type WideEventConfig struct {
	// Enabled enables wide event logging (single comprehensive log per request).
	Enabled bool `mapstructure:"enabled"`

	// IncludeUserAgent includes the User-Agent header in log events.
	IncludeUserAgent bool `mapstructure:"include_user_agent"`

	// IncludeHeaders lists additional headers to include in log events.
	IncludeHeaders []string `mapstructure:"include_headers"`
}

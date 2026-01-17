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

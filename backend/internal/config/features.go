package config

import "time"

// RateLimitConfig holds rate limiting defaults.
type RateLimitConfig struct {
	DefaultTokensPerMinute        int `mapstructure:"default_tokens_per_minute"`
	DefaultRequestsPerMinute      int `mapstructure:"default_requests_per_minute"`
	DefaultParallelRequestsKey    int `mapstructure:"default_parallel_requests_key"`
	DefaultParallelRequestsTenant int `mapstructure:"default_parallel_requests_tenant"`
}

// BudgetConfig holds budget and alerting settings.
type BudgetConfig struct {
	DefaultUSD           float64           `mapstructure:"default_usd"`
	WarningThresholdPerc float64           `mapstructure:"warning_threshold_perc"`
	RefreshSchedule      string            `mapstructure:"refresh_schedule"`
	Alert                BudgetAlertConfig `mapstructure:"alert"`
}

// BudgetAlertConfig holds budget alert settings.
type BudgetAlertConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Emails   []string      `mapstructure:"emails"`
	Webhooks []string      `mapstructure:"webhooks"`
	Cooldown time.Duration `mapstructure:"cooldown"`
	SMTP     SMTPConfig    `mapstructure:"smtp"`
	Webhook  WebhookConfig `mapstructure:"webhook"`
}

// SMTPConfig holds SMTP settings for email alerts.
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

// WebhookConfig holds webhook settings for alerts.
type WebhookConfig struct {
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	Secret     string        `mapstructure:"secret"`
}

// FilesConfig holds file storage settings.
type FilesConfig struct {
	Storage        string           `mapstructure:"storage"`
	MaxSizeMB      int              `mapstructure:"max_size_mb"`
	DefaultTTL     time.Duration    `mapstructure:"default_ttl"`
	MaxTTL         time.Duration    `mapstructure:"max_ttl"`
	EncryptionKey  string           `mapstructure:"encryption_key"`
	SweepInterval  time.Duration    `mapstructure:"sweep_interval"`
	SweepBatchSize int              `mapstructure:"sweep_batch_size"`
	S3             FilesS3Config    `mapstructure:"s3"`
	Local          FilesLocalConfig `mapstructure:"local"`
}

// FilesS3Config holds S3 storage settings.
type FilesS3Config struct {
	Bucket       string `mapstructure:"bucket"`
	Prefix       string `mapstructure:"prefix"`
	Region       string `mapstructure:"region"`
	Endpoint     string `mapstructure:"endpoint"`
	UsePathStyle bool   `mapstructure:"use_path_style"`
}

// FilesLocalConfig holds local file storage settings.
type FilesLocalConfig struct {
	Directory string `mapstructure:"directory"`
}

// AudioConfig holds audio processing settings.
type AudioConfig struct {
	MaxUploadMB int `mapstructure:"max_upload_mb"`
}

// BatchesConfig holds batch processing settings.
type BatchesConfig struct {
	MaxRequests    int           `mapstructure:"max_requests"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	DefaultTTL     time.Duration `mapstructure:"default_ttl"`
	MaxTTL         time.Duration `mapstructure:"max_ttl"`
}

// RetentionConfig holds data retention settings.
type RetentionConfig struct {
	MetadataDays  int  `mapstructure:"metadata_days"`
	ZeroRetention bool `mapstructure:"zero_retention"`
}

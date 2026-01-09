package config

import "time"

// ObservabilityConfig holds OTEL and metrics settings.
type ObservabilityConfig struct {
	OTLPEndpoint  string `mapstructure:"otlp_endpoint"`
	EnableOTLP    bool   `mapstructure:"enable_otlp"`
	EnableMetrics bool   `mapstructure:"enable_metrics"`
}

// TelemetryConfig holds telemetry settings.
type TelemetryConfig struct {
	Provider ProviderTelemetryConfig `mapstructure:"provider"`
}

// ProviderTelemetryConfig holds provider-level telemetry settings.
type ProviderTelemetryConfig struct {
	Enabled                bool                           `mapstructure:"enabled"`
	EvaluationInterval     time.Duration                  `mapstructure:"evaluation_interval"`
	WindowSize             time.Duration                  `mapstructure:"window_size"`
	IncidentRetentionDays  int                            `mapstructure:"incident_retention_days"`
	DownweightWhenDegraded bool                           `mapstructure:"downweight_when_degraded"`
	Defaults               ProviderSLIDefaults            `mapstructure:"defaults"`
	Overrides              map[string]ProviderSLIDefaults `mapstructure:"overrides"`
}

// ProviderSLIDefaults holds default SLI thresholds for providers.
type ProviderSLIDefaults struct {
	LatencyP95Ms         int     `mapstructure:"latency_p95_ms"`
	ErrorRateThreshold   float64 `mapstructure:"error_rate_threshold"`
	TimeoutRateThreshold float64 `mapstructure:"timeout_rate_threshold"`
	MinSamples           int     `mapstructure:"min_samples"`
}

// HealthConfig holds health check settings.
type HealthConfig struct {
	CheckInterval time.Duration `mapstructure:"check_interval"`
	RollingWindow int           `mapstructure:"rolling_window"`
	Cooldown      time.Duration `mapstructure:"cooldown"`
}

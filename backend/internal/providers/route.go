package providers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// Route represents a single provider deployment that can serve a public alias.
type Route struct {
	Alias                 string
	Provider              string
	Model                 string
	Weight                int
	Metadata              map[string]string
	Chat                  ChatCompletions
	ChatStream            ChatStreaming
	Embedding             EmbeddingsProvider
	Moderations           ModerationsProvider
	Image                 ImagesProvider
	AudioTranscribe       AudioTranscriber
	AudioTranslate        AudioTranslator
	AudioTranscribeStream AudioTranscriptionStreaming
	TextToSpeech          TextToSpeech
	TextToSpeechStream    TextToSpeechStreaming
	Models                ModelLister
	Health                HealthChecker
	Retry                 RetryConfig
	Tokenizer             string
}

// HealthChecker defines provider health probes consumed by the monitor.
type HealthChecker interface {
	Check(ctx context.Context) error
}

// HealthCheckerFunc converts a function into a checker.
type HealthCheckerFunc func(ctx context.Context) error

// Check implements HealthChecker.
func (f HealthCheckerFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// WrapHealth converts a raw function into a HealthChecker, returning nil when fn is nil.
func WrapHealth(fn func(ctx context.Context) error) HealthChecker {
	if fn == nil {
		return nil
	}
	return HealthCheckerFunc(fn)
}

// RetryConfig describes per-route retry behavior.
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	BackoffMultiplier float64
}

func (r RetryConfig) normalize() RetryConfig {
	result := r
	if result.MaxAttempts <= 0 {
		result.MaxAttempts = 1
	}
	if result.InitialBackoff < 0 {
		result.InitialBackoff = 0
	}
	if result.BackoffMultiplier <= 0 {
		result.BackoffMultiplier = 1
	}
	return result
}

func mergeRetry(defaults RetryConfig, override *config.ProviderRetryConfig, metadata map[string]string) RetryConfig {
	result := defaults
	if override != nil {
		if override.MaxAttempts > 0 {
			result.MaxAttempts = override.MaxAttempts
		}
		if override.InitialBackoff > 0 {
			result.InitialBackoff = override.InitialBackoff
		}
		if override.BackoffMultiplier > 0 {
			result.BackoffMultiplier = override.BackoffMultiplier
		}
	}
	if v := metadataValue(metadata, "retry_max_attempts"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			result.MaxAttempts = parsed
		}
	}
	if v := metadataValue(metadata, "retry_initial_backoff_ms"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			result.InitialBackoff = time.Duration(parsed) * time.Millisecond
		}
	}
	if v := metadataValue(metadata, "retry_backoff_multiplier"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			result.BackoffMultiplier = parsed
		}
	}
	return result.normalize()
}

func selectTokenizer(defaultValue string, override string, metadata map[string]string) string {
	if token := strings.TrimSpace(override); token != "" {
		return token
	}
	if token := metadataValue(metadata, "tokenizer"); token != "" {
		return token
	}
	return defaultValue
}

func metadataValue(md map[string]string, key string) string {
	if md == nil {
		return ""
	}
	return strings.TrimSpace(md[key])
}

// ResolveDeployment extracts deployment identifier from route metadata.
func (r Route) ResolveDeployment() string {
	if r.Metadata != nil {
		if dep := r.Metadata["deployment"]; dep != "" {
			return dep
		}
	}
	return r.Model
}

// ToModel converts route metadata back to a models.Model struct for APIs.
func (r Route) ToModel() models.Model {
	return models.Model{
		Alias:           r.Alias,
		Provider:        r.Provider,
		ProviderModel:   r.Model,
		ContextWindow:   0,
		MaxOutputTokens: 0,
	}
}

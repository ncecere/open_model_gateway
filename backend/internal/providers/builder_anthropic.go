package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/anthropic"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "anthropic",
		Description:  "Anthropic Claude API",
		Capabilities: []string{"chat", "chat_stream"},
		Descriptor: Descriptor{
			Summary: "Anthropic Claude Messages API (sync + SSE).",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.anthropic.api_key", Description: "Default Anthropic API key", Secret: true, Source: "config.providers.anthropic.api_key"},
				{Name: "providers.anthropic.base_url", Description: "Optional default base URL", Source: "config.providers.anthropic.base_url"},
				{Name: "providers.anthropic.version", Description: "Optional default API version", Source: "config.providers.anthropic.version"},
			},
			EntryFields: []Input{
				{Name: "api_key", Description: "Override API key", Secret: true, Source: "catalog.api_key"},
				{Name: "endpoint", Description: "Custom base URL (e.g. self-hosted proxy)", Source: "catalog.endpoint"},
				{Name: "anthropic_version", Description: "Claude API version", Source: "catalog.metadata.anthropic_version"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries two times; Anthropic 529/RateLimit headers dictate additional waits.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Uses the Claude /models endpoint via SDK for health.",
		},
		Builder: buildAnthropicRoute,
	})
}

func buildAnthropicRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.Anthropic
	cfgAnthropic := cfg.Providers.Anthropic

	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		case md["api_key"] != "":
			apiKey = strings.TrimSpace(md["api_key"])
		case strings.TrimSpace(cfgAnthropic.APIKey) != "":
			apiKey = strings.TrimSpace(cfgAnthropic.APIKey)
		default:
			apiKey = strings.TrimSpace(cfg.Providers.AnthropicKey)
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("anthropic provider requires api key")
	}

	baseURL := strings.TrimSpace(entry.Endpoint)
	if override != nil && strings.TrimSpace(override.BaseURL) != "" {
		baseURL = strings.TrimSpace(override.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(md["anthropic_base_url"])
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfgAnthropic.BaseURL)
	}

	version := strings.TrimSpace(md["anthropic_version"])
	if override != nil && strings.TrimSpace(override.Version) != "" {
		version = strings.TrimSpace(override.Version)
	}
	if version == "" && strings.TrimSpace(cfgAnthropic.Version) != "" {
		version = strings.TrimSpace(cfgAnthropic.Version)
	}

	defaultMax := entry.MaxOutputTokens
	opts := anthropic.Options{
		APIKey:           apiKey,
		BaseURL:          baseURL,
		Version:          version,
		DefaultMaxTokens: defaultMax,
	}

	adapter, err := anthropic.New(opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	route := Route{
		Alias:      entry.Alias,
		Provider:   entry.Provider,
		Model:      entry.ProviderModel,
		Weight:     weight,
		Metadata:   md,
		Chat:       adapter,
		ChatStream: adapter,
		Health:     WrapHealth(adapter.HealthCheck),
	}
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("anthropic", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

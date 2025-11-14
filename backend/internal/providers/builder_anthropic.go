package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/anthropic"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "anthropic",
		Description:  "Anthropic Claude API",
		Capabilities: []string{"chat", "chat_stream"},
		Builder:      buildAnthropicRoute,
	})
}

func buildAnthropicRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.Anthropic

	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		case md["api_key"] != "":
			apiKey = strings.TrimSpace(md["api_key"])
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

	version := strings.TrimSpace(md["anthropic_version"])
	if override != nil && strings.TrimSpace(override.Version) != "" {
		version = strings.TrimSpace(override.Version)
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
		Health:     adapter.HealthCheck,
	}
	return route, nil
}

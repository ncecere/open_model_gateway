package providers

import (
	"context"
	"fmt"
	"strings"

	groq "github.com/ncecere/open_model_gateway/backend/internal/adapters/groq"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "groq",
		Description:  "Groq OpenAI-compatible API (chat + streaming)",
		Capabilities: []string{"chat", "chat_stream"},
		Builder:      buildGroqRoute,
	})
}

func buildGroqRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.Groq

	apiKey := strings.TrimSpace(entry.APIKey)
	switch {
	case apiKey != "":
	case override != nil && strings.TrimSpace(override.APIKey) != "":
		apiKey = strings.TrimSpace(override.APIKey)
	case strings.TrimSpace(md["groq_api_key"]) != "":
		apiKey = strings.TrimSpace(md["groq_api_key"])
	default:
		apiKey = strings.TrimSpace(cfg.Providers.Groq.APIKey)
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("groq provider requires api key")
	}

	baseURL := strings.TrimSpace(entry.Endpoint)
	switch {
	case override != nil && strings.TrimSpace(override.BaseURL) != "":
		baseURL = strings.TrimSpace(override.BaseURL)
	case strings.TrimSpace(md["base_url"]) != "":
		baseURL = strings.TrimSpace(md["base_url"])
	case baseURL == "":
		baseURL = strings.TrimSpace(cfg.Providers.Groq.BaseURL)
	}

	region := ""
	switch {
	case override != nil && strings.TrimSpace(override.Region) != "":
		region = strings.TrimSpace(override.Region)
	case strings.TrimSpace(md["groq_region"]) != "":
		region = strings.TrimSpace(md["groq_region"])
	default:
		region = strings.TrimSpace(cfg.Providers.Groq.Region)
	}

	opts := groq.Options{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Region:  region,
	}
	adapter, err := groq.New(opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}
	if opts.BaseURL != "" {
		md["base_url"] = opts.BaseURL
	}
	if region != "" {
		md["groq_region"] = region
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

package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	groq "github.com/ncecere/open_model_gateway/backend/internal/adapters/groq"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "groq",
		Description:  "Groq OpenAI-compatible API (chat + streaming)",
		Capabilities: []string{"chat", "chat_stream"},
		Descriptor: Descriptor{
			Summary: "Groq-hosted OpenAI-compatible endpoint optimized for low-latency chat.",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.groq.api_key", Description: "Default Groq API key", Secret: true, Source: "config.providers.groq.api_key"},
				{Name: "providers.groq.base_url", Description: "Optional base URL override", Source: "config.providers.groq.base_url"},
				{Name: "providers.groq.region", Description: "Preferred region", Source: "config.providers.groq.region"},
			},
			EntryFields: []Input{
				{Name: "endpoint", Description: "Override base URL", Source: "catalog.endpoint"},
				{Name: "api_key", Description: "Override API key", Secret: true, Source: "catalog.api_key"},
				{Name: "groq_region", Description: "Override region", Source: "catalog.metadata.groq_region"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries Groq 429/500 responses twice.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Uses Groq's /v1/models endpoint for health checks via SDK.",
		},
		Builder: buildGroqRoute,
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
		Health:     WrapHealth(adapter.HealthCheck),
	}
	route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

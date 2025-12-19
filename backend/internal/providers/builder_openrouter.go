package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	openrouter "github.com/ncecere/open_model_gateway/backend/internal/adapters/openrouter"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "openrouter",
		Description:  "OpenRouter meta-routing API (chat, streaming, embeddings, model discovery)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "models"},
		Descriptor: Descriptor{
			Summary: "OpenRouter aggregator routing requests to multiple upstream LLM providers.",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.openrouter.api_key", Description: "Default OpenRouter API key", Secret: true, Source: "config.providers.openrouter.api_key"},
				{Name: "providers.openrouter.base_url", Description: "Optional private base URL", Source: "config.providers.openrouter.base_url"},
				{Name: "providers.openrouter.referer", Description: "Referer header for attribution", Source: "config.providers.openrouter.referer"},
				{Name: "providers.openrouter.app_name", Description: "App name metadata sent to OpenRouter", Source: "config.providers.openrouter.app_name"},
			},
			EntryFields: []Input{
				{Name: "api_key", Description: "Override API key", Secret: true, Source: "catalog.api_key"},
				{Name: "endpoint", Description: "Override base URL", Source: "catalog.endpoint"},
				{Name: "openrouter_referer", Description: "Per-entry referer header", Source: "catalog.metadata.openrouter_referer"},
				{Name: "openrouter_app_name", Description: "Per-entry app name", Source: "catalog.metadata.openrouter_app_name"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries OpenRouter 429/5xx responses twice.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Uses OpenRouter models listing for health.",
		},
		Builder: buildOpenRouterRoute,
	})
}

func buildOpenRouterRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.OpenRouter

	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" && override != nil && strings.TrimSpace(override.APIKey) != "" {
		apiKey = strings.TrimSpace(override.APIKey)
	}
	if apiKey == "" {
		if val := strings.TrimSpace(md["openrouter_api_key"]); val != "" {
			apiKey = val
		} else if val := strings.TrimSpace(cfg.Providers.OpenRouter.APIKey); val != "" {
			apiKey = val
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("openrouter provider requires api key")
	}

	baseURL := strings.TrimSpace(entry.Endpoint)
	if override != nil && strings.TrimSpace(override.BaseURL) != "" {
		baseURL = strings.TrimSpace(override.BaseURL)
	}
	if baseURL == "" {
		if val := strings.TrimSpace(md["base_url"]); val != "" {
			baseURL = val
		} else if val := strings.TrimSpace(cfg.Providers.OpenRouter.BaseURL); val != "" {
			baseURL = val
		}
	}

	referer := ""
	if override != nil && strings.TrimSpace(override.Referer) != "" {
		referer = strings.TrimSpace(override.Referer)
	} else if val := strings.TrimSpace(md["openrouter_referer"]); val != "" {
		referer = val
	} else if val := strings.TrimSpace(cfg.Providers.OpenRouter.Referer); val != "" {
		referer = val
	}

	appName := ""
	if override != nil && strings.TrimSpace(override.AppName) != "" {
		appName = strings.TrimSpace(override.AppName)
	} else if val := strings.TrimSpace(md["openrouter_app_name"]); val != "" {
		appName = val
	} else if val := strings.TrimSpace(cfg.Providers.OpenRouter.AppName); val != "" {
		appName = val
	}

	opts := openrouter.Options{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Referer: referer,
		AppName: appName,
	}
	adapter, err := openrouter.New(opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}
	if baseURL != "" {
		md["base_url"] = strings.TrimSpace(baseURL)
	}
	if referer != "" {
		md["openrouter_referer"] = referer
	}
	if appName != "" {
		md["openrouter_app_name"] = appName
	}

	route := Route{
		Alias:      entry.Alias,
		Provider:   entry.Provider,
		Model:      entry.ProviderModel,
		Weight:     weight,
		Metadata:   md,
		Chat:       adapter,
		ChatStream: adapter,
		Embedding:  adapter,
		Models:     adapter,
		Health:     WrapHealth(adapter.HealthCheck),
	}
	route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

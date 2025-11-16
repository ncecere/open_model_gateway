package providers

import (
	"context"
	"fmt"
	"strings"

	openrouter "github.com/ncecere/open_model_gateway/backend/internal/adapters/openrouter"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "openrouter",
		Description:  "OpenRouter meta-routing API (chat, streaming, embeddings, model discovery)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "models"},
		Builder:      buildOpenRouterRoute,
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
		Health:     adapter.HealthCheck,
	}
	return route, nil
}

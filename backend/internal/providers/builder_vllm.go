package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	native "github.com/ncecere/open_model_gateway/backend/internal/adapters/openai"
	vllm "github.com/ncecere/open_model_gateway/backend/internal/adapters/vllm"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

const (
	vllmModeOpenAI = "openai"
	vllmModeTGI    = "tgi"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "vllm",
		Description:  "vLLM / TGI adapter (OpenAI-compatible or /generate)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "models"},
		Descriptor: Descriptor{
			Summary: "Targets vLLM's OpenAI-compatible API or Hugging Face TGI endpoints.",
			Auth:    []string{"api_key", "auth_header"},
			ConfigInputs: []Input{
				{Name: "providers.vllm.base_url", Description: "Default base URL for vLLM/TGI (e.g., http://localhost:8000/v1)", Source: "config.providers.vllm.base_url"},
				{Name: "providers.vllm.mode", Description: "Routing mode: openai or tgi", Source: "config.providers.vllm.mode"},
				{Name: "providers.vllm.api_key", Description: "Optional shared API key", Secret: true, Source: "config.providers.vllm.api_key"},
				{Name: "providers.vllm.auth_header", Description: "Optional auth header name (Authorization, X-Api-Key)", Source: "config.providers.vllm.auth_header"},
			},
			EntryFields: []Input{
				{Name: "endpoint", Description: "Required base URL for the upstream vLLM/TGI service", Required: true, Source: "catalog.endpoint"},
				{Name: "api_key", Description: "Optional per-entry API key override", Secret: true, Source: "catalog.api_key"},
				{Name: "vllm_mode", Description: "Override mode per entry (openai or tgi)", Source: "catalog.metadata.vllm_mode"},
				{Name: "auth_header", Description: "Override auth header name", Source: "catalog.metadata.auth_header"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries upstream 429/5xx responses twice.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "OpenAI mode checks /models; TGI mode checks /health.",
		},
		Builder: buildVLLMRoute,
	})
}

func buildVLLMRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.VLLM
	baseCfg := cfg.Providers.VLLM

	baseURL := strings.TrimSpace(entry.Endpoint)
	if override != nil && strings.TrimSpace(override.BaseURL) != "" {
		baseURL = strings.TrimSpace(override.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(md["base_url"])
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(baseCfg.BaseURL)
	}
	if baseURL == "" {
		return Route{}, fmt.Errorf("vllm provider requires base_url (entry.endpoint, metadata.base_url, or providers.vllm.base_url)")
	}

	mode := strings.TrimSpace(md["vllm_mode"])
	if override != nil && strings.TrimSpace(override.Mode) != "" {
		mode = strings.TrimSpace(override.Mode)
	}
	if mode == "" {
		mode = strings.TrimSpace(baseCfg.Mode)
	}
	if mode == "" {
		mode = vllmModeOpenAI
	}
	mode = strings.ToLower(mode)
	switch mode {
	case vllmModeOpenAI, vllmModeTGI:
	default:
		return Route{}, fmt.Errorf("vllm provider mode must be openai or tgi")
	}

	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" && override != nil && strings.TrimSpace(override.APIKey) != "" {
		apiKey = strings.TrimSpace(override.APIKey)
	}
	if apiKey == "" {
		if val := strings.TrimSpace(md["api_key"]); val != "" {
			apiKey = val
		} else if val := strings.TrimSpace(baseCfg.APIKey); val != "" {
			apiKey = val
		}
	}

	authHeader := strings.TrimSpace(md["auth_header"])
	if override != nil && strings.TrimSpace(override.AuthHeader) != "" {
		authHeader = strings.TrimSpace(override.AuthHeader)
	}
	if authHeader == "" {
		authHeader = strings.TrimSpace(baseCfg.AuthHeader)
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	md["base_url"] = baseURL
	if mode != "" {
		md["vllm_mode"] = mode
	}
	if authHeader != "" {
		md["auth_header"] = authHeader
	}

	switch mode {
	case vllmModeTGI:
		adapter, err := vllm.NewTGI(vllm.Options{
			BaseURL:    baseURL,
			APIKey:     apiKey,
			AuthHeader: authHeader,
		})
		if err != nil {
			return Route{}, err
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
	default:
		adapter, err := native.New(native.Options{
			APIKey:     apiKey,
			BaseURL:    baseURL,
			AllowNoKey: true,
		})
		if err != nil {
			return Route{}, err
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
			Health:     wrapOpenAICompatibleHealth(adapter.HealthCheck),
		}
		route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
		route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
		route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
		return route, nil
	}
}

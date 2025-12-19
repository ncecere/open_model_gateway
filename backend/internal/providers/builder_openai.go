package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	native "github.com/ncecere/open_model_gateway/backend/internal/adapters/openai"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:        "openai",
		Description: "OpenAI native API (chat, streaming, embeddings, images, audio)",
		Capabilities: []string{
			"chat", "chat_stream", "embeddings", "images", "models",
			"audio_transcription", "audio_translation", "audio_speech", "moderations",
		},
		Descriptor: Descriptor{
			Summary: "Native OpenAI platform using the official REST API.",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.openai.api_key", Description: "Default API key when catalog entries omit one", Required: true, Secret: true, Source: "config.providers.openai.api_key"},
				{Name: "providers.openai.openai_organization", Description: "Optional organization header", Source: "config.providers.openai.openai_organization"},
				{Name: "providers.openai.base_url", Description: "Optional base URL override", Source: "config.providers.openai.base_url"},
			},
			EntryFields: []Input{
				{Name: "api_key", Description: "Override the API key for this catalog entry", Secret: true, Source: "catalog.api_key"},
				{Name: "endpoint", Description: "Override base URL (e.g. Azure/OpenAI-compatible host)", Source: "catalog.endpoint"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries 2x with 250ms backoff; provider-side 429/500 retry headers are honored when present.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Invokes the SDK health check which calls the OpenAI models endpoint.",
		},
		Builder: buildOpenAIRoute,
	})
	RegisterDefinition(Definition{
		Name:         "openai-compatible",
		Description:  "OpenAI API-compatible endpoint (custom base URL)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "images", "audio_transcription", "audio_translation", "audio_speech", "moderations"},
		Descriptor: Descriptor{
			Summary: "Targets third-party gateways that speak the OpenAI API (base URL + API key).",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.openai_compatible.base_url", Description: "Default base URL when catalog entries omit one", Source: "config.providers.openai_compatible.base_url"},
				{Name: "providers.openai_compatible.api_key", Description: "Fallback API key", Secret: true, Source: "config.providers.openai_compatible.api_key"},
				{Name: "providers.openai.api_key", Description: "Secondary fallback API key", Secret: true, Source: "config.providers.openai.api_key"},
			},
			EntryFields: []Input{
				{Name: "endpoint", Description: "Required base URL for the upstream gateway", Required: true, Source: "catalog.endpoint"},
				{Name: "api_key", Description: "Override API key per upstream", Secret: true, Source: "catalog.api_key"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries 2x; upstream gateways may implement their own backoff.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Performs a lightweight request against the upstream models endpoint.",
		},
		Builder: buildOpenAICompatibleRoute,
	})
}

func buildOpenAIRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	override := entry.ProviderOverrides.OpenAI
	openaiCfg := cfg.Providers.OpenAI

	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		case strings.TrimSpace(openaiCfg.APIKey) != "":
			apiKey = strings.TrimSpace(openaiCfg.APIKey)
		default:
			apiKey = strings.TrimSpace(cfg.Providers.OpenAIKey)
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("openai provider requires api key (providers.openai.api_key or catalog entry api_key)")
	}

	md := cloneMetadata(entry.Metadata)
	org := strings.TrimSpace(md["openai_organization"])
	if override != nil && strings.TrimSpace(override.Organization) != "" {
		org = strings.TrimSpace(override.Organization)
	} else if org == "" && strings.TrimSpace(openaiCfg.Organization) != "" {
		org = strings.TrimSpace(openaiCfg.Organization)
	}
	if org != "" {
		md["openai_organization"] = org
	}

	baseURL := strings.TrimSpace(entry.Endpoint)
	switch {
	case override != nil && strings.TrimSpace(override.BaseURL) != "":
		baseURL = strings.TrimSpace(override.BaseURL)
	case baseURL == "" && strings.TrimSpace(openaiCfg.BaseURL) != "":
		baseURL = strings.TrimSpace(openaiCfg.BaseURL)
	}
	if baseURL != "" {
		md["base_url"] = baseURL
	}

	setDefaultAudioMetadata(md, true, true)
	opts := native.Options{
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Organization: org,
	}
	adapter, err := native.New(opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	route := Route{
		Alias:                 entry.Alias,
		Provider:              entry.Provider,
		Model:                 entry.ProviderModel,
		Weight:                weight,
		Metadata:              md,
		Chat:                  adapter,
		ChatStream:            adapter,
		Embedding:             adapter,
		Moderations:           adapter,
		Image:                 adapter,
		AudioTranscribe:       adapter,
		AudioTranscribeStream: adapter,
		AudioTranslate:        adapter,
		TextToSpeech:          adapter,
		Models:                adapter,
		Health:                WrapHealth(adapter.HealthCheck),
	}
	route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

func buildOpenAICompatibleRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {

	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.OpenAICompatible
	compatCfg := cfg.Providers.OpenAICompatible
	openaiCfg := cfg.Providers.OpenAI
	baseURL := strings.TrimSpace(entry.Endpoint)
	if override != nil && strings.TrimSpace(override.BaseURL) != "" {
		baseURL = strings.TrimSpace(override.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(md["base_url"])
	}
	if baseURL == "" && strings.TrimSpace(compatCfg.BaseURL) != "" {
		baseURL = strings.TrimSpace(compatCfg.BaseURL)
	}
	if baseURL == "" {
		return Route{}, fmt.Errorf("openai-compatible provider requires base_url (entry.endpoint, metadata.base_url, or providers.openai_compatible.base_url)")
	}
	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		case md["api_key"] != "":
			apiKey = strings.TrimSpace(md["api_key"])
		case strings.TrimSpace(compatCfg.APIKey) != "":
			apiKey = strings.TrimSpace(compatCfg.APIKey)
		case strings.TrimSpace(openaiCfg.APIKey) != "":
			apiKey = strings.TrimSpace(openaiCfg.APIKey)
		default:
			apiKey = strings.TrimSpace(cfg.Providers.OpenAIKey)
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("openai-compatible provider requires api key")
	}
	org := strings.TrimSpace(md["openai_organization"])
	if override != nil && strings.TrimSpace(override.Organization) != "" {
		org = strings.TrimSpace(override.Organization)
	} else if org == "" && strings.TrimSpace(compatCfg.Organization) != "" {
		org = strings.TrimSpace(compatCfg.Organization)
	}
	if org != "" {
		md["openai_organization"] = org
	}
	opts := native.Options{
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Organization: org,
	}
	adapter, err := native.New(opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	setDefaultAudioMetadata(md, true, true)
	route := Route{
		Alias:    entry.Alias,
		Provider: entry.Provider,
		Model:    entry.ProviderModel,
		Weight:   weight,
		Metadata: func() map[string]string {
			md["base_url"] = baseURL
			return md
		}(),
		Chat:                  adapter,
		ChatStream:            adapter,
		Embedding:             adapter,
		Moderations:           adapter,
		Image:                 adapter,
		AudioTranscribe:       adapter,
		AudioTranscribeStream: adapter,
		AudioTranslate:        adapter,
		TextToSpeech:          adapter,
		Health:                WrapHealth(adapter.HealthCheck),
	}
	route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

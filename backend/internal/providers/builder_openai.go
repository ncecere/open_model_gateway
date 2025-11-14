package providers

import (
	"context"
	"fmt"
	"strings"

	native "github.com/ncecere/open_model_gateway/backend/internal/adapters/openai"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:        "openai",
		Description: "OpenAI native API (chat, streaming, embeddings, images, audio)",
		Capabilities: []string{
			"chat", "chat_stream", "embeddings", "images", "models",
			"audio_transcription", "audio_translation", "audio_speech",
		},
		Builder: buildOpenAIRoute,
	})
	RegisterDefinition(Definition{
		Name:         "openai-compatible",
		Description:  "OpenAI API-compatible endpoint (custom base URL)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "images", "audio_transcription", "audio_translation", "audio_speech"},
		Builder:      buildOpenAICompatibleRoute,
	})
}

func buildOpenAIRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	override := entry.ProviderOverrides.OpenAI
	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		default:
			apiKey = strings.TrimSpace(cfg.Providers.OpenAIKey)
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("openai provider requires api key (providers.openai_key or catalog entry api_key)")
	}

	md := cloneMetadata(entry.Metadata)
	if override != nil {
		if strings.TrimSpace(override.Organization) != "" {
			md["openai_organization"] = strings.TrimSpace(override.Organization)
		}
		if strings.TrimSpace(override.BaseURL) != "" {
			entry.Endpoint = strings.TrimSpace(override.BaseURL)
		}
	}
	opts := native.Options{
		APIKey:       apiKey,
		BaseURL:      strings.TrimSpace(entry.Endpoint),
		Organization: strings.TrimSpace(md["openai_organization"]),
	}
	adapter, err := native.New(opts)
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

	route := Route{
		Alias:           entry.Alias,
		Provider:        entry.Provider,
		Model:           entry.ProviderModel,
		Weight:          weight,
		Metadata:        md,
		Chat:            adapter,
		ChatStream:      adapter,
		Embedding:       adapter,
		Image:           adapter,
		AudioTranscribe: adapter,
		AudioTranslate:  adapter,
		TextToSpeech:    adapter,
		Models:          adapter,
		Health:          adapter.HealthCheck,
	}
	return route, nil
}

func buildOpenAICompatibleRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)
	md := cloneMetadata(entry.Metadata)
	override := entry.ProviderOverrides.OpenAICompatible
	baseURL := strings.TrimSpace(entry.Endpoint)
	if override != nil && strings.TrimSpace(override.BaseURL) != "" {
		baseURL = strings.TrimSpace(override.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(md["base_url"])
	}
	if baseURL == "" {
		return Route{}, fmt.Errorf("openai-compatible provider requires base_url (entry.endpoint or metadata.base_url)")
	}
	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case override != nil && strings.TrimSpace(override.APIKey) != "":
			apiKey = strings.TrimSpace(override.APIKey)
		case md["api_key"] != "":
			apiKey = strings.TrimSpace(md["api_key"])
		default:
			apiKey = strings.TrimSpace(cfg.Providers.OpenAIKey)
		}
	}
	if apiKey == "" {
		return Route{}, fmt.Errorf("openai-compatible provider requires api key")
	}
	opts := native.Options{
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Organization: strings.TrimSpace(md["openai_organization"]),
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
		Alias:    entry.Alias,
		Provider: entry.Provider,
		Model:    entry.ProviderModel,
		Weight:   weight,
		Metadata: func() map[string]string {
			md["base_url"] = baseURL
			return md
		}(),
		Chat:            adapter,
		ChatStream:      adapter,
		Embedding:       adapter,
		Image:           adapter,
		AudioTranscribe: adapter,
		AudioTranslate:  adapter,
		TextToSpeech:    adapter,
		Health:          adapter.HealthCheck,
	}
	return route, nil
}

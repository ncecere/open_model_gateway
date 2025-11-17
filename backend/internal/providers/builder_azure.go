package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/azureopenai"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:        "azure",
		Description: "Azure OpenAI (chat, embeddings, images, audio)",
		Capabilities: []string{
			"chat", "chat_stream", "embeddings", "images",
			"audio_transcription", "audio_translation", "moderations",
		},
		Descriptor: Descriptor{
			Summary: "Azure-hosted OpenAI deployments using Azure-specific endpoints and API versions.",
			Auth:    []string{"api_key"},
			ConfigInputs: []Input{
				{Name: "providers.azure_openai_endpoint", Description: "Default Azure OpenAI endpoint", Required: true, Source: "config.providers.azure_openai_endpoint"},
				{Name: "providers.azure_openai_key", Description: "Default Azure API key", Required: true, Secret: true, Source: "config.providers.azure_openai_key"},
				{Name: "providers.azure_openai_version", Description: "API version to use", Required: true, Source: "config.providers.azure_openai_version"},
			},
			EntryFields: []Input{
				{Name: "deployment", Description: "Azure deployment name", Required: true, Source: "catalog.deployment"},
				{Name: "endpoint", Description: "Override endpoint", Source: "catalog.endpoint"},
				{Name: "api_key", Description: "Override API key", Secret: true, Source: "catalog.api_key"},
				{Name: "api_version", Description: "Override API version", Source: "catalog.api_version"},
				{Name: "region", Description: "Region metadata", Source: "catalog.region"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries transient Azure errors twice; Azure's 429/503 headers dictate longer backoff.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Calls Azure deployments' /chat/completions with a noop payload for health.",
		},
		Builder: buildAzureRoute,
	})
}

func buildAzureRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)

	az := entry.ProviderOverrides.Azure

	deployment := entry.Deployment
	if az != nil && az.Deployment != "" {
		deployment = az.Deployment
	}

	endpoint := entry.Endpoint
	if endpoint == "" {
		if az != nil && az.Endpoint != "" {
			endpoint = az.Endpoint
		} else {
			endpoint = cfg.Providers.AzureOpenAIEndpoint
		}
	}
	apiKey := entry.APIKey
	if apiKey == "" {
		if az != nil && az.APIKey != "" {
			apiKey = az.APIKey
		} else {
			apiKey = cfg.Providers.AzureOpenAIKey
		}
	}
	apiVersion := entry.APIVersion
	if apiVersion == "" {
		switch {
		case az != nil && az.APIVersion != "":
			apiVersion = az.APIVersion
		case entry.Metadata != nil && entry.Metadata["api_version"] != "":
			apiVersion = entry.Metadata["api_version"]
		default:
			apiVersion = cfg.Providers.AzureOpenAIVersion
		}
	}
	region := entry.Region
	if region == "" && az != nil && az.Region != "" {
		region = az.Region
	}

	if endpoint == "" || apiKey == "" {
		return Route{}, fmt.Errorf("azure endpoint/api key must be provided")
	}

	adapter, err := azureopenai.New(azureopenai.Options{
		Endpoint:   endpoint,
		APIKey:     apiKey,
		APIVersion: apiVersion,
	})
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	metadata := cloneMetadata(entry.Metadata)
	metadata["deployment"] = deployment
	if region != "" {
		metadata["region"] = region
	}
	setDefaultAudioMetadata(metadata, false, false)

	route := Route{
		Alias:           entry.Alias,
		Provider:        entry.Provider,
		Model:           entry.ProviderModel,
		Weight:          weight,
		Metadata:        metadata,
		Chat:            adapter,
		ChatStream:      adapter,
		Embedding:       adapter,
		Moderations:     adapter,
		Image:           adapter,
		AudioTranscribe: adapter,
		AudioTranslate:  adapter,
		Models:          adapter,
		Health:          WrapHealth(adapter.HealthCheck),
	}
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 250 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

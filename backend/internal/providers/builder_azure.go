package providers

import (
	"context"
	"fmt"
	"strings"
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
				{Name: "providers.azure.endpoint", Description: "Default Azure OpenAI endpoint", Required: true, Source: "config.providers.azure.endpoint"},
				{Name: "providers.azure.api_key", Description: "Default Azure API key", Required: true, Secret: true, Source: "config.providers.azure.api_key"},
				{Name: "providers.azure.api_version", Description: "API version to use", Required: true, Source: "config.providers.azure.api_version"},
				{Name: "providers.azure.region", Description: "Default region metadata", Source: "config.providers.azure.region"},
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
	cfgAzure := cfg.Providers.Azure

	deployment := entry.Deployment
	if az != nil && strings.TrimSpace(az.Deployment) != "" {
		deployment = strings.TrimSpace(az.Deployment)
	} else if deployment == "" && strings.TrimSpace(cfgAzure.Deployment) != "" {
		deployment = strings.TrimSpace(cfgAzure.Deployment)
	}

	endpoint := strings.TrimSpace(entry.Endpoint)
	if endpoint == "" {
		if az != nil && strings.TrimSpace(az.Endpoint) != "" {
			endpoint = strings.TrimSpace(az.Endpoint)
		} else {
			endpoint = strings.TrimSpace(cfgAzure.Endpoint)
			if endpoint == "" {
				endpoint = strings.TrimSpace(cfg.Providers.AzureOpenAIEndpoint)
			}
		}
	}
	apiKey := strings.TrimSpace(entry.APIKey)
	if apiKey == "" {
		switch {
		case az != nil && strings.TrimSpace(az.APIKey) != "":
			apiKey = strings.TrimSpace(az.APIKey)
		case strings.TrimSpace(cfgAzure.APIKey) != "":
			apiKey = strings.TrimSpace(cfgAzure.APIKey)
		default:
			apiKey = strings.TrimSpace(cfg.Providers.AzureOpenAIKey)
		}
	}
	apiVersion := strings.TrimSpace(entry.APIVersion)
	if apiVersion == "" {
		switch {
		case az != nil && strings.TrimSpace(az.APIVersion) != "":
			apiVersion = strings.TrimSpace(az.APIVersion)
		case entry.Metadata != nil && strings.TrimSpace(entry.Metadata["api_version"]) != "":
			apiVersion = strings.TrimSpace(entry.Metadata["api_version"])
		case strings.TrimSpace(cfgAzure.APIVersion) != "":
			apiVersion = strings.TrimSpace(cfgAzure.APIVersion)
		default:
			apiVersion = strings.TrimSpace(cfg.Providers.AzureOpenAIVersion)
		}
	}
	region := strings.TrimSpace(entry.Region)
	if region == "" {
		switch {
		case az != nil && strings.TrimSpace(az.Region) != "":
			region = strings.TrimSpace(az.Region)
		case strings.TrimSpace(cfgAzure.Region) != "":
			region = strings.TrimSpace(cfgAzure.Region)
		}
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
	route.Capabilities = deriveCapabilities(entry.Modalities, route.Metadata)
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 3, InitialBackoff: 500 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("openai", entry.ProviderOverrides.Tokenizer, route.Metadata)
	return route, nil
}

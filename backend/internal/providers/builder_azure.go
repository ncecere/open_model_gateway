package providers

import (
	"context"
	"fmt"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/azureopenai"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:        "azure",
		Description: "Azure OpenAI (chat, embeddings, images, audio)",
		Capabilities: []string{
			"chat", "chat_stream", "embeddings", "images",
			"audio_transcription", "audio_translation",
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

	return Route{
		Alias:           entry.Alias,
		Provider:        entry.Provider,
		Model:           entry.ProviderModel,
		Weight:          weight,
		Metadata:        metadata,
		Chat:            adapter,
		ChatStream:      adapter,
		Embedding:       adapter,
		Image:           adapter,
		AudioTranscribe: adapter,
		AudioTranslate:  adapter,
		Models:          adapter,
		Health:          adapter.HealthCheck,
	}, nil
}

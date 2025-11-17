package providers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/bedrock"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func init() {
	RegisterDefinition(Definition{
		Name:         "bedrock",
		Description:  "AWS Bedrock (Anthropic Claude, Titan embeddings/images)",
		Capabilities: []string{"chat", "chat_stream", "embeddings", "images"},
		Descriptor: Descriptor{
			Summary: "AWS Bedrock adapter supporting Anthropic Claude chat/SSE and Titan embeddings/images.",
			Auth:    []string{"aws_keys", "sts_profile"},
			ConfigInputs: []Input{
				{Name: "providers.aws_region", Description: "Default AWS region", Required: true, Source: "config.providers.aws_region"},
				{Name: "providers.aws_access_key_id", Description: "Access key (when not using IAM role)", Secret: true, Source: "config.providers.aws_access_key_id"},
				{Name: "providers.aws_secret_access_key", Description: "Secret key", Secret: true, Source: "config.providers.aws_secret_access_key"},
			},
			EntryFields: []Input{
				{Name: "region", Description: "Override AWS region", Source: "catalog.region"},
				{Name: "aws_access_key_id", Description: "Override access key", Secret: true, Source: "catalog.metadata.aws_access_key_id"},
				{Name: "aws_secret_access_key", Description: "Override secret key", Secret: true, Source: "catalog.metadata.aws_secret_access_key"},
				{Name: "aws_session_token", Description: "STS session token", Secret: true, Source: "catalog.metadata.aws_session_token"},
				{Name: "aws_profile", Description: "Use shared credentials profile", Source: "catalog.metadata.aws_profile"},
				{Name: "bedrock_chat_format", Description: "Override chat format (e.g. anthropic_messages)", Source: "catalog.metadata.bedrock_chat_format"},
				{Name: "bedrock_embedding_format", Description: "Override embedding format", Source: "catalog.metadata.bedrock_embedding_format"},
			},
			RetryPolicy: RetryDescriptor{
				Description:   "Executor retries two times; adapters translate AWS throttling errors into retryable responses.",
				DefaultPolicy: "exponential 250ms backoff, max 2 attempts",
			},
			HealthNotes: "Uses Bedrock model invocation with minimal payload to verify IAM + region configuration.",
		},
		Builder: buildBedrockRoute,
	})
}

func buildBedrockRoute(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error) {
	cfg = EnsureConfig(cfg)

	override := entry.ProviderOverrides.Bedrock

	region := entry.Region
	if override != nil && strings.TrimSpace(override.Region) != "" {
		region = strings.TrimSpace(override.Region)
	}
	if region == "" {
		region = cfg.Providers.AWSRegion
	}
	if region == "" {
		return Route{}, fmt.Errorf("aws region required for bedrock provider")
	}

	metadata := cloneMetadata(entry.Metadata)
	metadata["region"] = region
	if entry.Deployment != "" {
		metadata["deployment"] = entry.Deployment
	}

	chatFormat := strings.TrimSpace(metadata["bedrock_chat_format"])
	if override != nil && strings.TrimSpace(override.ChatFormat) != "" {
		chatFormat = strings.TrimSpace(override.ChatFormat)
	}
	if chatFormat == "" && strings.Contains(entry.ProviderModel, ".anthropic.") {
		chatFormat = bedrock.ChatFormatAnthropicMessages
	}
	if chatFormat != "" {
		metadata["bedrock_chat_format"] = chatFormat
	}

	embeddingFormat := strings.TrimSpace(metadata["bedrock_embedding_format"])
	if override != nil && strings.TrimSpace(override.EmbeddingFormat) != "" {
		embeddingFormat = strings.TrimSpace(override.EmbeddingFormat)
	}
	if embeddingFormat == "" && strings.Contains(entry.ProviderModel, "titan-embed") {
		embeddingFormat = bedrock.EmbeddingFormatTitanText
	}
	if embeddingFormat != "" {
		metadata["bedrock_embedding_format"] = embeddingFormat
	}

	defaultMaxTokens := entry.MaxOutputTokens
	if defaultMaxTokens == 0 {
		switch {
		case override != nil && override.DefaultMaxTokens != 0:
			defaultMaxTokens = override.DefaultMaxTokens
		case metadata["bedrock_default_max_tokens"] != "":
			if parsed, err := strconv.Atoi(metadata["bedrock_default_max_tokens"]); err == nil {
				defaultMaxTokens = int32(parsed)
			}
		}
	}

	var embedDims int32
	switch {
	case override != nil && override.EmbedDims != 0:
		embedDims = override.EmbedDims
	case metadata["bedrock_embed_dims"] != "":
		if parsed, err := strconv.Atoi(metadata["bedrock_embed_dims"]); err == nil {
			embedDims = int32(parsed)
		}
	}

	embedNormalize := false
	switch {
	case override != nil && override.EmbedNormalize:
		embedNormalize = true
	case metadata["bedrock_embed_normalize"] != "":
		if parsed, err := strconv.ParseBool(metadata["bedrock_embed_normalize"]); err == nil {
			embedNormalize = parsed
		}
	}

	imageTask := strings.TrimSpace(metadata["bedrock_image_task_type"])
	if override != nil && strings.TrimSpace(override.ImageTaskType) != "" {
		imageTask = strings.TrimSpace(override.ImageTaskType)
	}
	if imageTask == "" && supportsModality(entry.Modalities, "image") {
		imageTask = "TEXT_IMAGE"
	}
	if imageTask != "" {
		metadata["bedrock_image_task_type"] = imageTask
	}

	opts := bedrock.Options{
		Region:           region,
		ModelID:          entry.ProviderModel,
		ChatFormat:       chatFormat,
		EmbeddingFormat:  embeddingFormat,
		DefaultMaxTokens: defaultMaxTokens,
		ImageTaskType:    imageTask,
		AnthropicVersion: func() string {
			if override != nil && strings.TrimSpace(override.AnthropicVersion) != "" {
				return strings.TrimSpace(override.AnthropicVersion)
			}
			return metadata["anthropic_version"]
		}(),
		EmbedDimensions: embedDims,
		EmbedNormalize:  embedNormalize,
		AccessKeyID: func() string {
			if override != nil && strings.TrimSpace(override.AccessKeyID) != "" {
				return strings.TrimSpace(override.AccessKeyID)
			}
			if v := metadata["aws_access_key_id"]; v != "" {
				return v
			}
			return cfg.Providers.AWSAccessKeyID
		}(),
		SecretAccessKey: func() string {
			if override != nil && strings.TrimSpace(override.SecretAccessKey) != "" {
				return strings.TrimSpace(override.SecretAccessKey)
			}
			if v := metadata["aws_secret_access_key"]; v != "" {
				return v
			}
			return cfg.Providers.AWSSecretAccessKey
		}(),
		SessionToken: func() string {
			if override != nil && strings.TrimSpace(override.SessionToken) != "" {
				return strings.TrimSpace(override.SessionToken)
			}
			return metadata["aws_session_token"]
		}(),
		Profile: func() string {
			if override != nil && strings.TrimSpace(override.Profile) != "" {
				return strings.TrimSpace(override.Profile)
			}
			return metadata["aws_profile"]
		}(),
		Metadata: metadata,
	}

	adapter, err := bedrock.New(ctx, opts)
	if err != nil {
		return Route{}, err
	}

	weight := entry.Weight
	if weight == 0 {
		weight = 100
	}

	metadata["model_id"] = entry.ProviderModel

	route := Route{
		Alias:    entry.Alias,
		Provider: entry.Provider,
		Model:    entry.ProviderModel,
		Weight:   weight,
		Metadata: metadata,
		Health:   WrapHealth(adapter.HealthCheck),
	}
	route.Retry = mergeRetry(RetryConfig{MaxAttempts: 2, InitialBackoff: 300 * time.Millisecond, BackoffMultiplier: 2}, entry.ProviderOverrides.Retry, route.Metadata)
	route.Tokenizer = selectTokenizer("anthropic", entry.ProviderOverrides.Tokenizer, route.Metadata)

	if supportsModality(entry.Modalities, "text") && chatFormat != "" {
		route.Chat = adapter
		if chatFormat == bedrock.ChatFormatAnthropicMessages {
			route.ChatStream = adapter
		}
	}
	if supportsEmbedding(entry.Modalities) && embeddingFormat != "" {
		route.Embedding = adapter
	}
	if imageTask != "" && supportsModality(entry.Modalities, "image") {
		route.Image = adapter
	}

	if route.Chat == nil && route.Embedding == nil && route.Image == nil {
		return Route{}, errors.New("bedrock route has no supported modalities")
	}

	return route, nil
}

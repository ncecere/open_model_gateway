// Package metadata provides typed structs for provider route metadata.
// It replaces untyped map[string]string with compile-time type-safe access.
package metadata

import (
	"strconv"
	"strings"
	"time"
)

// Route represents typed metadata for a provider route.
// This provides type-safe access to metadata fields while remaining
// backwards compatible with existing map[string]string configurations.
type Route struct {
	// Core identifies common routing metadata.
	Core CoreMetadata

	// RateLimits holds per-route rate limit overrides.
	RateLimits RateLimitMetadata

	// Retry holds retry configuration overrides.
	Retry RetryMetadata

	// Audio holds audio-related configuration.
	Audio AudioMetadata

	// Provider holds provider-specific metadata (AWS, Azure, etc.)
	Provider ProviderMetadata

	// raw stores the original map for unknown/extension fields.
	raw map[string]string
}

// CoreMetadata contains common fields used across providers.
type CoreMetadata struct {
	// Deployment is the provider-specific deployment/model identifier.
	Deployment string

	// Region is the cloud region for the deployment.
	Region string

	// Tokenizer specifies which tokenizer to use for token counting.
	Tokenizer string
}

// RateLimitMetadata holds per-route rate limiting configuration.
type RateLimitMetadata struct {
	// RequestsPerMinute overrides the default RPM limit.
	RequestsPerMinute *int32

	// TokensPerMinute overrides the default TPM limit.
	TokensPerMinute *int32

	// ParallelRequests overrides the default parallel request limit.
	ParallelRequests *int32
}

// RetryMetadata holds retry configuration for a route.
type RetryMetadata struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts *int

	// InitialBackoff is the initial backoff duration.
	InitialBackoff *time.Duration

	// BackoffMultiplier is the backoff multiplier for exponential backoff.
	BackoffMultiplier *float64
}

// AudioMetadata holds audio-related configuration.
type AudioMetadata struct {
	// Voice is the default voice for TTS.
	Voice string

	// DefaultVoice is an alias for Voice.
	DefaultVoice string

	// Format is the default audio output format.
	Format string

	// Formats lists supported audio formats.
	Formats []string

	// TimestampGranularities lists supported granularities.
	TimestampGranularities []string

	// StreamingEnabled indicates if streaming is supported.
	StreamingEnabled *bool
}

// ProviderMetadata holds provider-specific configuration.
type ProviderMetadata struct {
	// AWS holds AWS/Bedrock-specific settings.
	AWS AWSMetadata

	// Azure holds Azure-specific settings.
	Azure AzureMetadata

	// Vertex holds Vertex AI-specific settings.
	Vertex VertexMetadata

	// Groq holds Groq-specific settings.
	Groq GroqMetadata

	// VLLM holds vLLM-specific settings.
	VLLM VLLMMetadata
}

// AWSMetadata holds AWS Bedrock-specific configuration.
type AWSMetadata struct {
	// AccessKeyID for AWS credentials override.
	AccessKeyID string

	// SecretAccessKey for AWS credentials override.
	SecretAccessKey string

	// SessionToken for temporary credentials.
	SessionToken string

	// Profile is the AWS profile name.
	Profile string

	// ModelID is the Bedrock model identifier.
	ModelID string

	// ChatFormat is the chat request format (messages, titan, llama).
	ChatFormat string

	// EmbeddingFormat is the embedding format (titan, cohere).
	EmbeddingFormat string

	// DefaultMaxTokens is the default max tokens for the model.
	DefaultMaxTokens *int32

	// EmbedDimensions is the embedding dimensions.
	EmbedDimensions *int32

	// EmbedNormalize indicates whether to normalize embeddings.
	EmbedNormalize *bool

	// ImageTaskType is the image generation task type.
	ImageTaskType string

	// ImageQuality is the default image quality.
	ImageQuality string

	// ImageStyle is the default image style.
	ImageStyle string

	// ImageSeed is the seed for image generation.
	ImageSeed *int64

	// AnthropicVersion is the Anthropic API version for Bedrock.
	AnthropicVersion string
}

// AzureMetadata holds Azure-specific configuration.
type AzureMetadata struct {
	// APIVersion is the Azure OpenAI API version.
	APIVersion string
}

// VertexMetadata holds Vertex AI-specific configuration.
type VertexMetadata struct {
	// EditMode is the image edit mode.
	EditMode string

	// PersonGeneration controls person generation in images.
	PersonGeneration string
}

// GroqMetadata holds Groq-specific configuration.
type GroqMetadata struct {
	// Region is the Groq region.
	Region string
}

// VLLMMetadata holds vLLM-specific configuration.
type VLLMMetadata struct {
	// BaseURL is the vLLM server base URL.
	BaseURL string

	// AuthHeader is the custom authentication header name.
	AuthHeader string
}

// FromMap parses a map[string]string into a typed Route metadata struct.
func FromMap(m map[string]string) Route {
	if m == nil {
		return Route{raw: make(map[string]string)}
	}

	r := Route{raw: m}

	// Core metadata
	r.Core.Deployment = getString(m, "deployment")
	r.Core.Region = getString(m, "region")
	r.Core.Tokenizer = getString(m, "tokenizer")

	// Rate limits
	r.RateLimits.RequestsPerMinute = getInt32Ptr(m, "requests_per_minute")
	r.RateLimits.TokensPerMinute = getInt32Ptr(m, "tokens_per_minute")
	r.RateLimits.ParallelRequests = getInt32Ptr(m, "parallel_requests")

	// Retry config
	r.Retry.MaxAttempts = getIntPtr(m, "retry_max_attempts")
	r.Retry.InitialBackoff = getDurationMsPtr(m, "retry_initial_backoff_ms")
	r.Retry.BackoffMultiplier = getFloat64Ptr(m, "retry_backoff_multiplier")

	// Audio metadata
	r.Audio.Voice = getString(m, "audio_voice")
	r.Audio.DefaultVoice = getString(m, "audio_default_voice")
	r.Audio.Format = getString(m, "audio_format")
	r.Audio.Formats = getStringList(m, "audio_formats")
	r.Audio.TimestampGranularities = getStringList(m, "audio_timestamp_granularities")
	r.Audio.StreamingEnabled = parseBoolPtr(getString(m, "audio_streaming"))

	// AWS/Bedrock metadata
	r.Provider.AWS.AccessKeyID = getString(m, "aws_access_key_id")
	r.Provider.AWS.SecretAccessKey = getString(m, "aws_secret_access_key")
	r.Provider.AWS.SessionToken = getString(m, "aws_session_token")
	r.Provider.AWS.Profile = getString(m, "aws_profile")
	r.Provider.AWS.ModelID = getString(m, "model_id")
	r.Provider.AWS.ChatFormat = getString(m, "bedrock_chat_format")
	r.Provider.AWS.EmbeddingFormat = getString(m, "bedrock_embedding_format")
	r.Provider.AWS.DefaultMaxTokens = getInt32Ptr(m, "bedrock_default_max_tokens")
	r.Provider.AWS.EmbedDimensions = getInt32Ptr(m, "bedrock_embed_dims")
	r.Provider.AWS.EmbedNormalize = getBoolPtr(m, "bedrock_embed_normalize")
	r.Provider.AWS.ImageTaskType = getString(m, "bedrock_image_task_type")
	r.Provider.AWS.ImageQuality = getString(m, "bedrock_image_quality")
	r.Provider.AWS.ImageStyle = getString(m, "bedrock_image_style")
	r.Provider.AWS.ImageSeed = getInt64Ptr(m, "bedrock_image_seed")
	r.Provider.AWS.AnthropicVersion = getString(m, "anthropic_version")

	// Azure metadata
	r.Provider.Azure.APIVersion = getString(m, "api_version")

	// Vertex metadata
	r.Provider.Vertex.EditMode = getString(m, "vertex_edit_mode")
	r.Provider.Vertex.PersonGeneration = getString(m, "vertex_person_generation")

	// Groq metadata
	r.Provider.Groq.Region = getString(m, "groq_region")

	// vLLM metadata
	r.Provider.VLLM.BaseURL = getString(m, "base_url")
	r.Provider.VLLM.AuthHeader = getString(m, "auth_header")

	return r
}

// ToMap converts the typed metadata back to a map[string]string.
func (r Route) ToMap() map[string]string {
	m := make(map[string]string)

	// Copy raw first to preserve unknown keys
	for k, v := range r.raw {
		m[k] = v
	}

	// Core metadata
	setString(m, "deployment", r.Core.Deployment)
	setString(m, "region", r.Core.Region)
	setString(m, "tokenizer", r.Core.Tokenizer)

	// Rate limits
	setInt32Ptr(m, "requests_per_minute", r.RateLimits.RequestsPerMinute)
	setInt32Ptr(m, "tokens_per_minute", r.RateLimits.TokensPerMinute)
	setInt32Ptr(m, "parallel_requests", r.RateLimits.ParallelRequests)

	// Retry config
	setIntPtr(m, "retry_max_attempts", r.Retry.MaxAttempts)
	setDurationMsPtr(m, "retry_initial_backoff_ms", r.Retry.InitialBackoff)
	setFloat64Ptr(m, "retry_backoff_multiplier", r.Retry.BackoffMultiplier)

	// Audio metadata
	setString(m, "audio_voice", r.Audio.Voice)
	setString(m, "audio_default_voice", r.Audio.DefaultVoice)
	setString(m, "audio_format", r.Audio.Format)
	setStringList(m, "audio_formats", r.Audio.Formats)
	setStringList(m, "audio_timestamp_granularities", r.Audio.TimestampGranularities)
	setBoolPtr(m, "audio_streaming", r.Audio.StreamingEnabled)

	// AWS/Bedrock metadata
	setString(m, "aws_access_key_id", r.Provider.AWS.AccessKeyID)
	setString(m, "aws_secret_access_key", r.Provider.AWS.SecretAccessKey)
	setString(m, "aws_session_token", r.Provider.AWS.SessionToken)
	setString(m, "aws_profile", r.Provider.AWS.Profile)
	setString(m, "model_id", r.Provider.AWS.ModelID)
	setString(m, "bedrock_chat_format", r.Provider.AWS.ChatFormat)
	setString(m, "bedrock_embedding_format", r.Provider.AWS.EmbeddingFormat)
	setInt32Ptr(m, "bedrock_default_max_tokens", r.Provider.AWS.DefaultMaxTokens)
	setInt32Ptr(m, "bedrock_embed_dims", r.Provider.AWS.EmbedDimensions)
	setBoolPtr(m, "bedrock_embed_normalize", r.Provider.AWS.EmbedNormalize)
	setString(m, "bedrock_image_task_type", r.Provider.AWS.ImageTaskType)
	setString(m, "bedrock_image_quality", r.Provider.AWS.ImageQuality)
	setString(m, "bedrock_image_style", r.Provider.AWS.ImageStyle)
	setInt64Ptr(m, "bedrock_image_seed", r.Provider.AWS.ImageSeed)
	setString(m, "anthropic_version", r.Provider.AWS.AnthropicVersion)

	// Azure metadata
	setString(m, "api_version", r.Provider.Azure.APIVersion)

	// Vertex metadata
	setString(m, "vertex_edit_mode", r.Provider.Vertex.EditMode)
	setString(m, "vertex_person_generation", r.Provider.Vertex.PersonGeneration)

	// Groq metadata
	setString(m, "groq_region", r.Provider.Groq.Region)

	// vLLM metadata
	setString(m, "base_url", r.Provider.VLLM.BaseURL)
	setString(m, "auth_header", r.Provider.VLLM.AuthHeader)

	return m
}

// Raw returns the underlying raw map for access to unknown/extension fields.
func (r Route) Raw() map[string]string {
	return r.raw
}

// Get returns a raw value by key, for accessing extension fields.
func (r Route) Get(key string) string {
	if r.raw == nil {
		return ""
	}
	return strings.TrimSpace(r.raw[key])
}

// Helper functions for parsing

func getString(m map[string]string, key string) string {
	return strings.TrimSpace(m[key])
}

func getIntPtr(m map[string]string, key string) *int {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.Atoi(v); err == nil {
		return &parsed
	}
	return nil
}

func getInt32Ptr(m map[string]string, key string) *int32 {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.ParseInt(v, 10, 32); err == nil {
		val := int32(parsed)
		return &val
	}
	return nil
}

func getInt64Ptr(m map[string]string, key string) *int64 {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
		return &parsed
	}
	return nil
}

func getFloat64Ptr(m map[string]string, key string) *float64 {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.ParseFloat(v, 64); err == nil {
		return &parsed
	}
	return nil
}

func getBoolPtr(m map[string]string, key string) *bool {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.ParseBool(v); err == nil {
		return &parsed
	}
	return nil
}

func getDurationMsPtr(m map[string]string, key string) *time.Duration {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
		d := time.Duration(parsed) * time.Millisecond
		return &d
	}
	return nil
}

func getStringList(m map[string]string, key string) []string {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseBoolPtr(v string) *bool {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "true", "yes", "1", "enabled":
		val := true
		return &val
	case "false", "no", "0", "disabled":
		val := false
		return &val
	}
	return nil
}

// Helper functions for serializing

func setString(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func setIntPtr(m map[string]string, key string, value *int) {
	if value != nil {
		m[key] = strconv.Itoa(*value)
	}
}

func setInt32Ptr(m map[string]string, key string, value *int32) {
	if value != nil {
		m[key] = strconv.FormatInt(int64(*value), 10)
	}
}

func setInt64Ptr(m map[string]string, key string, value *int64) {
	if value != nil {
		m[key] = strconv.FormatInt(*value, 10)
	}
}

func setFloat64Ptr(m map[string]string, key string, value *float64) {
	if value != nil {
		m[key] = strconv.FormatFloat(*value, 'f', -1, 64)
	}
}

func setBoolPtr(m map[string]string, key string, value *bool) {
	if value != nil {
		m[key] = strconv.FormatBool(*value)
	}
}

func setDurationMsPtr(m map[string]string, key string, value *time.Duration) {
	if value != nil {
		m[key] = strconv.FormatInt(value.Milliseconds(), 10)
	}
}

func setStringList(m map[string]string, key string, value []string) {
	if len(value) > 0 {
		m[key] = strings.Join(value, ",")
	}
}

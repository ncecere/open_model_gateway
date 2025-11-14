package config

// ProviderOverrides captures provider specific configuration for a model catalog entry.
type ProviderOverrides struct {
	Azure            *AzureProviderConfig            `mapstructure:"azure" json:"azure,omitempty"`
	Vertex           *VertexProviderConfig           `mapstructure:"vertex" json:"vertex,omitempty"`
	Bedrock          *BedrockProviderConfig          `mapstructure:"bedrock" json:"bedrock,omitempty"`
	OpenAI           *OpenAIProviderConfig           `mapstructure:"openai" json:"openai,omitempty"`
	OpenAICompatible *OpenAICompatibleProviderConfig `mapstructure:"openai_compatible" json:"openai_compatible,omitempty"`
	Anthropic        *AnthropicProviderConfig        `mapstructure:"anthropic" json:"anthropic,omitempty"`
}

type AzureProviderConfig struct {
	Deployment string `mapstructure:"deployment" json:"deployment"`
	Endpoint   string `mapstructure:"endpoint" json:"endpoint"`
	APIKey     string `mapstructure:"api_key" json:"api_key"`
	APIVersion string `mapstructure:"api_version" json:"api_version"`
	Region     string `mapstructure:"region" json:"region"`
}

type VertexProviderConfig struct {
	ProjectID         string `mapstructure:"gcp_project_id" json:"gcp_project_id"`
	Location          string `mapstructure:"vertex_location" json:"vertex_location"`
	Publisher         string `mapstructure:"vertex_publisher" json:"vertex_publisher"`
	CredentialsJSON   string `mapstructure:"gcp_credentials_json" json:"gcp_credentials_json"`
	CredentialsFormat string `mapstructure:"gcp_credentials_format" json:"gcp_credentials_format"`
}

type BedrockProviderConfig struct {
	Region           string `mapstructure:"region" json:"region"`
	ChatFormat       string `mapstructure:"bedrock_chat_format" json:"bedrock_chat_format"`
	EmbeddingFormat  string `mapstructure:"bedrock_embedding_format" json:"bedrock_embedding_format"`
	DefaultMaxTokens int32  `mapstructure:"bedrock_default_max_tokens" json:"bedrock_default_max_tokens"`
	EmbedDims        int32  `mapstructure:"bedrock_embed_dims" json:"bedrock_embed_dims"`
	EmbedNormalize   bool   `mapstructure:"bedrock_embed_normalize" json:"bedrock_embed_normalize"`
	ImageTaskType    string `mapstructure:"bedrock_image_task_type" json:"bedrock_image_task_type"`
	AnthropicVersion string `mapstructure:"anthropic_version" json:"anthropic_version"`
	AccessKeyID      string `mapstructure:"aws_access_key_id" json:"aws_access_key_id"`
	SecretAccessKey  string `mapstructure:"aws_secret_access_key" json:"aws_secret_access_key"`
	SessionToken     string `mapstructure:"aws_session_token" json:"aws_session_token"`
	Profile          string `mapstructure:"aws_profile" json:"aws_profile"`
}

type OpenAIProviderConfig struct {
	APIKey       string `mapstructure:"api_key" json:"api_key"`
	Organization string `mapstructure:"openai_organization" json:"openai_organization"`
	BaseURL      string `mapstructure:"base_url" json:"base_url"`
}

type OpenAICompatibleProviderConfig struct {
	BaseURL      string `mapstructure:"base_url" json:"base_url"`
	APIKey       string `mapstructure:"api_key" json:"api_key"`
	Organization string `mapstructure:"openai_organization" json:"openai_organization"`
}

type AnthropicProviderConfig struct {
	APIKey  string `mapstructure:"api_key" json:"api_key"`
	BaseURL string `mapstructure:"base_url" json:"base_url"`
	Version string `mapstructure:"version" json:"version"`
}

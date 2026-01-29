package catalog

// Preset is a ready-made model catalog template that an admin can apply with
// minimal input (typically just picking a provider and optionally overriding
// the alias). Pricing is per 1M tokens in USD unless noted.
type Preset struct {
	// Identity
	Alias         string `json:"alias"`
	Provider      string `json:"provider"`
	ProviderModel string `json:"provider_model"`

	// Classification
	ModelType  string   `json:"model_type"`
	Modalities []string `json:"modalities,omitempty"`

	// Capabilities
	ContextWindow   int32 `json:"context_window,omitempty"`
	MaxOutputTokens int32 `json:"max_output_tokens,omitempty"`
	SupportsTools   bool  `json:"supports_tools,omitempty"`

	// Pricing (per 1M tokens, USD)
	PriceInput  float64 `json:"price_input"`
	PriceOutput float64 `json:"price_output"`

	// Display
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
}

// DefaultPresets returns the built-in model presets grouped by provider.
func DefaultPresets() []Preset {
	return []Preset{
		// --- OpenAI ---
		{Alias: "gpt-4.1", Provider: "openai", ProviderModel: "gpt-4.1", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 1047576, MaxOutputTokens: 32768, SupportsTools: true, PriceInput: 2.00, PriceOutput: 8.00, Category: "OpenAI", Description: "GPT-4.1 flagship"},
		{Alias: "gpt-4.1-mini", Provider: "openai", ProviderModel: "gpt-4.1-mini", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 1047576, MaxOutputTokens: 32768, SupportsTools: true, PriceInput: 0.40, PriceOutput: 1.60, Category: "OpenAI", Description: "GPT-4.1 Mini cost-efficient"},
		{Alias: "gpt-4.1-nano", Provider: "openai", ProviderModel: "gpt-4.1-nano", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 1047576, MaxOutputTokens: 32768, SupportsTools: true, PriceInput: 0.10, PriceOutput: 0.40, Category: "OpenAI", Description: "GPT-4.1 Nano fastest/cheapest"},
		{Alias: "gpt-4o", Provider: "openai", ProviderModel: "gpt-4o", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 128000, MaxOutputTokens: 16384, SupportsTools: true, PriceInput: 2.50, PriceOutput: 10.00, Category: "OpenAI", Description: "GPT-4o multimodal"},
		{Alias: "gpt-4o-mini", Provider: "openai", ProviderModel: "gpt-4o-mini", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 128000, MaxOutputTokens: 16384, SupportsTools: true, PriceInput: 0.15, PriceOutput: 0.60, Category: "OpenAI", Description: "GPT-4o Mini affordable multimodal"},
		{Alias: "o3", Provider: "openai", ProviderModel: "o3", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 200000, MaxOutputTokens: 100000, SupportsTools: true, PriceInput: 2.00, PriceOutput: 8.00, Category: "OpenAI", Description: "o3 reasoning"},
		{Alias: "o3-mini", Provider: "openai", ProviderModel: "o3-mini", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 200000, MaxOutputTokens: 100000, SupportsTools: true, PriceInput: 1.10, PriceOutput: 4.40, Category: "OpenAI", Description: "o3-mini reasoning cost-efficient"},
		{Alias: "o4-mini", Provider: "openai", ProviderModel: "o4-mini", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 100000, SupportsTools: true, PriceInput: 1.10, PriceOutput: 4.40, Category: "OpenAI", Description: "o4-mini reasoning multimodal"},
		{Alias: "gpt-image-1", Provider: "openai", ProviderModel: "gpt-image-1", ModelType: "image", Modalities: []string{"image"}, PriceInput: 0, PriceOutput: 0, Category: "OpenAI", Description: "GPT Image 1 generation"},
		{Alias: "text-embedding-3-small", Provider: "openai", ProviderModel: "text-embedding-3-small", ModelType: "embedding", Modalities: []string{"text"}, ContextWindow: 8191, PriceInput: 0.02, PriceOutput: 0, Category: "OpenAI", Description: "Small embedding model"},
		{Alias: "text-embedding-3-large", Provider: "openai", ProviderModel: "text-embedding-3-large", ModelType: "embedding", Modalities: []string{"text"}, ContextWindow: 8191, PriceInput: 0.13, PriceOutput: 0, Category: "OpenAI", Description: "Large embedding model"},
		{Alias: "tts-1", Provider: "openai", ProviderModel: "tts-1", ModelType: "audio_speech", Modalities: []string{"audio"}, PriceInput: 15.00, PriceOutput: 0, Category: "OpenAI", Description: "Text-to-speech standard"},
		{Alias: "tts-1-hd", Provider: "openai", ProviderModel: "tts-1-hd", ModelType: "audio_speech", Modalities: []string{"audio"}, PriceInput: 30.00, PriceOutput: 0, Category: "OpenAI", Description: "Text-to-speech HD"},
		{Alias: "gpt-4o-mini-tts", Provider: "openai", ProviderModel: "gpt-4o-mini-tts", ModelType: "audio_speech", Modalities: []string{"audio"}, PriceInput: 0.60, PriceOutput: 12.00, Category: "OpenAI", Description: "GPT-4o Mini TTS"},

		// --- Anthropic ---
		{Alias: "claude-sonnet-4", Provider: "anthropic", ProviderModel: "claude-sonnet-4-20250514", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 64000, SupportsTools: true, PriceInput: 3.00, PriceOutput: 15.00, Category: "Anthropic", Description: "Claude Sonnet 4"},
		{Alias: "claude-opus-4", Provider: "anthropic", ProviderModel: "claude-opus-4-20250514", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 32000, SupportsTools: true, PriceInput: 15.00, PriceOutput: 75.00, Category: "Anthropic", Description: "Claude Opus 4"},
		{Alias: "claude-3.5-sonnet", Provider: "anthropic", ProviderModel: "claude-3-5-sonnet-20241022", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 3.00, PriceOutput: 15.00, Category: "Anthropic", Description: "Claude 3.5 Sonnet"},
		{Alias: "claude-3.5-haiku", Provider: "anthropic", ProviderModel: "claude-3-5-haiku-20241022", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 200000, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 0.80, PriceOutput: 4.00, Category: "Anthropic", Description: "Claude 3.5 Haiku fast/cheap"},

		// --- Google Vertex (Gemini) ---
		{Alias: "gemini-2.5-pro", Provider: "vertex", ProviderModel: "gemini-2.5-pro-preview-06-05", ModelType: "llm", Modalities: []string{"text", "image", "audio", "video"}, ContextWindow: 1048576, MaxOutputTokens: 65536, SupportsTools: true, PriceInput: 1.25, PriceOutput: 10.00, Category: "Google", Description: "Gemini 2.5 Pro multimodal"},
		{Alias: "gemini-2.5-flash", Provider: "vertex", ProviderModel: "gemini-2.5-flash-preview-05-20", ModelType: "llm", Modalities: []string{"text", "image", "audio", "video"}, ContextWindow: 1048576, MaxOutputTokens: 65536, SupportsTools: true, PriceInput: 0.15, PriceOutput: 0.60, Category: "Google", Description: "Gemini 2.5 Flash fast/cheap"},
		{Alias: "gemini-2.0-flash", Provider: "vertex", ProviderModel: "gemini-2.0-flash", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 1048576, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 0.10, PriceOutput: 0.40, Category: "Google", Description: "Gemini 2.0 Flash"},

		// --- AWS Bedrock (Claude) ---
		{Alias: "bedrock-claude-sonnet-4", Provider: "bedrock", ProviderModel: "us.anthropic.claude-sonnet-4-20250514-v1:0", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 64000, SupportsTools: true, PriceInput: 3.00, PriceOutput: 15.00, Category: "AWS Bedrock", Description: "Claude Sonnet 4 via Bedrock"},
		{Alias: "bedrock-claude-3.5-sonnet", Provider: "bedrock", ProviderModel: "us.anthropic.claude-3-5-sonnet-20241022-v2:0", ModelType: "llm", Modalities: []string{"text", "image"}, ContextWindow: 200000, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 3.00, PriceOutput: 15.00, Category: "AWS Bedrock", Description: "Claude 3.5 Sonnet via Bedrock"},
		{Alias: "bedrock-claude-3.5-haiku", Provider: "bedrock", ProviderModel: "us.anthropic.claude-3-5-haiku-20241022-v1:0", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 200000, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 0.80, PriceOutput: 4.00, Category: "AWS Bedrock", Description: "Claude 3.5 Haiku via Bedrock"},

		// --- Groq ---
		{Alias: "groq-llama-3.3-70b", Provider: "groq", ProviderModel: "llama-3.3-70b-versatile", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 128000, MaxOutputTokens: 32768, SupportsTools: true, PriceInput: 0.59, PriceOutput: 0.79, Category: "Groq", Description: "Llama 3.3 70B on Groq"},
		{Alias: "groq-llama-3.1-8b", Provider: "groq", ProviderModel: "llama-3.1-8b-instant", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 128000, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 0.05, PriceOutput: 0.08, Category: "Groq", Description: "Llama 3.1 8B instant on Groq"},
		{Alias: "groq-gemma2-9b", Provider: "groq", ProviderModel: "gemma2-9b-it", ModelType: "llm", Modalities: []string{"text"}, ContextWindow: 8192, MaxOutputTokens: 8192, SupportsTools: true, PriceInput: 0.20, PriceOutput: 0.20, Category: "Groq", Description: "Gemma 2 9B on Groq"},
	}
}

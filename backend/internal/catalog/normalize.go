package catalog

import (
	"fmt"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// NormalizeWarning describes a non-fatal issue found during normalization.
type NormalizeWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (w NormalizeWarning) String() string {
	return fmt.Sprintf("%s: %s", w.Field, w.Message)
}

// NormalizeResult holds the normalized entry plus any warnings.
type NormalizeResult struct {
	Entry    config.ModelCatalogEntry `json:"entry"`
	Warnings []NormalizeWarning       `json:"warnings,omitempty"`
}

// NormalizeEntry applies canonical defaults and provider-aware normalization to
// a model catalog entry. It is used by both config loading and the admin API
// upsert path so behaviour is identical regardless of how the entry arrives.
//
// The function never mutates its input; it returns a new entry.
func NormalizeEntry(entry config.ModelCatalogEntry) NormalizeResult {
	var warnings []NormalizeWarning
	out := entry

	// --- alias ---
	out.Alias = strings.TrimSpace(out.Alias)

	// --- provider ---
	out.Provider = NormalizeProviderSlug(out.Provider)

	// --- provider_model ---
	out.ProviderModel = strings.TrimSpace(out.ProviderModel)

	// --- model_type (smart default from provider model when missing) ---
	out.ModelType = NormalizeModelType(out.ModelType)
	if out.ModelType == "" {
		out.ModelType = inferModelType(out.Provider, out.ProviderModel, out.Modalities)
	}

	// --- modalities (infer from model_type when empty) ---
	if len(out.Modalities) == 0 {
		out.Modalities = inferModalities(out.ModelType)
		if len(out.Modalities) > 0 {
			warnings = append(warnings, NormalizeWarning{
				Field:   "modalities",
				Message: fmt.Sprintf("auto-set to %v based on model_type=%q", out.Modalities, out.ModelType),
			})
		}
	}

	// --- supports_tools (auto-detect for common LLM models) ---
	// Only set if the incoming value is false and we can infer it should be true.
	if !out.SupportsTools && out.ModelType == "llm" {
		if inferToolSupport(out.Provider, out.ProviderModel) {
			out.SupportsTools = true
			warnings = append(warnings, NormalizeWarning{
				Field:   "supports_tools",
				Message: "auto-enabled based on provider and model",
			})
		}
	}

	// --- deployment (default to provider_model) ---
	out.Deployment = strings.TrimSpace(out.Deployment)
	if out.Deployment == "" && out.ProviderModel != "" {
		out.Deployment = out.ProviderModel
	}

	// --- weight ---
	if out.Weight == 0 {
		out.Weight = 100
	}

	// --- currency ---
	out.Currency = strings.ToUpper(strings.TrimSpace(out.Currency))
	if out.Currency == "" {
		out.Currency = "USD"
	}

	// --- enabled (nil means true) ---
	if out.Enabled == nil {
		t := true
		out.Enabled = &t
	}

	// --- metadata ---
	if out.Metadata == nil {
		out.Metadata = map[string]string{}
	}

	// --- pricing tiers ---
	if out.PricingTiers == nil {
		out.PricingTiers = config.PricingTiers{}
	}

	// --- pricing sanity warnings ---
	if out.PriceInput < 0 {
		out.PriceInput = 0
		warnings = append(warnings, NormalizeWarning{
			Field:   "price_input",
			Message: "negative value clamped to 0",
		})
	}
	if out.PriceOutput < 0 {
		out.PriceOutput = 0
		warnings = append(warnings, NormalizeWarning{
			Field:   "price_output",
			Message: "negative value clamped to 0",
		})
	}

	// Warn when pricing is absent for model types that should have it
	if out.PriceInput == 0 && out.PriceOutput == 0 && len(out.PricingTiers) == 0 {
		switch out.ModelType {
		case "llm", "embedding":
			warnings = append(warnings, NormalizeWarning{
				Field:   "pricing",
				Message: fmt.Sprintf("no pricing configured for %s model; cost tracking will report $0", out.ModelType),
			})
		case "image":
			warnings = append(warnings, NormalizeWarning{
				Field:   "pricing",
				Message: "no pricing tiers configured for image model; add per_image tiers for accurate cost tracking",
			})
		}
	}

	// --- provider-specific normalization ---
	normalizeProviderFields(&out, &warnings)

	return NormalizeResult{Entry: out, Warnings: warnings}
}

// normalizeProviderFields applies provider-specific field coercion.
func normalizeProviderFields(entry *config.ModelCatalogEntry, warnings *[]NormalizeWarning) {
	entry.Endpoint = strings.TrimSpace(entry.Endpoint)
	entry.APIKey = strings.TrimSpace(entry.APIKey)
	entry.APIVersion = strings.TrimSpace(entry.APIVersion)
	entry.Region = strings.TrimSpace(entry.Region)

	switch entry.Provider {
	case "azure":
		if az := entry.ProviderOverrides.Azure; az != nil {
			if entry.Deployment == entry.ProviderModel && strings.TrimSpace(az.Deployment) != "" {
				entry.Deployment = strings.TrimSpace(az.Deployment)
			}
			if entry.Endpoint == "" && strings.TrimSpace(az.Endpoint) != "" {
				entry.Endpoint = strings.TrimSpace(az.Endpoint)
			}
			if entry.APIKey == "" && strings.TrimSpace(az.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(az.APIKey)
			}
			if entry.APIVersion == "" && strings.TrimSpace(az.APIVersion) != "" {
				entry.APIVersion = strings.TrimSpace(az.APIVersion)
			}
			if entry.Region == "" && strings.TrimSpace(az.Region) != "" {
				entry.Region = strings.TrimSpace(az.Region)
			}
		}

	case "openai":
		if cfg := entry.ProviderOverrides.OpenAI; cfg != nil {
			if entry.Endpoint == "" && strings.TrimSpace(cfg.BaseURL) != "" {
				entry.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}

	case "openai-compatible":
		if cfg := entry.ProviderOverrides.OpenAICompatible; cfg != nil {
			if entry.Endpoint == "" && strings.TrimSpace(cfg.BaseURL) != "" {
				entry.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}

	case "openrouter":
		if cfg := entry.ProviderOverrides.OpenRouter; cfg != nil {
			if entry.Endpoint == "" && strings.TrimSpace(cfg.BaseURL) != "" {
				entry.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}

	case "groq":
		if cfg := entry.ProviderOverrides.Groq; cfg != nil {
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
			if entry.Region == "" && strings.TrimSpace(cfg.Region) != "" {
				entry.Region = strings.TrimSpace(cfg.Region)
			}
		}

	case "vllm":
		if cfg := entry.ProviderOverrides.VLLM; cfg != nil {
			if entry.Endpoint == "" && strings.TrimSpace(cfg.BaseURL) != "" {
				entry.Endpoint = strings.TrimSpace(cfg.BaseURL)
			}
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}

	case "bedrock":
		if cfg := entry.ProviderOverrides.Bedrock; cfg != nil {
			if entry.Region == "" && strings.TrimSpace(cfg.Region) != "" {
				entry.Region = strings.TrimSpace(cfg.Region)
			}
		}

	case "vertex":
		if cfg := entry.ProviderOverrides.Vertex; cfg != nil {
			if entry.Region == "" && strings.TrimSpace(cfg.Location) != "" {
				entry.Region = strings.TrimSpace(cfg.Location)
			}
		}

	case "anthropic":
		if cfg := entry.ProviderOverrides.Anthropic; cfg != nil {
			if entry.APIKey == "" && strings.TrimSpace(cfg.APIKey) != "" {
				entry.APIKey = strings.TrimSpace(cfg.APIKey)
			}
		}
	}
}

// inferModelType guesses model_type from provider model name and modalities.
func inferModelType(provider, providerModel string, modalities []string) string {
	lower := strings.ToLower(providerModel)

	// Check modalities first
	for _, m := range modalities {
		switch strings.ToLower(m) {
		case "embedding", "embeddings":
			return "embedding"
		case "image":
			return "image"
		case "audio":
			if strings.Contains(lower, "tts") || strings.Contains(lower, "speech") {
				return "audio_speech"
			}
			if strings.Contains(lower, "transcri") || strings.Contains(lower, "whisper") {
				return "audio_transcription"
			}
		}
	}

	// Heuristic from model name
	switch {
	case strings.Contains(lower, "embed"):
		return "embedding"
	case strings.Contains(lower, "image") || strings.Contains(lower, "imagen") ||
		strings.Contains(lower, "dall") || strings.Contains(lower, "stable-diffusion") ||
		strings.Contains(lower, "titan-image") || strings.Contains(lower, "flux"):
		return "image"
	case strings.Contains(lower, "tts") || strings.Contains(lower, "speech"):
		return "audio_speech"
	case strings.Contains(lower, "transcri") || strings.Contains(lower, "whisper"):
		return "audio_transcription"
	case strings.Contains(lower, "moderation"):
		return "moderation"
	}

	return "llm"
}

// inferModalities returns sensible modalities given a model type.
func inferModalities(modelType string) []string {
	switch modelType {
	case "llm":
		return []string{"text"}
	case "embedding":
		return []string{"embedding"}
	case "image":
		return []string{"image"}
	case "audio_transcription", "audio_speech":
		return []string{"audio"}
	case "moderation":
		return []string{"text"}
	case "video":
		return []string{"video"}
	default:
		return nil
	}
}

// inferToolSupport returns true for known LLM models that support tool calling.
func inferToolSupport(provider, providerModel string) bool {
	lower := strings.ToLower(providerModel)

	// Models that definitely support tools
	toolModels := []string{
		"gpt-4", "gpt-3.5-turbo", "gpt-4o",
		"claude-3", "claude-4",
		"gemini",
		"llama-3", "llama3",
		"mixtral",
		"qwen",
		"command-r",
	}
	for _, prefix := range toolModels {
		if strings.Contains(lower, prefix) {
			return true
		}
	}

	// Provider-wide defaults
	switch provider {
	case "openai":
		return true // all OpenAI chat models support tools
	}

	return false
}

package catalog

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestNormalizeEntry_MinimalLLM(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "gpt-4o",
		Provider:      "openai",
		ProviderModel: "gpt-4o",
	})
	e := result.Entry
	if e.ModelType != "llm" {
		t.Errorf("expected model_type=llm, got %q", e.ModelType)
	}
	if len(e.Modalities) == 0 || e.Modalities[0] != "text" {
		t.Errorf("expected modalities=[text], got %v", e.Modalities)
	}
	if !e.SupportsTools {
		t.Error("expected supports_tools=true for gpt-4o")
	}
	if e.Deployment != "gpt-4o" {
		t.Errorf("expected deployment=gpt-4o, got %q", e.Deployment)
	}
	if e.Weight != 100 {
		t.Errorf("expected weight=100, got %d", e.Weight)
	}
	if e.Currency != "USD" {
		t.Errorf("expected currency=USD, got %q", e.Currency)
	}
	if !e.IsEnabled() {
		t.Error("expected enabled=true by default")
	}
}

func TestNormalizeEntry_InfersEmbedding(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "text-embedding-3-small",
		Provider:      "openai",
		ProviderModel: "text-embedding-3-small",
	})
	if result.Entry.ModelType != "embedding" {
		t.Errorf("expected model_type=embedding, got %q", result.Entry.ModelType)
	}
	if len(result.Entry.Modalities) == 0 || result.Entry.Modalities[0] != "embedding" {
		t.Errorf("expected modalities=[embedding], got %v", result.Entry.Modalities)
	}
}

func TestNormalizeEntry_InfersImage(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "gpt-image-1",
		Provider:      "openai",
		ProviderModel: "gpt-image-1",
	})
	if result.Entry.ModelType != "image" {
		t.Errorf("expected model_type=image, got %q", result.Entry.ModelType)
	}
}

func TestNormalizeEntry_InfersAudioSpeech(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "gpt-4o-mini-tts",
		Provider:      "openai",
		ProviderModel: "gpt-4o-mini-tts",
	})
	if result.Entry.ModelType != "audio_speech" {
		t.Errorf("expected model_type=audio_speech, got %q", result.Entry.ModelType)
	}
}

func TestNormalizeEntry_NegativePriceClamped(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "test",
		Provider:      "openai",
		ProviderModel: "test",
		PriceInput:    -1.0,
		PriceOutput:   -2.0,
	})
	if result.Entry.PriceInput != 0 {
		t.Errorf("expected price_input=0, got %f", result.Entry.PriceInput)
	}
	if result.Entry.PriceOutput != 0 {
		t.Errorf("expected price_output=0, got %f", result.Entry.PriceOutput)
	}
	if len(result.Warnings) < 2 {
		t.Error("expected warnings for negative prices")
	}
}

func TestNormalizeEntry_AzureOverrideMerge(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "gpt-4o-azure",
		Provider:      "azure",
		ProviderModel: "gpt-4o",
		ProviderOverrides: config.ProviderOverrides{
			Azure: &config.AzureProviderConfig{
				Deployment: "gpt4o-east",
				Endpoint:   "https://myres.openai.azure.com",
				APIKey:     "key123",
				APIVersion: "2024-07-01",
				Region:     "eastus",
			},
		},
	})
	e := result.Entry
	if e.Deployment != "gpt4o-east" {
		t.Errorf("expected deployment from azure override, got %q", e.Deployment)
	}
	if e.Endpoint != "https://myres.openai.azure.com" {
		t.Errorf("expected endpoint from azure override, got %q", e.Endpoint)
	}
	if e.APIKey != "key123" {
		t.Errorf("expected api_key from azure override")
	}
	if e.Region != "eastus" {
		t.Errorf("expected region from azure override, got %q", e.Region)
	}
}

func TestNormalizeEntry_ProviderSlugNormalization(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "test",
		Provider:      "OpenAI_Compatible",
		ProviderModel: "test",
	})
	if result.Entry.Provider != "openai-compatible" {
		t.Errorf("expected openai-compatible, got %q", result.Entry.Provider)
	}
}

func TestNormalizeEntry_PricingWarningForLLM(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "test",
		Provider:      "openai",
		ProviderModel: "gpt-4o",
	})
	hasPricingWarning := false
	for _, w := range result.Warnings {
		if w.Field == "pricing" {
			hasPricingWarning = true
			break
		}
	}
	if !hasPricingWarning {
		t.Error("expected pricing warning for LLM without pricing")
	}
}

func TestNormalizeEntry_NoPricingWarningWhenSet(t *testing.T) {
	result := NormalizeEntry(config.ModelCatalogEntry{
		Alias:         "test",
		Provider:      "openai",
		ProviderModel: "gpt-4o",
		PriceInput:    2.5,
		PriceOutput:   10.0,
	})
	for _, w := range result.Warnings {
		if w.Field == "pricing" {
			t.Error("did not expect pricing warning when prices are set")
		}
	}
}

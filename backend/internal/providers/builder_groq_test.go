package providers

import (
	"context"
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestBuildGroqRoute(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProviderConfig{
			Groq: config.GroqProviderConfig{
				APIKey: "test",
			},
		},
	}
	entry := config.ModelCatalogEntry{
		Alias:         "groq-llama",
		Provider:      "groq",
		ProviderModel: "llama-3.3-70b-versatile",
		ModelType:     "llm",
		Metadata: map[string]string{
			"groq_region": "us-east-1",
		},
	}
	route, err := buildGroqRoute(context.Background(), cfg, entry)
	if err != nil {
		t.Fatalf("build groq route: %v", err)
	}
	if route.Metadata["groq_region"] != "us-east-1" {
		t.Fatalf("expected region metadata, got %+v", route.Metadata)
	}
	if route.Chat == nil || route.ChatStream == nil {
		t.Fatalf("expected chat capabilities")
	}
}

func TestBuildGroqRouteMissingKey(t *testing.T) {
	entry := config.ModelCatalogEntry{
		Alias:         "groq-llama",
		Provider:      "groq",
		ProviderModel: "llama-3.3-70b-versatile",
	}
	if _, err := buildGroqRoute(context.Background(), &config.Config{}, entry); err == nil {
		t.Fatalf("expected error when api key missing")
	}
}

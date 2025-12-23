package providers

import (
	"context"
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestBuildVLLMRouteOpenAIAllowsNoKey(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProviderConfig{
			VLLM: config.VLLMProviderConfig{
				BaseURL: "http://localhost:8000/v1",
				Mode:    "openai",
			},
		},
	}
	entry := config.ModelCatalogEntry{
		Alias:         "vllm-openai",
		Provider:      "vllm",
		ProviderModel: "llama-3",
		Modalities:    []string{"text"},
	}
	route, err := buildVLLMRoute(context.Background(), cfg, entry)
	if err != nil {
		t.Fatalf("build vllm route: %v", err)
	}
	if route.Chat == nil || route.ChatStream == nil {
		t.Fatalf("expected chat capabilities")
	}
	if route.Metadata["base_url"] != "http://localhost:8000/v1" {
		t.Fatalf("expected base_url metadata")
	}
}

func TestBuildVLLMRouteTGIAuthHeader(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProviderConfig{
			VLLM: config.VLLMProviderConfig{
				BaseURL:    "http://localhost:8080",
				Mode:       "tgi",
				AuthHeader: "X-Api-Key",
			},
		},
	}
	entry := config.ModelCatalogEntry{
		Alias:         "vllm-tgi",
		Provider:      "vllm",
		ProviderModel: "llama-3",
		Modalities:    []string{"text"},
	}
	route, err := buildVLLMRoute(context.Background(), cfg, entry)
	if err != nil {
		t.Fatalf("build vllm route: %v", err)
	}
	if route.Metadata["auth_header"] != "X-Api-Key" {
		t.Fatalf("expected auth_header metadata")
	}
	if route.Chat == nil || route.ChatStream == nil {
		t.Fatalf("expected chat capabilities")
	}
}

func TestBuildVLLMRouteInvalidMode(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProviderConfig{
			VLLM: config.VLLMProviderConfig{
				BaseURL: "http://localhost:8000",
				Mode:    "bad-mode",
			},
		},
	}
	entry := config.ModelCatalogEntry{
		Alias:         "vllm-invalid",
		Provider:      "vllm",
		ProviderModel: "llama-3",
		Modalities:    []string{"text"},
	}
	if _, err := buildVLLMRoute(context.Background(), cfg, entry); err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}

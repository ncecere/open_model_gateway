package providers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	providerfixtures "github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

type catalogFixturePayload struct {
	ConfigPath string                     `json:"config_path"`
	Entries    []config.ModelCatalogEntry `json:"entries"`
}

func TestCatalogFixtureBuildsRoutes(t *testing.T) {
	data, err := providerfixtures.Read("provider_catalog.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var payload catalogFixturePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if payload.ConfigPath == "" {
		t.Fatalf("fixture missing config path")
	}

	cfg, err := config.Load(config.Options{ConfigFile: payload.ConfigPath})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ModelCatalog = payload.Entries

	factory := NewFactory(cfg)
	routes, err := factory.Build(context.Background())
	if err != nil {
		t.Fatalf("build routes: %v", err)
	}
	if len(routes) == 0 {
		t.Fatalf("expected at least one route from fixture")
	}

	for alias, rs := range routes {
		for _, route := range rs {
			if route.Retry.MaxAttempts <= 0 {
				t.Fatalf("alias %s retry max attempts unset", alias)
			}
			if route.Retry.BackoffMultiplier <= 0 {
				t.Fatalf("alias %s retry multiplier unset", alias)
			}
			if strings.TrimSpace(route.Tokenizer) == "" {
				t.Fatalf("alias %s tokenizer missing", alias)
			}
		}
	}
}

package providers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

	configPath := payload.ConfigPath
	if _, err := os.Stat(configPath); err != nil {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			t.Fatalf("working dir: %v", cwdErr)
		}
		repoRoot := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
		trimmed := payload.ConfigPath
		for strings.HasPrefix(trimmed, "../") {
			trimmed = strings.TrimPrefix(trimmed, "../")
		}
		relativeFromRoot := filepath.Clean(filepath.Join(repoRoot, trimmed))
		if _, altErr := os.Stat(relativeFromRoot); altErr == nil {
			configPath = relativeFromRoot
		} else {
			t.Fatalf("fixture config path not found: %s (%v)", payload.ConfigPath, err)
		}
	}

	cfg, err := config.Load(config.Options{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	filtered := make([]config.ModelCatalogEntry, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		if strings.EqualFold(entry.Provider, "vertex") {
			continue
		}
		filtered = append(filtered, entry)
	}
	cfg.ModelCatalog = filtered

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

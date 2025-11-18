package providers

import (
	"context"
	"encoding/json"
	"fmt"
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
		path, resolveErr := resolveConfigPath(payload.ConfigPath)
		if resolveErr != nil {
			t.Fatalf("fixture config path not found: %s (%v)", payload.ConfigPath, resolveErr)
		}
		configPath = path
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

func resolveConfigPath(relative string) (string, error) {
	if relative == "" {
		return "", fmt.Errorf("empty config path")
	}
	if filepath.IsAbs(relative) {
		if _, err := os.Stat(relative); err == nil {
			return relative, nil
		}
		return "", fmt.Errorf("absolute path not found: %s", relative)
	}

	start, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := start
	for {
		candidate := filepath.Clean(filepath.Join(dir, relative))
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("unable to resolve path %s from %s", relative, start)
}

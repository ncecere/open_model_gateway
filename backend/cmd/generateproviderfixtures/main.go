package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	"github.com/ncecere/open_model_gateway/backend/internal/runtime"
)

type catalogFixture struct {
	GeneratedAt time.Time                  `json:"generated_at"`
	Source      string                     `json:"source"`
	ConfigPath  string                     `json:"config_path"`
	Entries     []config.ModelCatalogEntry `json:"entries"`
}

func main() {
	var (
		output    = flag.String("output", "internal/providers/fixtures/testdata/provider_catalog.json", "output path for generated fixture")
		configArg = flag.String("config", "", "path to router config (defaults to $ROUTER_CONFIG_FILE or deploy/router.local.yaml)")
		skipDB    = flag.Bool("skip-db", false, "skip querying Postgres; only use static config entries")
	)
	flag.Parse()

	ctx := context.Background()
	cfgPath := resolveConfigPath(*configArg)
	absPath, err := filepath.Abs(cfgPath)
	if err != nil {
		log.Fatalf("resolve config path: %v", err)
	}
	cfgOpts := config.Options{ConfigFile: absPath}

	var entries []config.ModelCatalogEntry
	source := "config"

	if *skipDB {
		cfg, err := config.Load(cfgOpts)
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		var emptyRows []db.ModelCatalog
		entries, err = router.MergeEntries(cfg.ModelCatalog, emptyRows)
		if err != nil {
			log.Fatalf("merge entries: %v", err)
		}
	} else {
		rt, err := runtime.New(ctx, runtime.Options{Config: cfgOpts, SkipMigrations: true})
		if err != nil {
			log.Fatalf("init runtime: %v", err)
		}
		defer func() {
			_ = rt.Shutdown(ctx)
		}()

		rows, err := rt.Container.Queries.ListModelCatalog(ctx)
		if err != nil {
			log.Fatalf("list catalog: %v", err)
		}
		entries, err = router.MergeEntries(rt.Config.ModelCatalog, rows)
		if err != nil {
			log.Fatalf("merge entries: %v", err)
		}
		source = "config+db"
	}

	payload := catalogFixture{
		GeneratedAt: time.Now().UTC(),
		Source:      source,
		ConfigPath:  absPath,
		Entries:     entries,
	}

	if err := writeFixture(*output, payload); err != nil {
		log.Fatalf("write fixture: %v", err)
	}
	log.Printf("wrote %d catalog entries to %s", len(entries), *output)
}

func resolveConfigPath(flagValue string) string {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed
	}
	if env := strings.TrimSpace(os.Getenv("ROUTER_CONFIG_FILE")); env != "" {
		return env
	}
	defaultPath := filepath.Clean("deploy/router.local.yaml")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	alt := filepath.Clean(filepath.Join("..", "deploy", "router.local.yaml"))
	return alt
}

func writeFixture(path string, payload catalogFixture) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create fixture dir: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

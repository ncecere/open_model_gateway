package app

import (
	"context"
	"fmt"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	runtimecatalog "github.com/ncecere/open_model_gateway/backend/internal/runtime/catalog"
)

// Routing captures provider routing artifacts (entries, factory, engine).
type Routing struct {
	Entries []config.ModelCatalogEntry
	Factory *providers.Factory
	Engine  *router.Engine
}

// BuildRouting hydrates the model catalog, persists entries, and prepares the
// provider factory + router engine.
func BuildRouting(ctx context.Context, cfg *config.Config, queries *db.Queries) (*Routing, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	dbEntries, err := queries.ListModelCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("load model catalog: %w", err)
	}
	entries, err := router.MergeEntries(cfg.ModelCatalog, dbEntries)
	if err != nil {
		return nil, fmt.Errorf("merge model catalog: %w", err)
	}
	override := *cfg
	override.ModelCatalog = entries
	factory := providers.NewFactory(&override)
	engine := router.NewEngine()
	if err := engine.Reload(ctx, factory); err != nil {
		return nil, fmt.Errorf("init router engine: %w", err)
	}
	if err := runtimecatalog.EnsurePersisted(ctx, queries, entries); err != nil {
		return nil, err
	}
	return &Routing{Entries: entries, Factory: factory, Engine: engine}, nil
}

package providers

import (
	"context"
	"fmt"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// Builder constructs a provider Route for a catalog entry.
type Builder func(ctx context.Context, cfg *config.Config, entry config.ModelCatalogEntry) (Route, error)

// Factory builds provider routes from configuration using a registry of builders.
type Factory struct {
	cfg      *config.Config
	builders map[string]Builder
}

// NewFactory creates a factory with the default provider registry.
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{cfg: cfg, builders: cloneDefaultBuilders()}
}

// Register allows tests or callers to override provider builders.
func (f *Factory) Register(name string, builder Builder) {
	if f.builders == nil {
		f.builders = make(map[string]Builder)
	}
	f.builders[name] = builder
}

// Build iterates over model catalog entries and instantiates provider adapters.
func (f *Factory) Build(ctx context.Context) (map[string][]Route, error) {
	routes := make(map[string][]Route)
	for _, entry := range f.cfg.ModelCatalog {
		if !entry.IsEnabled() {
			continue
		}
		builder, ok := f.builders[entry.Provider]
		if !ok {
			return nil, fmt.Errorf("alias %q: provider %q unsupported", entry.Alias, entry.Provider)
		}
		route, err := builder(ctx, f.cfg, entry)
		if err != nil {
			return nil, fmt.Errorf("alias %q: %w", entry.Alias, err)
		}
		routes[entry.Alias] = append(routes[entry.Alias], route)
	}
	return routes, nil
}

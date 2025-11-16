package adminprovider

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	openrouter "github.com/ncecere/open_model_gateway/backend/internal/adapters/openrouter"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

// Service exposes provider registry metadata and discovery helpers for admin consumers.
type Service struct {
	cfg *config.Config

	adapterOnce sync.Once
	adapter     *openrouter.Adapter
	adapterErr  error

	cacheMu sync.RWMutex
	cache   OpenRouterCatalog
}

// Definition describes a registered provider adapter.
type Definition struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

// OpenRouterCatalog captures cached discovery results.
type OpenRouterCatalog struct {
	RefreshedAt time.Time                 `json:"refreshed_at"`
	ExpiresAt   time.Time                 `json:"expires_at"`
	Models      []openrouter.CatalogModel `json:"models"`
}

// NewService returns a provider metadata service.
func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// List returns all registered providers sorted by name.
func (s *Service) List(ctx context.Context) ([]Definition, error) {
	defs := providers.DefaultDefinitions()
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		caps := append([]string(nil), def.Capabilities...)
		sort.Strings(caps)
		out = append(out, Definition{
			Name:         def.Name,
			Description:  def.Description,
			Capabilities: caps,
		})
	}
	return out, nil
}

// OpenRouterCatalog returns the cached OpenRouter catalog, refreshing as needed.
func (s *Service) OpenRouterCatalog(ctx context.Context) (OpenRouterCatalog, error) {
	if s == nil {
		return OpenRouterCatalog{}, errors.New("provider service unavailable")
	}
	ttl := s.catalogTTL()
	s.cacheMu.RLock()
	cached := s.cache
	if len(cached.Models) > 0 && time.Until(cached.ExpiresAt) > 0 {
		s.cacheMu.RUnlock()
		return cached, nil
	}
	s.cacheMu.RUnlock()

	adapter, err := s.openRouterAdapter()
	if err != nil {
		return OpenRouterCatalog{}, err
	}
	models, err := adapter.Catalog(ctx)
	if err != nil {
		return OpenRouterCatalog{}, err
	}

	refreshed := time.Now().UTC()
	catalog := OpenRouterCatalog{
		RefreshedAt: refreshed,
		ExpiresAt:   refreshed.Add(ttl),
		Models:      models,
	}

	s.cacheMu.Lock()
	s.cache = catalog
	s.cacheMu.Unlock()
	return catalog, nil
}

func (s *Service) openRouterAdapter() (*openrouter.Adapter, error) {
	s.adapterOnce.Do(func() {
		if s.cfg == nil {
			s.adapterErr = errors.New("openrouter catalog: config not initialized")
			return
		}
		apiKey := strings.TrimSpace(s.cfg.Providers.OpenRouter.APIKey)
		if apiKey == "" {
			s.adapterErr = errors.New("openrouter catalog: providers.openrouter.api_key is required")
			return
		}
		opts := openrouter.Options{
			APIKey:  apiKey,
			BaseURL: s.cfg.Providers.OpenRouter.BaseURL,
			Referer: s.cfg.Providers.OpenRouter.Referer,
			AppName: s.cfg.Providers.OpenRouter.AppName,
		}
		s.adapter, s.adapterErr = openrouter.New(opts)
	})
	return s.adapter, s.adapterErr
}

func (s *Service) catalogTTL() time.Duration {
	if s == nil || s.cfg == nil {
		return 10 * time.Minute
	}
	ttl := s.cfg.Providers.OpenRouter.ModelsCacheTTL
	if ttl <= 0 {
		return 10 * time.Minute
	}
	return ttl
}

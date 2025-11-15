package router

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

type Engine struct {
	mu     sync.RWMutex
	routes map[string][]providers.Route
	state  map[string]*routeState
}

type routeState struct {
	consecutiveFailures int
	openUntil           time.Time
}

const (
	failureThreshold = 3
	openDuration     = time.Minute
)

func NewEngine() *Engine {
	return &Engine{
		routes: make(map[string][]providers.Route),
		state:  make(map[string]*routeState),
	}
}

func (e *Engine) Reload(ctx context.Context, factory *providers.Factory) error {
	routes, err := factory.Build(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	newState := make(map[string]*routeState, len(routes))
	for alias, rts := range routes {
		for _, route := range rts {
			key := routeKey(alias, route)
			if old, ok := e.state[key]; ok {
				newState[key] = old
			} else {
				newState[key] = &routeState{}
			}
		}
	}

	e.routes = routes
	e.state = newState
	return nil
}

func (e *Engine) SelectRoutes(alias string) []providers.Route {
	e.mu.RLock()
	defer e.mu.RUnlock()

	healthy := make([]providers.Route, 0)
	now := time.Now()
	for _, route := range e.routes[alias] {
		st := e.state[routeKey(alias, route)]
		if st == nil || st.openUntil.Before(now) {
			healthy = append(healthy, route)
		}
	}

	if len(healthy) <= 1 {
		return healthy
	}

	idx := weightedSelect(healthy)
	if idx != 0 {
		selected := healthy[idx]
		healthy[idx] = healthy[0]
		healthy[0] = selected
	}

	return healthy
}

func (e *Engine) ReportSuccess(alias string, route providers.Route) {
	e.mu.Lock()
	defer e.mu.Unlock()

	st := e.state[routeKey(alias, route)]
	if st == nil {
		st = &routeState{}
		e.state[routeKey(alias, route)] = st
	}
	st.consecutiveFailures = 0
	st.openUntil = time.Time{}
}

func (e *Engine) ReportFailure(alias string, route providers.Route) {
	e.mu.Lock()
	defer e.mu.Unlock()

	st := e.state[routeKey(alias, route)]
	if st == nil {
		st = &routeState{}
		e.state[routeKey(alias, route)] = st
	}

	st.consecutiveFailures++
	if st.consecutiveFailures >= failureThreshold {
		st.openUntil = time.Now().Add(openDuration)
	}
}

func weightedSelect(routes []providers.Route) int {
	total := 0
	for _, r := range routes {
		if r.Weight > 0 {
			total += r.Weight
		}
	}
	if total == 0 {
		return rand.Intn(len(routes))
	}
	draw := rand.Intn(total)
	sum := 0
	for idx, r := range routes {
		weight := r.Weight
		if weight <= 0 {
			weight = 1
		}
		sum += weight
		if draw < sum {
			return idx
		}
	}
	return 0
}

func routeKey(alias string, route providers.Route) string {
	deployment := route.Metadata["deployment"]
	if deployment == "" {
		deployment = route.Model
	}
	return alias + "::" + deployment
}

// ListAliases returns the set of configured aliases and their routes.
func (e *Engine) ListAliases() map[string][]providers.Route {
	e.mu.RLock()
	defer e.mu.RUnlock()

	copyMap := make(map[string][]providers.Route, len(e.routes))
	for alias, routes := range e.routes {
		out := make([]providers.Route, len(routes))
		copy(out, routes)
		copyMap[alias] = out
	}
	return copyMap
}

func MergeEntries(cfgEntries []config.ModelCatalogEntry, dbEntries []db.ModelCatalog) ([]config.ModelCatalogEntry, error) {
	merged := make(map[string]config.ModelCatalogEntry)
	for _, entry := range cfgEntries {
		entry.Provider = catalog.NormalizeProviderSlug(entry.Provider)
		if strings.TrimSpace(entry.ModelType) == "" {
			entry.ModelType = "llm"
		}
		merged[entry.Alias] = entry
	}

	for _, row := range dbEntries {
		enabled := row.Enabled
		entry := config.ModelCatalogEntry{
			Alias:           row.Alias,
			Provider:        catalog.NormalizeProviderSlug(row.Provider),
			ProviderModel:   row.ProviderModel,
			ModelType:       row.ModelType,
			ContextWindow:   row.ContextWindow,
			MaxOutputTokens: row.MaxOutputTokens,
			SupportsTools:   row.SupportsTools,
			PriceInput:      row.PriceInput.InexactFloat64(),
			PriceOutput:     row.PriceOutput.InexactFloat64(),
			Currency:        row.Currency,
			Enabled:         &enabled,
			Deployment:      row.Deployment,
			Endpoint:        row.Endpoint,
			APIKey:          row.ApiKey,
			APIVersion:      row.ApiVersion,
			Region:          row.Region,
			Weight:          int(row.Weight),
			Metadata:        map[string]string{},
		}

		if len(row.ModalitiesJson) > 0 {
			if err := json.Unmarshal(row.ModalitiesJson, &entry.Modalities); err != nil {
				return nil, err
			}
		}
		if len(row.MetadataJson) > 0 {
			if err := json.Unmarshal(row.MetadataJson, &entry.Metadata); err != nil {
				return nil, err
			}
		}
		if len(row.ProviderConfigJson) > 0 {
			if err := json.Unmarshal(row.ProviderConfigJson, &entry.ProviderOverrides); err != nil {
				return nil, err
			}
		}

		merged[entry.Alias] = entry
	}

	out := make([]config.ModelCatalogEntry, 0, len(merged))
	for _, v := range merged {
		out = append(out, v)
	}
	return out, nil
}

func BuildFactory(cfg *config.Config, entries []config.ModelCatalogEntry) (*providers.Factory, error) {
	if cfg == nil {
		return nil, errors.New("config required")
	}
	override := *cfg
	override.ModelCatalog = entries
	return providers.NewFactory(&override), nil
}

package containers

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/health"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	runtimetype "github.com/ncecere/open_model_gateway/backend/internal/runtime/tenants"
)

// RoutingContainer holds provider routing artifacts.
type RoutingContainer struct {
	Factory   *providers.Factory
	Engine    *router.Engine
	HealthMon *health.Monitor

	tenantModels   *runtimetype.AccessStore
	tenantModelsMu sync.RWMutex
}

// NewRoutingContainer creates a new RoutingContainer.
func NewRoutingContainer(factory *providers.Factory, engine *router.Engine, healthMon *health.Monitor) *RoutingContainer {
	return &RoutingContainer{
		Factory:   factory,
		Engine:    engine,
		HealthMon: healthMon,
	}
}

// LoadTenantModelAccess loads tenant model access from the access store.
func (r *RoutingContainer) LoadTenantModelAccess(store *runtimetype.AccessStore) {
	r.tenantModelsMu.Lock()
	defer r.tenantModelsMu.Unlock()
	r.tenantModels = store
}

// IsModelAllowed checks if a tenant is allowed to access a model alias.
func (r *RoutingContainer) IsModelAllowed(tenantID uuid.UUID, alias string) bool {
	r.tenantModelsMu.RLock()
	defer r.tenantModelsMu.RUnlock()
	if r.tenantModels == nil {
		return true
	}
	return r.tenantModels.IsAllowed(tenantID, alias)
}

// SetTenantModels sets the allowed model aliases for a tenant.
func (r *RoutingContainer) SetTenantModels(tenantID uuid.UUID, aliases []string) {
	r.tenantModelsMu.Lock()
	defer r.tenantModelsMu.Unlock()
	if r.tenantModels == nil {
		r.tenantModels = runtimetype.NewAccessStore(nil)
	}
	r.tenantModels.Set(tenantID, aliases)
}

// ClearTenantModels clears the allowed model aliases for a tenant.
func (r *RoutingContainer) ClearTenantModels(tenantID uuid.UUID) {
	r.tenantModelsMu.Lock()
	defer r.tenantModelsMu.Unlock()
	if r.tenantModels != nil {
		r.tenantModels.Clear(tenantID)
	}
}

// Reload rebuilds provider routes after catalog changes.
func (r *RoutingContainer) Reload(ctx context.Context, factory *providers.Factory) error {
	r.Factory = factory
	return r.Engine.Reload(ctx, factory)
}

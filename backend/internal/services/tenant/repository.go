package tenant

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for tenant operations.
type Repository interface {
	ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error)
	GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error)
	GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error)
	GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error)
	SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error) {
	return a.q.ListUserTenants(ctx, userID)
}

func (a *queriesAdapter) GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error) {
	return a.q.GetTenantMembership(ctx, params)
}

func (a *queriesAdapter) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	return a.q.GetTenantByID(ctx, id)
}

func (a *queriesAdapter) GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error) {
	return a.q.GetTenantBudgetOverride(ctx, tenantID)
}

func (a *queriesAdapter) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	return a.q.SumUsageForTenant(ctx, params)
}

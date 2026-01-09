package adminbudget

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin budget operations.
type Repository interface {
	// Budget defaults operations
	GetBudgetDefaults(ctx context.Context) (db.BudgetDefault, error)
	UpsertBudgetDefaults(ctx context.Context, params db.UpsertBudgetDefaultsParams) (db.BudgetDefault, error)

	// Tenant budget override operations
	GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error)
	ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error)
	UpsertTenantBudgetOverride(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error)
	DeleteTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) error
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) GetBudgetDefaults(ctx context.Context) (db.BudgetDefault, error) {
	return a.q.GetBudgetDefaults(ctx)
}

func (a *queriesAdapter) UpsertBudgetDefaults(ctx context.Context, params db.UpsertBudgetDefaultsParams) (db.BudgetDefault, error) {
	return a.q.UpsertBudgetDefaults(ctx, params)
}

func (a *queriesAdapter) GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error) {
	return a.q.GetTenantBudgetOverride(ctx, tenantID)
}

func (a *queriesAdapter) ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error) {
	return a.q.ListTenantBudgetOverrides(ctx)
}

func (a *queriesAdapter) UpsertTenantBudgetOverride(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error) {
	return a.q.UpsertTenantBudgetOverride(ctx, params)
}

func (a *queriesAdapter) DeleteTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) error {
	return a.q.DeleteTenantBudgetOverride(ctx, tenantID)
}

package rbac

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for RBAC operations.
type Repository interface {
	GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error)
	ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error) {
	return a.q.GetTenantMembership(ctx, params)
}

func (a *queriesAdapter) ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error) {
	return a.q.ListUserTenants(ctx, userID)
}

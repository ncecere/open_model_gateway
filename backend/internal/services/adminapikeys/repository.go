package adminapikeys

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin API key operations.
type Repository interface {
	// Admin API key operations
	CreateAdminAPIKey(ctx context.Context, params db.CreateAdminAPIKeyParams) (db.AdminApiKey, error)
	GetAdminAPIKeyByID(ctx context.Context, id pgtype.UUID) (db.AdminApiKey, error)
	GetAdminAPIKeyByPrefix(ctx context.Context, prefix string) (db.AdminApiKey, error)
	ListAdminAPIKeys(ctx context.Context) ([]db.ListAdminAPIKeysRow, error)
	ListAdminAPIKeysByOwner(ctx context.Context, ownerUserID pgtype.UUID) ([]db.ListAdminAPIKeysByOwnerRow, error)
	RevokeAdminAPIKey(ctx context.Context, id pgtype.UUID) (db.AdminApiKey, error)
	UpdateAdminAPIKeyLastUsed(ctx context.Context, id pgtype.UUID) error

	// User operations (for authorization)
	GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) CreateAdminAPIKey(ctx context.Context, params db.CreateAdminAPIKeyParams) (db.AdminApiKey, error) {
	return a.q.CreateAdminAPIKey(ctx, params)
}

func (a *queriesAdapter) GetAdminAPIKeyByID(ctx context.Context, id pgtype.UUID) (db.AdminApiKey, error) {
	return a.q.GetAdminAPIKeyByID(ctx, id)
}

func (a *queriesAdapter) GetAdminAPIKeyByPrefix(ctx context.Context, prefix string) (db.AdminApiKey, error) {
	return a.q.GetAdminAPIKeyByPrefix(ctx, prefix)
}

func (a *queriesAdapter) ListAdminAPIKeys(ctx context.Context) ([]db.ListAdminAPIKeysRow, error) {
	return a.q.ListAdminAPIKeys(ctx)
}

func (a *queriesAdapter) ListAdminAPIKeysByOwner(ctx context.Context, ownerUserID pgtype.UUID) ([]db.ListAdminAPIKeysByOwnerRow, error) {
	return a.q.ListAdminAPIKeysByOwner(ctx, ownerUserID)
}

func (a *queriesAdapter) RevokeAdminAPIKey(ctx context.Context, id pgtype.UUID) (db.AdminApiKey, error) {
	return a.q.RevokeAdminAPIKey(ctx, id)
}

func (a *queriesAdapter) UpdateAdminAPIKeyLastUsed(ctx context.Context, id pgtype.UUID) error {
	return a.q.UpdateAdminAPIKeyLastUsed(ctx, id)
}

func (a *queriesAdapter) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return a.q.GetUserByID(ctx, id)
}

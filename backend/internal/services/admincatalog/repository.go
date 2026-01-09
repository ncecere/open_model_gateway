package admincatalog

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin catalog operations.
type Repository interface {
	ListModelCatalog(ctx context.Context) ([]db.ModelCatalog, error)
	UpsertModelCatalogEntry(ctx context.Context, params db.UpsertModelCatalogEntryParams) (db.ModelCatalog, error)
	DeleteModelCatalogEntry(ctx context.Context, alias string) error
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) ListModelCatalog(ctx context.Context) ([]db.ModelCatalog, error) {
	return a.q.ListModelCatalog(ctx)
}

func (a *queriesAdapter) UpsertModelCatalogEntry(ctx context.Context, params db.UpsertModelCatalogEntryParams) (db.ModelCatalog, error) {
	return a.q.UpsertModelCatalogEntry(ctx, params)
}

func (a *queriesAdapter) DeleteModelCatalogEntry(ctx context.Context, alias string) error {
	return a.q.DeleteModelCatalogEntry(ctx, alias)
}

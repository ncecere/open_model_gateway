package adminratelimit

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin rate limit operations.
type Repository interface {
	UpsertRateLimitDefaults(ctx context.Context, params db.UpsertRateLimitDefaultsParams) (db.RateLimitDefault, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) UpsertRateLimitDefaults(ctx context.Context, params db.UpsertRateLimitDefaultsParams) (db.RateLimitDefault, error) {
	return a.q.UpsertRateLimitDefaults(ctx, params)
}

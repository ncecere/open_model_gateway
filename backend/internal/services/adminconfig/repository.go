package adminconfig

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin config operations.
type Repository interface {
	UpsertSystemSetting(ctx context.Context, params db.UpsertSystemSettingParams) (db.SystemSetting, error)
	GetSystemSetting(ctx context.Context, key string) (db.SystemSetting, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) UpsertSystemSetting(ctx context.Context, params db.UpsertSystemSettingParams) (db.SystemSetting, error) {
	return a.q.UpsertSystemSetting(ctx, params)
}

func (a *queriesAdapter) GetSystemSetting(ctx context.Context, key string) (db.SystemSetting, error) {
	return a.q.GetSystemSetting(ctx, key)
}

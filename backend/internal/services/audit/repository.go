package audit

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for audit operations.
type Repository interface {
	ListAuditLogs(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error)
	InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) ListAuditLogs(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error) {
	return a.q.ListAuditLogs(ctx, params)
}

func (a *queriesAdapter) InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error) {
	return a.q.InsertAuditLog(ctx, params)
}

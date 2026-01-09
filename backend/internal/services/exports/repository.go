package exports

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for export operations.
type Repository interface {
	CreateUsageExport(ctx context.Context, params db.CreateUsageExportParams) (db.UsageExport, error)
	GetUsageExport(ctx context.Context, id pgtype.UUID) (db.UsageExport, error)
	ListUsageExportsAdmin(ctx context.Context, params db.ListUsageExportsAdminParams) ([]db.UsageExport, error)
	ListUsageExportsByRequester(ctx context.Context, params db.ListUsageExportsByRequesterParams) ([]db.UsageExport, error)
	ClaimUsageExport(ctx context.Context) (db.UsageExport, error)
	CompleteUsageExport(ctx context.Context, params db.CompleteUsageExportParams) (db.UsageExport, error)
	FailUsageExport(ctx context.Context, params db.FailUsageExportParams) (db.UsageExport, error)
	ExportUsageRows(ctx context.Context, params db.ExportUsageRowsParams) ([]db.ExportUsageRowsRow, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) CreateUsageExport(ctx context.Context, params db.CreateUsageExportParams) (db.UsageExport, error) {
	return a.q.CreateUsageExport(ctx, params)
}

func (a *queriesAdapter) GetUsageExport(ctx context.Context, id pgtype.UUID) (db.UsageExport, error) {
	return a.q.GetUsageExport(ctx, id)
}

func (a *queriesAdapter) ListUsageExportsAdmin(ctx context.Context, params db.ListUsageExportsAdminParams) ([]db.UsageExport, error) {
	return a.q.ListUsageExportsAdmin(ctx, params)
}

func (a *queriesAdapter) ListUsageExportsByRequester(ctx context.Context, params db.ListUsageExportsByRequesterParams) ([]db.UsageExport, error) {
	return a.q.ListUsageExportsByRequester(ctx, params)
}

func (a *queriesAdapter) ClaimUsageExport(ctx context.Context) (db.UsageExport, error) {
	return a.q.ClaimUsageExport(ctx)
}

func (a *queriesAdapter) CompleteUsageExport(ctx context.Context, params db.CompleteUsageExportParams) (db.UsageExport, error) {
	return a.q.CompleteUsageExport(ctx, params)
}

func (a *queriesAdapter) FailUsageExport(ctx context.Context, params db.FailUsageExportParams) (db.UsageExport, error) {
	return a.q.FailUsageExport(ctx, params)
}

func (a *queriesAdapter) ExportUsageRows(ctx context.Context, params db.ExportUsageRowsParams) ([]db.ExportUsageRowsRow, error) {
	return a.q.ExportUsageRows(ctx, params)
}

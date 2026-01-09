package usagepipeline

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for usage pipeline operations.
type Repository interface {
	// Usage recording
	InsertRequestRecord(ctx context.Context, params db.InsertRequestRecordParams) (db.Request, error)
	InsertUsageRecord(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error)

	// Budget evaluation
	SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)

	// Alert dispatch
	GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error)
	UpdateTenantBudgetAlertState(ctx context.Context, params db.UpdateTenantBudgetAlertStateParams) error
	InsertBudgetAlertEvent(ctx context.Context, params db.InsertBudgetAlertEventParams) error

	// Transaction support
	WithTx(tx pgx.Tx) Repository
}

// TxBeginner abstracts the ability to begin a database transaction.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries, pool *pgxpool.Pool) Repository {
	return &queriesAdapter{q: q, pool: pool}
}

func (a *queriesAdapter) InsertRequestRecord(ctx context.Context, params db.InsertRequestRecordParams) (db.Request, error) {
	return a.q.InsertRequestRecord(ctx, params)
}

func (a *queriesAdapter) InsertUsageRecord(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error) {
	return a.q.InsertUsageRecord(ctx, params)
}

func (a *queriesAdapter) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	return a.q.SumUsageForTenant(ctx, params)
}

func (a *queriesAdapter) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	return a.q.GetTenantByID(ctx, id)
}

func (a *queriesAdapter) UpdateTenantBudgetAlertState(ctx context.Context, params db.UpdateTenantBudgetAlertStateParams) error {
	return a.q.UpdateTenantBudgetAlertState(ctx, params)
}

func (a *queriesAdapter) InsertBudgetAlertEvent(ctx context.Context, params db.InsertBudgetAlertEventParams) error {
	return a.q.InsertBudgetAlertEvent(ctx, params)
}

func (a *queriesAdapter) WithTx(tx pgx.Tx) Repository {
	return &queriesAdapter{q: a.q.WithTx(tx), pool: a.pool}
}

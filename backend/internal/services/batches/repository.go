package batches

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for batch operations.
type Repository interface {
	// Batch operations
	CreateBatch(ctx context.Context, params db.CreateBatchParams) (db.Batch, error)
	GetBatch(ctx context.Context, params db.GetBatchParams) (db.Batch, error)
	GetBatchByID(ctx context.Context, id pgtype.UUID) (db.Batch, error)
	ListBatches(ctx context.Context, params db.ListBatchesParams) ([]db.Batch, error)
	ListBatchesCursor(ctx context.Context, params db.ListBatchesCursorParams) ([]db.Batch, error)
	ListBatchesAdmin(ctx context.Context, params db.ListBatchesAdminParams) ([]db.ListBatchesAdminRow, error)
	CancelBatch(ctx context.Context, params db.CancelBatchParams) (db.Batch, error)
	GetOldestQueuedBatch(ctx context.Context) (db.Batch, error)
	MarkBatchInProgress(ctx context.Context, id pgtype.UUID) (db.Batch, error)
	MarkBatchFinalStatus(ctx context.Context, params db.MarkBatchFinalStatusParams) (db.Batch, error)
	IncrementBatchCounts(ctx context.Context, params db.IncrementBatchCountsParams) error

	// Batch item operations
	InsertBatchItem(ctx context.Context, params db.InsertBatchItemParams) (db.BatchItem, error)
	ClaimNextBatchItem(ctx context.Context, batchID pgtype.UUID) (db.BatchItem, error)
	CompleteBatchItem(ctx context.Context, params db.CompleteBatchItemParams) error
	FailBatchItem(ctx context.Context, params db.FailBatchItemParams) error

	// Transaction support
	WithTx(tx pgx.Tx) Repository
}

// TxBeginner abstracts beginning a database transaction.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// queriesAdapter wraps *db.Queries and *pgxpool.Pool to implement the Repository interface.
type queriesAdapter struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewQueriesRepository creates a Repository backed by *db.Queries and *pgxpool.Pool.
func NewQueriesRepository(q *db.Queries, pool *pgxpool.Pool) Repository {
	return &queriesAdapter{q: q, pool: pool}
}

func (a *queriesAdapter) CreateBatch(ctx context.Context, params db.CreateBatchParams) (db.Batch, error) {
	return a.q.CreateBatch(ctx, params)
}

func (a *queriesAdapter) GetBatch(ctx context.Context, params db.GetBatchParams) (db.Batch, error) {
	return a.q.GetBatch(ctx, params)
}

func (a *queriesAdapter) GetBatchByID(ctx context.Context, id pgtype.UUID) (db.Batch, error) {
	return a.q.GetBatchByID(ctx, id)
}

func (a *queriesAdapter) ListBatches(ctx context.Context, params db.ListBatchesParams) ([]db.Batch, error) {
	return a.q.ListBatches(ctx, params)
}

func (a *queriesAdapter) ListBatchesCursor(ctx context.Context, params db.ListBatchesCursorParams) ([]db.Batch, error) {
	return a.q.ListBatchesCursor(ctx, params)
}

func (a *queriesAdapter) ListBatchesAdmin(ctx context.Context, params db.ListBatchesAdminParams) ([]db.ListBatchesAdminRow, error) {
	return a.q.ListBatchesAdmin(ctx, params)
}

func (a *queriesAdapter) CancelBatch(ctx context.Context, params db.CancelBatchParams) (db.Batch, error) {
	return a.q.CancelBatch(ctx, params)
}

func (a *queriesAdapter) GetOldestQueuedBatch(ctx context.Context) (db.Batch, error) {
	return a.q.GetOldestQueuedBatch(ctx)
}

func (a *queriesAdapter) MarkBatchInProgress(ctx context.Context, id pgtype.UUID) (db.Batch, error) {
	return a.q.MarkBatchInProgress(ctx, id)
}

func (a *queriesAdapter) MarkBatchFinalStatus(ctx context.Context, params db.MarkBatchFinalStatusParams) (db.Batch, error) {
	return a.q.MarkBatchFinalStatus(ctx, params)
}

func (a *queriesAdapter) IncrementBatchCounts(ctx context.Context, params db.IncrementBatchCountsParams) error {
	return a.q.IncrementBatchCounts(ctx, params)
}

func (a *queriesAdapter) InsertBatchItem(ctx context.Context, params db.InsertBatchItemParams) (db.BatchItem, error) {
	return a.q.InsertBatchItem(ctx, params)
}

func (a *queriesAdapter) ClaimNextBatchItem(ctx context.Context, batchID pgtype.UUID) (db.BatchItem, error) {
	return a.q.ClaimNextBatchItem(ctx, batchID)
}

func (a *queriesAdapter) CompleteBatchItem(ctx context.Context, params db.CompleteBatchItemParams) error {
	return a.q.CompleteBatchItem(ctx, params)
}

func (a *queriesAdapter) FailBatchItem(ctx context.Context, params db.FailBatchItemParams) error {
	return a.q.FailBatchItem(ctx, params)
}

func (a *queriesAdapter) WithTx(tx pgx.Tx) Repository {
	return &queriesAdapter{q: a.q.WithTx(tx), pool: a.pool}
}

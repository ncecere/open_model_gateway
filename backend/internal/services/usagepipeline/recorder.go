package usagepipeline

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// UsageRecorder persists request + usage rows inside a single transaction.
type UsageRecorder struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewUsageRecorder(pool *pgxpool.Pool, queries *db.Queries) *UsageRecorder {
	return &UsageRecorder{pool: pool, queries: queries}
}

// Persist writes the request + usage rows with the provided cost in cents/micros.
func (r *UsageRecorder) Persist(ctx context.Context, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	if r == nil {
		return ErrRecorderUnavailable
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)
	if err := insertRequest(ctx, qtx, rec, ts, costCents, costMicros); err != nil {
		return err
	}
	if rec.Success {
		if err := insertUsage(ctx, qtx, rec, ts, costCents, costMicros); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

var ErrRecorderUnavailable = errors.New("usage recorder unavailable")

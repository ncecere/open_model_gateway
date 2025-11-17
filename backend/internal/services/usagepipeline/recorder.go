package usagepipeline

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// UsageRecorder persists request + usage rows inside a single transaction.
type UsageRecorder struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	tracer  trace.Tracer
}

func NewUsageRecorder(pool *pgxpool.Pool, queries *db.Queries) *UsageRecorder {
	return &UsageRecorder{pool: pool, queries: queries, tracer: otel.Tracer("open-model-gateway/usage-recorder")}
}

// Persist writes the request + usage rows with the provided cost in cents/micros.
func (r *UsageRecorder) Persist(ctx context.Context, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	if r == nil {
		return ErrRecorderUnavailable
	}
	ctx, span := r.tracer.Start(ctx, "UsageRecorder.Persist", trace.WithAttributes(
		attribute.String("model", rec.Alias),
		attribute.String("provider", rec.Provider),
	))
	defer span.End()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)
	if err := insertRequest(ctx, qtx, rec, ts, costCents, costMicros); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if rec.Success {
		if err := insertUsage(ctx, qtx, rec, ts, costCents, costMicros); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

var ErrRecorderUnavailable = errors.New("usage recorder unavailable")

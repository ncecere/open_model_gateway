package usagepipeline

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

func insertRequest(ctx context.Context, q *db.Queries, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	latency := rec.Latency.Milliseconds()
	if latency < 0 {
		latency = 0
	}

	_, err := q.InsertRequestRecord(ctx, db.InsertRequestRecordParams{
		TenantID:       toPgUUID(rec.Context.TenantID),
		ApiKeyID:       toPgNullableUUID(rec.Context.APIKeyID),
		Ts:             pgtype.Timestamptz{Time: ts, Valid: true},
		ModelAlias:     rec.Alias,
		Provider:       rec.Provider,
		LatencyMs:      int32(latency),
		Status:         int32(rec.Status),
		ErrorCode:      toPgText(rec.ErrorCode),
		InputTokens:    int64(rec.Usage.PromptTokens),
		OutputTokens:   int64(rec.Usage.CompletionTokens),
		CostCents:      costCents,
		CostUsdMicros:  costMicros,
		IdempotencyKey: toPgText(rec.IdempotencyKey),
		TraceID:        toPgText(rec.TraceID),
	})
	return err
}

func insertUsage(ctx context.Context, q *db.Queries, rec Record, ts time.Time, costCents int64, costMicros int64) error {
	_, err := q.InsertUsageRecord(ctx, db.InsertUsageRecordParams{
		TenantID:      toPgUUID(rec.Context.TenantID),
		ApiKeyID:      toPgNullableUUID(rec.Context.APIKeyID),
		Ts:            pgtype.Timestamptz{Time: ts, Valid: true},
		ModelAlias:    rec.Alias,
		Provider:      rec.Provider,
		InputTokens:   int64(rec.Usage.PromptTokens),
		OutputTokens:  int64(rec.Usage.CompletionTokens),
		Requests:      1,
		CostCents:     costCents,
		CostUsdMicros: costMicros,
	})
	return err
}

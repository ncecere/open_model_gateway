package usagepipeline

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// BudgetEvaluator centralizes budget config, window math, and usage aggregation.
type BudgetEvaluator struct {
	queries *db.Queries
	cfgMu   sync.RWMutex
	cfg     config.BudgetConfig
}

func NewBudgetEvaluator(cfg config.BudgetConfig, queries *db.Queries) *BudgetEvaluator {
	return &BudgetEvaluator{queries: queries, cfg: cfg}
}

func (b *BudgetEvaluator) Config() config.BudgetConfig {
	b.cfgMu.RLock()
	defer b.cfgMu.RUnlock()
	return b.cfg
}

func (b *BudgetEvaluator) SetConfig(cfg config.BudgetConfig) {
	b.cfgMu.Lock()
	b.cfg = cfg
	b.cfgMu.Unlock()
}

func (b *BudgetEvaluator) EffectiveLimit(rc *requestctx.Context) int64 {
	if rc != nil && rc.BudgetLimitCents > 0 {
		return rc.BudgetLimitCents
	}
	cfg := b.Config()
	return dollarsToCents(cfg.DefaultUSD)
}

func (b *BudgetEvaluator) Schedule(rc *requestctx.Context) string {
	if rc != nil {
		if sched := strings.TrimSpace(rc.BudgetRefreshSchedule); sched != "" {
			return config.NormalizeBudgetRefreshSchedule(sched)
		}
	}
	return config.NormalizeBudgetRefreshSchedule(b.Config().RefreshSchedule)
}

func (b *BudgetEvaluator) SumUsage(ctx context.Context, tenantID uuid.UUID, now time.Time, schedule string) (int64, error) {
	start, end := periodBounds(now, schedule)
	row, err := b.queries.SumUsageForTenant(ctx, db.SumUsageForTenantParams{
		TenantID: toPgUUID(tenantID),
		Ts:       pgtype.Timestamptz{Time: start, Valid: true},
		Ts_2:     pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return toInt64(row.TotalCostCents), nil
}

func (b *BudgetEvaluator) Check(ctx context.Context, rc *requestctx.Context, now time.Time) (BudgetStatus, error) {
	if rc == nil {
		return BudgetStatus{}, errors.New("request context missing")
	}
	limit := b.EffectiveLimit(rc)
	if limit <= 0 {
		return BudgetStatus{}, errors.New("invalid budget limit")
	}

	schedule := b.Schedule(rc)
	total, err := b.SumUsage(ctx, rc.TenantID, now, schedule)
	if err != nil {
		return BudgetStatus{}, err
	}

	exceeded := total >= limit
	warning := !exceeded && overThreshold(total, limit, rc.WarningThreshold)

	return BudgetStatus{
		TotalCostCents: total,
		LimitCents:     limit,
		Warning:        warning,
		Exceeded:       exceeded,
	}, nil
}

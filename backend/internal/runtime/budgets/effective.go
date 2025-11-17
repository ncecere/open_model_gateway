package budgets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Effective computes the tenant-specific budget and warning threshold.
func Effective(ctx context.Context, queries *db.Queries, cfg config.BudgetConfig, tenantID uuid.UUID) (float64, float64, error) {
	limit := cfg.DefaultUSD
	warning := cfg.WarningThresholdPerc
	if tenantID == uuid.Nil || queries == nil {
		return limit, warning, nil
	}
	override, err := queries.GetTenantBudgetOverride(ctx, toPgUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return limit, warning, nil
		}
		return 0, 0, err
	}
	if value, ok := override.BudgetUsd.Float64(); ok && value > 0 {
		limit = value
	}
	if value, ok := override.WarningThreshold.Float64(); ok && value > 0 {
		warning = value
	}
	return limit, warning, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

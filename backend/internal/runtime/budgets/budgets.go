package budgets

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Load pulls persisted defaults (if any) and applies them to cfg.Budgets.
func Load(ctx context.Context, queries *db.Queries, cfg *config.Config) error {
	if queries == nil || cfg == nil {
		return nil
	}
	defaults, err := queries.GetBudgetDefaults(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	cfg.Budgets = FromRecord(cfg.Budgets, defaults)
	return nil
}

// FromRecord merges a stored record into the provided base config.
func FromRecord(base config.BudgetConfig, record db.BudgetDefault) config.BudgetConfig {
	if val, ok := record.DefaultUsd.Float64(); ok {
		base.DefaultUSD = val
	}
	if val, ok := record.WarningThreshold.Float64(); ok && val > 0 && val <= 1 {
		base.WarningThresholdPerc = val
	}
	if schedule := strings.TrimSpace(record.RefreshSchedule); schedule != "" {
		base.RefreshSchedule = config.NormalizeBudgetRefreshSchedule(schedule)
	}
	base.Alert.Emails = record.AlertEmails
	base.Alert.Webhooks = record.AlertWebhooks
	if record.AlertCooldownSeconds > 0 {
		base.Alert.Cooldown = time.Duration(record.AlertCooldownSeconds) * time.Second
	}
	base.Alert.Enabled = len(base.Alert.Emails) > 0 || len(base.Alert.Webhooks) > 0
	return base
}

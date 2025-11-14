package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// LoadBudgetDefaults pulls persisted defaults (if any) and applies them to cfg.Budgets.
func LoadBudgetDefaults(ctx context.Context, queries *db.Queries, cfg *config.Config) error {
	defaults, err := queries.GetBudgetDefaults(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	cfg.Budgets = BudgetConfigFromRecord(cfg.Budgets, defaults)
	return nil
}

// BudgetConfigFromRecord merges a stored record into the provided base config.
func BudgetConfigFromRecord(base config.BudgetConfig, record db.BudgetDefault) config.BudgetConfig {
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

// UpdateBudgetConfig swaps the in-memory config and notifies dependent services.
func (c *Container) UpdateBudgetConfig(cfg config.BudgetConfig) {
	c.Config.Budgets = cfg
	if c.UsageLogger != nil {
		c.UsageLogger.SetConfig(cfg)
	}
}

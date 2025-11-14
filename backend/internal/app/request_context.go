package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

const defaultWarnFloor = 0.5

// BuildRequestContext translates an API key record into the runtime request context
// used by both HTTP handlers and background workers.
func BuildRequestContext(ctx context.Context, container *Container, record db.ApiKey) (*requestctx.Context, error) {
	if container == nil {
		return nil, errors.New("container required")
	}

	tenantID, err := uuidFromPg(record.TenantID)
	if err != nil || tenantID == uuid.Nil {
		tenantID, err = ensureAPIKeyTenant(ctx, container, record)
		if err != nil {
			return nil, err
		}
	}
	keyID, err := uuidFromPg(record.ID)
	if err != nil {
		return nil, err
	}

	var scopes []string
	if len(record.ScopesJson) > 0 {
		if err := json.Unmarshal(record.ScopesJson, &scopes); err != nil {
			scopes = nil
		}
	}

	quota := struct {
		BudgetUSD        float64 `json:"budget_usd"`
		WarningThreshold float64 `json:"warning_threshold"`
	}{}
	if len(record.QuotaJson) > 0 {
		_ = json.Unmarshal(record.QuotaJson, &quota)
	}

	var tenantBudget int64
	var tenantWarn float64
	schedule := container.Config.Budgets.RefreshSchedule
	alertsEnabled := container.Config.Budgets.Alert.Enabled
	alertEmails := container.Config.Budgets.Alert.Emails
	alertWebhooks := container.Config.Budgets.Alert.Webhooks
	alertCooldown := container.Config.Budgets.Alert.Cooldown
	alertLastLevel := ""
	var alertLastSent time.Time
	hasOverride := false

	if override, err := container.Queries.GetTenantBudgetOverride(ctx, record.TenantID); err == nil {
		if budget, ok := override.BudgetUsd.Float64(); ok {
			tenantBudget = int64(math.Round(budget * 100))
		}
		if w, ok := override.WarningThreshold.Float64(); ok {
			tenantWarn = w
		}
		schedule = config.NormalizeBudgetRefreshSchedule(override.RefreshSchedule)
		if len(override.AlertEmails) > 0 {
			alertEmails = trimStrings(override.AlertEmails)
		}
		if len(override.AlertWebhooks) > 0 {
			alertWebhooks = trimStrings(override.AlertWebhooks)
		}
		if override.AlertCooldownSeconds > 0 {
			alertCooldown = time.Duration(override.AlertCooldownSeconds) * time.Second
		}
		if override.LastAlertLevel.Valid {
			alertLastLevel = strings.TrimSpace(override.LastAlertLevel.String)
		}
		if override.LastAlertAt.Valid {
			alertLastSent = override.LastAlertAt.Time
		}
		hasOverride = true
	}
	if len(alertEmails) > 0 || len(alertWebhooks) > 0 {
		alertsEnabled = true
	}

	limit := int64(math.Round(quota.BudgetUSD * 100))
	if limit <= 0 {
		if tenantBudget > 0 {
			limit = tenantBudget
		} else {
			limit = int64(math.Round(container.Config.Budgets.DefaultUSD * 100))
		}
	}

	warn := quota.WarningThreshold
	if warn <= 0 {
		if tenantWarn > 0 {
			warn = tenantWarn
		} else {
			warn = container.Config.Budgets.WarningThresholdPerc
		}
	}
	if warn < defaultWarnFloor {
		warn = defaultWarnFloor
	}

	return &requestctx.Context{
		TenantID:              tenantID,
		APIKeyID:              keyID,
		APIKeyPrefix:          record.Prefix,
		Scopes:                scopes,
		BudgetLimitCents:      limit,
		WarningThreshold:      warn,
		BudgetRefreshSchedule: schedule,
		AlertsEnabled:         alertsEnabled,
		AlertEmails:           alertEmails,
		AlertWebhooks:         alertWebhooks,
		AlertCooldown:         alertCooldown,
		AlertLastLevel:        alertLastLevel,
		AlertLastSent:         alertLastSent,
		HasBudgetOverride:     hasOverride,
	}, nil
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if val := strings.TrimSpace(v); val != "" {
			out = append(out, val)
		}
	}
	return out
}

func ensureAPIKeyTenant(ctx context.Context, container *Container, record db.ApiKey) (uuid.UUID, error) {
	if container == nil || container.Accounts == nil || container.Queries == nil {
		return uuid.UUID{}, errors.New("tenant missing for api key")
	}
	if !record.OwnerUserID.Valid {
		return uuid.UUID{}, errors.New("api key tenant not set")
	}

	user, err := container.Queries.GetUserByID(ctx, record.OwnerUserID)
	if err != nil {
		return uuid.UUID{}, err
	}

	_, tenant, err := container.Accounts.EnsurePersonalTenant(ctx, user)
	if err != nil {
		return uuid.UUID{}, err
	}

	tenantUUID, err := uuidFromPg(tenant.ID)
	if err != nil {
		return uuid.UUID{}, err
	}

	slog.Info("assigning personal tenant to api key", "api_key_id", record.ID.Bytes, "tenant_id", tenantUUID.String())

	_, err = container.Queries.UpdateAPIKeyTenant(ctx, db.UpdateAPIKeyTenantParams{
		ID:       record.ID,
		TenantID: toPgUUID(tenantUUID),
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	record.TenantID = toPgUUID(tenantUUID)

	return tenantUUID, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	var out pgtype.UUID
	if id == uuid.Nil {
		out.Valid = false
		return out
	}
	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
}

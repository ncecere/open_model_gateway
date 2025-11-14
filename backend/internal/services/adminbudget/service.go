package adminbudget

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

var (
	ErrServiceUnavailable = errors.New("admin budget service not initialized")
	ErrInvalidDefault     = errors.New("default_usd must be positive")
	ErrInvalidThreshold   = errors.New("warning_threshold must be between 0 and 1")
	ErrInvalidOverride    = errors.New("budget_usd must be positive")
)

// Service wraps DB + config helpers for admin budget operations.
type Service struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewService(queries *db.Queries, cfg *config.Config) *Service {
	return &Service{queries: queries, cfg: cfg}
}

type DefaultUpdate struct {
	DefaultUSD           float64
	WarningThreshold     float64
	RefreshSchedule      string
	AlertEmails          []string
	AlertWebhooks        []string
	AlertCooldownSeconds int32
	UpdatedByUserID      uuid.UUID
}

type OverrideRequest struct {
	BudgetUSD            float64
	WarningThreshold     float64
	RefreshSchedule      string
	AlertEmails          []string
	AlertWebhooks        []string
	AlertCooldownSeconds *int32
}

func (s *Service) UpdateDefaults(ctx context.Context, req DefaultUpdate) (db.BudgetDefault, error) {
	if s == nil || s.queries == nil || s.cfg == nil {
		return db.BudgetDefault{}, ErrServiceUnavailable
	}
	if req.DefaultUSD <= 0 {
		return db.BudgetDefault{}, ErrInvalidDefault
	}
	if req.WarningThreshold <= 0 || req.WarningThreshold > 1 {
		return db.BudgetDefault{}, ErrInvalidThreshold
	}
	refresh := config.NormalizeBudgetRefreshSchedule(req.RefreshSchedule)
	emails := trimStrings(req.AlertEmails)
	webhooks := trimStrings(req.AlertWebhooks)
	cooldown := req.AlertCooldownSeconds
	if cooldown <= 0 {
		cooldown = int32(time.Hour / time.Second)
	}
	return s.queries.UpsertBudgetDefaults(ctx, db.UpsertBudgetDefaultsParams{
		DefaultUsd:           decimal.NewFromFloat(req.DefaultUSD).Round(2),
		WarningThreshold:     decimal.NewFromFloat(req.WarningThreshold),
		RefreshSchedule:      refresh,
		AlertEmails:          emails,
		AlertWebhooks:        webhooks,
		AlertCooldownSeconds: cooldown,
		CreatedByUserID:      optionalPgUUID(req.UpdatedByUserID),
		UpdatedByUserID:      optionalPgUUID(req.UpdatedByUserID),
	})
}

func (s *Service) GetDefaults(ctx context.Context) (db.BudgetDefault, error) {
	if s == nil || s.queries == nil {
		return db.BudgetDefault{}, ErrServiceUnavailable
	}
	return s.queries.GetBudgetDefaults(ctx)
}

func (s *Service) ListOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	return s.queries.ListTenantBudgetOverrides(ctx)
}

func (s *Service) GetOverride(ctx context.Context, tenantID uuid.UUID) (db.TenantBudgetOverride, error) {
	if s == nil || s.queries == nil {
		return db.TenantBudgetOverride{}, ErrServiceUnavailable
	}
	return s.queries.GetTenantBudgetOverride(ctx, toPgUUID(tenantID))
}

func (s *Service) UpsertOverride(ctx context.Context, tenantID uuid.UUID, req OverrideRequest) (db.TenantBudgetOverride, error) {
	if s == nil || s.queries == nil || s.cfg == nil {
		return db.TenantBudgetOverride{}, ErrServiceUnavailable
	}
	if req.BudgetUSD <= 0 {
		return db.TenantBudgetOverride{}, ErrInvalidOverride
	}
	if req.WarningThreshold <= 0 || req.WarningThreshold > 1 {
		return db.TenantBudgetOverride{}, ErrInvalidThreshold
	}
	params := s.buildOverrideParams(tenantID, req)
	return s.queries.UpsertTenantBudgetOverride(ctx, params)
}

func (s *Service) DeleteOverride(ctx context.Context, tenantID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	return s.queries.DeleteTenantBudgetOverride(ctx, toPgUUID(tenantID))
}

func (s *Service) buildOverrideParams(tenantID uuid.UUID, req OverrideRequest) db.UpsertTenantBudgetOverrideParams {
	schedule := s.cfg.Budgets.RefreshSchedule
	if strings.TrimSpace(req.RefreshSchedule) != "" {
		schedule = config.NormalizeBudgetRefreshSchedule(req.RefreshSchedule)
	}
	alertEmails := trimStrings(req.AlertEmails)
	alertWebhooks := trimStrings(req.AlertWebhooks)
	if len(alertEmails) == 0 && len(alertWebhooks) == 0 && s.cfg.Budgets.Alert.Enabled {
		alertEmails = s.cfg.Budgets.Alert.Emails
		alertWebhooks = s.cfg.Budgets.Alert.Webhooks
	}
	cooldownSeconds := int32(s.cfg.Budgets.Alert.Cooldown / time.Second)
	if req.AlertCooldownSeconds != nil && *req.AlertCooldownSeconds > 0 {
		cooldownSeconds = *req.AlertCooldownSeconds
	}
	if cooldownSeconds <= 0 {
		cooldownSeconds = int32(time.Hour / time.Second)
	}
	return db.UpsertTenantBudgetOverrideParams{
		TenantID:             toPgUUID(tenantID),
		BudgetUsd:            decimal.NewFromFloat(req.BudgetUSD).Round(2),
		WarningThreshold:     decimal.NewFromFloat(req.WarningThreshold),
		RefreshSchedule:      schedule,
		AlertEmails:          alertEmails,
		AlertWebhooks:        alertWebhooks,
		AlertCooldownSeconds: cooldownSeconds,
	}
}

func trimStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func optionalPgUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return toPgUUID(id)
}

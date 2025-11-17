package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
)

// Params captures the dependencies required to execute bootstrap seeding.
type Params struct {
	Queries      *db.Queries
	AdminAuth    *auth.AdminAuthService
	PersonalSvc  *accounts.PersonalService
	Bootstrap    config.BootstrapConfig
	Budgets      config.BudgetConfig
	KeyLimits    map[string]limits.LimitConfig
	TenantLimits map[uuid.UUID]limits.LimitConfig
}

// Ensure seeds tenants, users, API keys, memberships, rate limits, and budgets
// using the supplied bootstrap configuration. The operation is idempotent.
func Ensure(ctx context.Context, p Params) error {
	if p.Queries == nil {
		return errors.New("bootstrap queries required")
	}

	// Tenants
	for _, tenant := range p.Bootstrap.Tenants {
		name := strings.TrimSpace(tenant.Name)
		if name == "" {
			continue
		}
		if _, err := p.Queries.GetTenantByName(ctx, name); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("bootstrap tenant %q lookup: %w", name, err)
			}
			status := db.TenantStatusActive
			if strings.EqualFold(tenant.Status, "suspended") {
				status = db.TenantStatusSuspended
			}
			if _, err := p.Queries.CreateTenant(ctx, db.CreateTenantParams{
				Name:   name,
				Status: status,
				Kind:   db.TenantKindOrganization,
			}); err != nil {
				return fmt.Errorf("bootstrap tenant %q create: %w", name, err)
			}
		}
	}

	// Admin users
	for _, user := range p.Bootstrap.AdminUsers {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		var dbUser db.User
		var err error
		dbUser, err = p.Queries.GetUserByEmail(ctx, email)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("bootstrap admin %q lookup: %w", email, err)
			}
			dbUser, err = p.Queries.CreateUser(ctx, db.CreateUserParams{
				Email: email,
				Name:  strings.TrimSpace(user.Name),
			})
			if err != nil {
				return fmt.Errorf("bootstrap admin %q create: %w", email, err)
			}
		}

		if p.PersonalSvc != nil {
			if updated, _, err := p.PersonalSvc.EnsurePersonalTenant(ctx, dbUser); err == nil {
				dbUser = updated
			} else {
				return fmt.Errorf("bootstrap admin %q personal tenant: %w", email, err)
			}
		}

		if strings.TrimSpace(user.Password) != "" {
			if p.AdminAuth == nil {
				return errors.New("admin auth service required for passwords")
			}
			userID, err := uuidFromPg(dbUser.ID)
			if err != nil {
				return fmt.Errorf("bootstrap admin %q user id: %w", email, err)
			}
			if err := p.AdminAuth.UpsertLocalPassword(ctx, userID, email, user.Password); err != nil {
				return fmt.Errorf("bootstrap admin %q password: %w", email, err)
			}
		}

		if err := p.Queries.SetUserSuperAdmin(ctx, db.SetUserSuperAdminParams{
			ID:           dbUser.ID,
			IsSuperAdmin: user.IsSuperAdmin(),
		}); err != nil {
			return fmt.Errorf("bootstrap admin %q super admin: %w", email, err)
		}
	}

	// API keys
	for _, key := range p.Bootstrap.APIKeys {
		prefix := strings.TrimSpace(key.Prefix)
		if prefix == "" {
			continue
		}
		var keyRecord db.ApiKey
		notFound := false
		record, err := p.Queries.GetAPIKeyByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				notFound = true
			} else {
				return fmt.Errorf("bootstrap api key %q lookup: %w", prefix, err)
			}
		} else {
			keyRecord = record
		}

		tenantName := strings.TrimSpace(key.Tenant)
		if tenantName == "" {
			return fmt.Errorf("bootstrap api key %q missing tenant", prefix)
		}
		tenant, err := p.Queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap api key %q tenant %q: %w", prefix, tenantName, err)
		}

		if notFound {
			hash, err := auth.HashPassword(strings.TrimSpace(key.Secret))
			if err != nil {
				return fmt.Errorf("bootstrap api key %q hash: %w", prefix, err)
			}

			created, err := p.Queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
				TenantID:    tenant.ID,
				Prefix:      prefix,
				SecretHash:  hash,
				Name:        strings.TrimSpace(key.Name),
				ScopesJson:  []byte("[]"),
				QuotaJson:   []byte("{}"),
				Kind:        db.ApiKeyKindService,
				OwnerUserID: pgtype.UUID{Valid: false},
			})
			if err != nil {
				return fmt.Errorf("bootstrap api key %q create: %w", prefix, err)
			}
			keyRecord = created
		}

		override := limitFromBootstrapRate(key.RateLimit)
		if p.KeyLimits != nil {
			p.KeyLimits[prefix] = override
		}
		if keyRecord.ID.Valid && (override.RequestsPerMinute > 0 || override.TokensPerMinute > 0 || override.ParallelRequests > 0) {
			if _, err := p.Queries.UpsertAPIKeyRateLimit(ctx, db.UpsertAPIKeyRateLimitParams{
				ApiKeyID:          keyRecord.ID,
				RequestsPerMinute: int32(override.RequestsPerMinute),
				TokensPerMinute:   int32(override.TokensPerMinute),
				ParallelRequests:  int32(override.ParallelRequests),
			}); err != nil {
				return fmt.Errorf("bootstrap api key %q rate limit upsert: %w", prefix, err)
			}
		}
	}

	// Memberships
	for _, member := range p.Bootstrap.Memberships {
		tenantName := strings.TrimSpace(member.Tenant)
		email := strings.TrimSpace(member.Email)
		roleValue := strings.TrimSpace(member.Role)
		if tenantName == "" || email == "" {
			continue
		}

		tenant, err := p.Queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap membership tenant %q: %w", tenantName, err)
		}

		user, err := p.Queries.GetUserByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("bootstrap membership user %q: %w", email, err)
		}

		role, ok := rbac.ParseRole(roleValue)
		if !ok {
			return fmt.Errorf("bootstrap membership role %q invalid", roleValue)
		}

		existing, err := p.Queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
			TenantID: tenant.ID,
			UserID:   user.ID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if _, err := p.Queries.AddTenantMembership(ctx, db.AddTenantMembershipParams{
					TenantID: tenant.ID,
					UserID:   user.ID,
					Role:     role,
				}); err != nil {
					return fmt.Errorf("bootstrap membership add %q/%q: %w", tenantName, email, err)
				}
			} else {
				return fmt.Errorf("bootstrap membership lookup %q/%q: %w", tenantName, email, err)
			}
		} else if existing.Role != role {
			if _, err := p.Queries.UpdateTenantMembershipRole(ctx, db.UpdateTenantMembershipRoleParams{
				TenantID: tenant.ID,
				UserID:   user.ID,
				Role:     role,
			}); err != nil {
				return fmt.Errorf("bootstrap membership update %q/%q: %w", tenantName, email, err)
			}
		}
	}

	// Tenant limits
	for _, limit := range p.Bootstrap.TenantLimits {
		tenantName := strings.TrimSpace(limit.Tenant)
		if tenantName == "" {
			continue
		}
		tenant, err := p.Queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap tenant limit %q: %w", tenantName, err)
		}
		tenantUUID, err := uuidFromPg(tenant.ID)
		if err != nil {
			return fmt.Errorf("bootstrap tenant limit %q id: %w", tenantName, err)
		}
		override := limitFromBootstrapRate(limit.Limits)
		if p.TenantLimits != nil {
			p.TenantLimits[tenantUUID] = override
		}
		if _, err := p.Queries.UpsertTenantRateLimit(ctx, db.UpsertTenantRateLimitParams{
			TenantID:          tenant.ID,
			RequestsPerMinute: int32(limit.Limits.RequestsPerMinute),
			TokensPerMinute:   int32(limit.Limits.TokensPerMinute),
			ParallelRequests:  int32(limit.Limits.ParallelRequests),
		}); err != nil {
			return fmt.Errorf("bootstrap tenant limit %q upsert: %w", tenantName, err)
		}
	}

	// Tenant budgets
	for _, entry := range p.Bootstrap.TenantBudgets {
		tenantName := strings.TrimSpace(entry.Tenant)
		if tenantName == "" {
			continue
		}
		tenant, err := p.Queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap tenant budget %q: %w", tenantName, err)
		}

		budgetValue := p.Budgets.DefaultUSD
		if entry.BudgetUSD != nil {
			budgetValue = *entry.BudgetUSD
		}
		if budgetValue <= 0 {
			return fmt.Errorf("bootstrap tenant budget %q budget_usd must be > 0", tenantName)
		}

		warning := p.Budgets.WarningThresholdPerc
		if entry.WarningThreshold != nil {
			warning = *entry.WarningThreshold
		}

		refreshSchedule := p.Budgets.RefreshSchedule
		if strings.TrimSpace(entry.RefreshSchedule) != "" {
			refreshSchedule = config.NormalizeBudgetRefreshSchedule(entry.RefreshSchedule)
		}

		alertEmails := entry.AlertEmails
		alertWebhooks := entry.AlertWebhooks
		if len(alertEmails) == 0 && len(alertWebhooks) == 0 && p.Budgets.Alert.Enabled {
			alertEmails = p.Budgets.Alert.Emails
			alertWebhooks = p.Budgets.Alert.Webhooks
		}

		alertCooldown := p.Budgets.Alert.Cooldown
		if entry.AlertCooldown > 0 {
			alertCooldown = entry.AlertCooldown
		}
		if alertCooldown <= 0 {
			alertCooldown = time.Hour
		}

		cooldownSeconds := alertCooldown / time.Second
		if cooldownSeconds > math.MaxInt32 {
			cooldownSeconds = math.MaxInt32
		}

		params := db.UpsertTenantBudgetOverrideParams{
			TenantID:             tenant.ID,
			BudgetUsd:            decimal.NewFromFloat(budgetValue).Round(2),
			WarningThreshold:     decimal.NewFromFloat(warning),
			RefreshSchedule:      refreshSchedule,
			AlertEmails:          alertEmails,
			AlertWebhooks:        alertWebhooks,
			AlertCooldownSeconds: int32(cooldownSeconds),
		}

		if _, err := p.Queries.UpsertTenantBudgetOverride(ctx, params); err != nil {
			return fmt.Errorf("bootstrap tenant budget %q upsert: %w", tenantName, err)
		}
	}

	return nil
}

func limitFromBootstrapRate(rate config.BootstrapRateLimit) limits.LimitConfig {
	return limits.LimitConfig{
		RequestsPerMinute: rate.RequestsPerMinute,
		TokensPerMinute:   rate.TokensPerMinute,
		ParallelRequests:  rate.ParallelRequests,
	}
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

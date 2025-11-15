package admintenant

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
)

// Service centralizes admin-facing tenant operations.
type Service struct {
	queries         *db.Queries
	cfg             *config.Config
	timezone        *time.Location
	dbPool          *pgxpool.Pool
	accounts        *accounts.PersonalService
	adminAuth       *auth.AdminAuthService
	setTenantModels func(uuid.UUID, []string)
	setTenantRate   func(uuid.UUID, *limits.LimitConfig)
	setAPIKeyRate   func(string, *limits.LimitConfig)
}

// NewService builds an admin tenant service.
func NewService(cfg *config.Config, queries *db.Queries, tz *time.Location, pool *pgxpool.Pool, accounts *accounts.PersonalService, adminAuth *auth.AdminAuthService, setTenantModels func(uuid.UUID, []string), setTenantRate func(uuid.UUID, *limits.LimitConfig), setAPIKeyRate func(string, *limits.LimitConfig)) *Service {
	if tz == nil {
		tz = time.UTC
	}
	return &Service{
		cfg:             cfg,
		queries:         queries,
		timezone:        tz,
		dbPool:          pool,
		accounts:        accounts,
		adminAuth:       adminAuth,
		setTenantModels: setTenantModels,
		setTenantRate:   setTenantRate,
		setAPIKeyRate:   setAPIKeyRate,
	}
}

var (
	ErrServiceUnavailable   = errors.New("admin tenant service not initialized")
	ErrInvalidModelList     = errors.New("models must include at least one alias")
	ErrModelNotFound        = errors.New("model not found")
	ErrAPIKeyTenantMismatch = errors.New("api key does not belong to tenant")
	ErrLocalAuthDisabled    = errors.New("local authentication disabled")
	ErrInvalidRateLimit     = errors.New("rate limits must be positive")
)

// ListItem represents a tenant row plus budget summary.
type ListItem struct {
	ID             uuid.UUID
	Name           string
	Status         db.TenantStatus
	CreatedAt      time.Time
	BudgetLimitUSD float64
	BudgetUsedUSD  float64
	WarningThresh  *float64
}

// PersonalListItem represents a personal tenant linked to a specific user.
type PersonalListItem struct {
	TenantID        uuid.UUID
	UserID          uuid.UUID
	UserEmail       string
	UserName        string
	Status          db.TenantStatus
	CreatedAt       time.Time
	BudgetLimitUSD  float64
	BudgetUsedUSD   float64
	WarningThresh   *float64
	MembershipCount int64
}

// List returns tenant summaries with budget usage.
func (s *Service) List(ctx context.Context, limit, offset int32) ([]ListItem, error) {
	if s == nil || s.queries == nil || s.cfg == nil {
		return nil, ErrServiceUnavailable
	}
	rows, err := s.queries.ListTenants(ctx, db.ListTenantsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	overrides, err := s.queries.ListTenantBudgetOverrides(ctx)
	if err != nil {
		return nil, err
	}
	overrideMap := make(map[string]db.TenantBudgetOverride, len(overrides))
	for _, ov := range overrides {
		tenantID, err := uuidFromPg(ov.TenantID)
		if err != nil {
			continue
		}
		overrideMap[tenantID.String()] = ov
	}
	now := time.Now().In(s.timezone)
	items := make([]ListItem, 0, len(rows))
	for _, rec := range rows {
		if rec.Kind == db.TenantKindPersonal {
			continue
		}
		tenantID, err := uuidFromPg(rec.ID)
		if err != nil {
			continue
		}
		created, err := timeFromPg(rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		limitUSD := s.cfg.Budgets.DefaultUSD
		schedule := s.cfg.Budgets.RefreshSchedule
		var warnPtr *float64
		if override, ok := overrideMap[tenantID.String()]; ok {
			if val, ok := override.BudgetUsd.Float64(); ok && val > 0 {
				limitUSD = val
			}
			if warn, ok := override.WarningThreshold.Float64(); ok {
				warnCopy := warn
				warnPtr = &warnCopy
			}
			if rs := strings.TrimSpace(override.RefreshSchedule); rs != "" {
				schedule = rs
			}
		}
		used, err := s.computeUsageUSD(ctx, tenantID, schedule, now)
		if err != nil {
			return nil, err
		}
		items = append(items, ListItem{
			ID:             tenantID,
			Name:           rec.Name,
			Status:         rec.Status,
			CreatedAt:      created.In(s.timezone),
			BudgetLimitUSD: limitUSD,
			BudgetUsedUSD:  used,
			WarningThresh:  warnPtr,
		})
	}
	return items, nil
}

// ListPersonal returns personal tenants grouped by user.
func (s *Service) ListPersonal(ctx context.Context, limit, offset int32) ([]PersonalListItem, error) {
	if s == nil || s.queries == nil || s.cfg == nil {
		return nil, ErrServiceUnavailable
	}
	rows, err := s.queries.ListPersonalTenants(ctx, db.ListPersonalTenantsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	overrides, err := s.queries.ListTenantBudgetOverrides(ctx)
	if err != nil {
		return nil, err
	}
	overrideMap := make(map[string]db.TenantBudgetOverride, len(overrides))
	for _, ov := range overrides {
		tenantID, err := uuidFromPg(ov.TenantID)
		if err != nil {
			continue
		}
		overrideMap[tenantID.String()] = ov
	}
	now := time.Now().In(s.timezone)
	items := make([]PersonalListItem, 0, len(rows))
	for _, rec := range rows {
		tenantID, err := uuidFromPg(rec.ID)
		if err != nil {
			continue
		}
		userID, err := uuidFromPg(rec.UserID)
		if err != nil {
			continue
		}
		created, err := timeFromPg(rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		limitUSD := s.cfg.Budgets.DefaultUSD
		schedule := s.cfg.Budgets.RefreshSchedule
		var warnPtr *float64
		if override, ok := overrideMap[tenantID.String()]; ok {
			if val, ok := override.BudgetUsd.Float64(); ok && val > 0 {
				limitUSD = val
			}
			if warn, ok := override.WarningThreshold.Float64(); ok {
				warnCopy := warn
				warnPtr = &warnCopy
			}
			if rs := strings.TrimSpace(override.RefreshSchedule); rs != "" {
				schedule = rs
			}
		}
		used, err := s.computeUsageUSD(ctx, tenantID, schedule, now)
		if err != nil {
			return nil, err
		}
		items = append(items, PersonalListItem{
			TenantID:        tenantID,
			UserID:          userID,
			UserEmail:       rec.UserEmail,
			UserName:        rec.UserName,
			Status:          rec.Status,
			CreatedAt:       created.In(s.timezone),
			BudgetLimitUSD:  limitUSD,
			BudgetUsedUSD:   used,
			WarningThresh:   warnPtr,
			MembershipCount: rec.MembershipCount,
		})
	}
	return items, nil
}

// CreateTenant inserts a new tenant record.
func (s *Service) CreateTenant(ctx context.Context, name string, status db.TenantStatus) (db.Tenant, error) {
	if s == nil || s.queries == nil {
		return db.Tenant{}, ErrServiceUnavailable
	}
	return s.queries.CreateTenant(ctx, db.CreateTenantParams{
		Name:   name,
		Status: status,
		Kind:   db.TenantKindOrganization,
	})
}

// UpdateTenantName updates tenant metadata.
func (s *Service) UpdateTenantName(ctx context.Context, tenantID uuid.UUID, name string) (db.Tenant, error) {
	if s == nil || s.queries == nil {
		return db.Tenant{}, ErrServiceUnavailable
	}
	return s.queries.UpdateTenantName(ctx, db.UpdateTenantNameParams{
		ID:   toPgUUID(tenantID),
		Name: name,
	})
}

// UpdateTenantStatus updates tenant status.
func (s *Service) UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status db.TenantStatus) (db.Tenant, error) {
	if s == nil || s.queries == nil {
		return db.Tenant{}, ErrServiceUnavailable
	}
	return s.queries.UpdateTenantStatus(ctx, db.UpdateTenantStatusParams{
		ID:     toPgUUID(tenantID),
		Status: status,
	})
}

// ListModels returns tenant model aliases.
func (s *Service) ListModels(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	return s.queries.ListTenantModels(ctx, toPgUUID(tenantID))
}

// SetModels replaces tenant model list after validating aliases.
func (s *Service) SetModels(ctx context.Context, tenantID uuid.UUID, aliases []string) ([]string, error) {
	if s == nil || s.queries == nil || s.dbPool == nil {
		return nil, ErrServiceUnavailable
	}
	finalList, err := s.normalizeModelAliases(ctx, aliases)
	if err != nil {
		return nil, err
	}
	tx, err := s.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteTenantModels(ctx, toPgUUID(tenantID)); err != nil {
		return nil, err
	}
	for _, alias := range finalList {
		if err := qtx.InsertTenantModel(ctx, db.InsertTenantModelParams{
			TenantID: toPgUUID(tenantID),
			Alias:    alias,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.setTenantModels != nil {
		s.setTenantModels(tenantID, finalList)
	}
	return finalList, nil
}

// DeleteModels removes all tenant models.
func (s *Service) DeleteModels(ctx context.Context, tenantID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	if err := s.queries.DeleteTenantModels(ctx, toPgUUID(tenantID)); err != nil {
		return err
	}
	if s.setTenantModels != nil {
		s.setTenantModels(tenantID, nil)
	}
	return nil
}

// APIKeyCreateResult bundles key + secrets.
type APIKeyCreateResult struct {
	Key    db.ApiKey
	Secret string
	Token  string
}

// CreateAPIKey issues a new service key.
func (s *Service) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, name string, scopesJSON, quotaJSON []byte, rateLimit *limits.LimitConfig) (APIKeyCreateResult, error) {
	if s == nil || s.queries == nil {
		return APIKeyCreateResult{}, ErrServiceUnavailable
	}
	prefix, secret, token, err := auth.GenerateAPIKey()
	if err != nil {
		return APIKeyCreateResult{}, err
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return APIKeyCreateResult{}, err
	}
	key, err := s.queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		TenantID:    toPgUUID(tenantID),
		Prefix:      prefix,
		SecretHash:  hash,
		Name:        name,
		ScopesJson:  scopesJSON,
		QuotaJson:   quotaJSON,
		Kind:        db.ApiKeyKindService,
		OwnerUserID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		return APIKeyCreateResult{}, err
	}
	if rateLimit != nil {
		if _, err := s.queries.UpsertAPIKeyRateLimit(ctx, db.UpsertAPIKeyRateLimitParams{
			ApiKeyID:          key.ID,
			RequestsPerMinute: int32(rateLimit.RequestsPerMinute),
			TokensPerMinute:   int32(rateLimit.TokensPerMinute),
			ParallelRequests:  int32(rateLimit.ParallelRequests),
		}); err != nil {
			return APIKeyCreateResult{}, err
		}
		if s.setAPIKeyRate != nil {
			s.setAPIKeyRate(prefix, rateLimit)
		}
	}
	return APIKeyCreateResult{Key: key, Secret: secret, Token: token}, nil
}

// ListAPIKeys returns keys for tenant.
func (s *Service) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]db.ApiKey, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	return s.queries.ListAPIKeysByTenant(ctx, toPgUUID(tenantID))
}

// RevokeAPIKey revokes a key ensuring tenant ownership.
func (s *Service) RevokeAPIKey(ctx context.Context, tenantID, apiKeyID uuid.UUID) (db.ApiKey, error) {
	if s == nil || s.queries == nil {
		return db.ApiKey{}, ErrServiceUnavailable
	}
	if _, err := s.queries.GetTenantByID(ctx, toPgUUID(tenantID)); err != nil {
		return db.ApiKey{}, err
	}
	record, err := s.queries.RevokeAPIKey(ctx, toPgUUID(apiKeyID))
	if err != nil {
		return db.ApiKey{}, err
	}
	recTenant, err := uuidFromPg(record.TenantID)
	if err != nil {
		return db.ApiKey{}, err
	}
	if recTenant != tenantID {
		return db.ApiKey{}, ErrAPIKeyTenantMismatch
	}
	return record, nil
}

// Membership represents membership details.
type Membership struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Email    string
	Role     db.MembershipRole
	Created  time.Time
}

// ListMemberships fetches tenant members.
func (s *Service) ListMemberships(ctx context.Context, tenantID uuid.UUID) ([]Membership, error) {
	if s == nil || s.queries == nil {
		return nil, ErrServiceUnavailable
	}
	rows, err := s.queries.ListTenantMembers(ctx, toPgUUID(tenantID))
	if err != nil {
		return nil, err
	}
	out := make([]Membership, 0, len(rows))
	for _, row := range rows {
		tid, err := uuidFromPg(row.TenantID)
		if err != nil {
			return nil, err
		}
		uid, err := uuidFromPg(row.UserID)
		if err != nil {
			return nil, err
		}
		created, err := timeFromPg(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, Membership{
			TenantID: tid,
			UserID:   uid,
			Email:    row.UserEmail,
			Role:     row.Role,
			Created:  created,
		})
	}
	return out, nil
}

// UpsertMembership ensures membership exists and optional password set.
func (s *Service) UpsertMembership(ctx context.Context, tenantID uuid.UUID, email string, role db.MembershipRole, password string) (Membership, error) {
	if s == nil || s.queries == nil {
		return Membership{}, ErrServiceUnavailable
	}
	user, err := s.ensureUser(ctx, email)
	if err != nil {
		return Membership{}, err
	}
	userUUID, err := uuidFromPg(user.ID)
	if err != nil {
		return Membership{}, err
	}
	if password != "" {
		if s.adminAuth == nil || s.cfg == nil || !s.cfg.Admin.Local.Enabled {
			return Membership{}, ErrLocalAuthDisabled
		}
		if err := s.adminAuth.UpsertLocalPassword(ctx, userUUID, email, password); err != nil {
			return Membership{}, err
		}
	}

	membership, err := s.upsertMembershipRecord(ctx, toPgUUID(tenantID), user.ID, role)
	if err != nil {
		return Membership{}, err
	}
	created, err := timeFromPg(membership.CreatedAt)
	if err != nil {
		return Membership{}, err
	}
	tenantUUID, err := uuidFromPg(membership.TenantID)
	if err != nil {
		return Membership{}, err
	}
	return Membership{
		TenantID: tenantUUID,
		UserID:   userUUID,
		Email:    user.Email,
		Role:     role,
		Created:  created,
	}, nil
}

// RemoveMembership deletes membership.
func (s *Service) RemoveMembership(ctx context.Context, tenantID, userID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	return s.queries.RemoveTenantMembership(ctx, db.RemoveTenantMembershipParams{
		TenantID: toPgUUID(tenantID),
		UserID:   toPgUUID(userID),
	})
}

// GetTenantRateLimitOverride returns the tenant-level rate limit override (if any).
func (s *Service) GetTenantRateLimitOverride(ctx context.Context, tenantID uuid.UUID) (limits.LimitConfig, bool, error) {
	if s == nil || s.queries == nil {
		return limits.LimitConfig{}, false, ErrServiceUnavailable
	}
	record, err := s.queries.GetTenantRateLimit(ctx, toPgUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return limits.LimitConfig{}, false, nil
		}
		return limits.LimitConfig{}, false, err
	}
	cfg := limits.LimitConfig{
		RequestsPerMinute: int(record.RequestsPerMinute),
		TokensPerMinute:   int(record.TokensPerMinute),
		ParallelRequests:  int(record.ParallelRequests),
	}
	return cfg, true, nil
}

// UpsertTenantRateLimitOverride stores a tenant-specific override.
func (s *Service) UpsertTenantRateLimitOverride(ctx context.Context, tenantID uuid.UUID, req limits.LimitConfig) (limits.LimitConfig, error) {
	if s == nil || s.queries == nil {
		return limits.LimitConfig{}, ErrServiceUnavailable
	}
	if req.RequestsPerMinute <= 0 || req.TokensPerMinute <= 0 || req.ParallelRequests <= 0 {
		return limits.LimitConfig{}, ErrInvalidRateLimit
	}
	record, err := s.queries.UpsertTenantRateLimit(ctx, db.UpsertTenantRateLimitParams{
		TenantID:          toPgUUID(tenantID),
		RequestsPerMinute: int32(req.RequestsPerMinute),
		TokensPerMinute:   int32(req.TokensPerMinute),
		ParallelRequests:  int32(req.ParallelRequests),
	})
	if err != nil {
		return limits.LimitConfig{}, err
	}
	cfg := limits.LimitConfig{
		RequestsPerMinute: int(record.RequestsPerMinute),
		TokensPerMinute:   int(record.TokensPerMinute),
		ParallelRequests:  int(record.ParallelRequests),
	}
	if s.setTenantRate != nil {
		s.setTenantRate(tenantID, &cfg)
	}
	return cfg, nil
}

// DeleteTenantRateLimitOverride removes the tenant-level override (if set).
func (s *Service) DeleteTenantRateLimitOverride(ctx context.Context, tenantID uuid.UUID) error {
	if s == nil || s.queries == nil {
		return ErrServiceUnavailable
	}
	if err := s.queries.DeleteTenantRateLimit(ctx, toPgUUID(tenantID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if s.setTenantRate != nil {
		s.setTenantRate(tenantID, nil)
	}
	return nil
}

func (s *Service) normalizeModelAliases(ctx context.Context, aliases []string) ([]string, error) {
	if len(aliases) == 0 {
		return nil, ErrInvalidModelList
	}
	unique := make(map[string]string, len(aliases))
	finalList := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		trimmed := strings.TrimSpace(alias)
		if trimmed == "" {
			return nil, ErrInvalidModelList
		}
		norm := strings.ToLower(trimmed)
		if _, exists := unique[norm]; exists {
			continue
		}
		if _, err := s.queries.GetModelByAlias(ctx, trimmed); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: %s", ErrModelNotFound, trimmed)
			}
			return nil, err
		}
		unique[norm] = trimmed
		finalList = append(finalList, trimmed)
	}
	if len(finalList) == 0 {
		return nil, ErrInvalidModelList
	}
	sort.Strings(finalList)
	return finalList, nil
}

func (s *Service) ensureUser(ctx context.Context, email string) (db.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			user, err = s.queries.CreateUser(ctx, db.CreateUserParams{
				Email: email,
				Name:  email,
			})
			if err != nil {
				return db.User{}, err
			}
		} else {
			return db.User{}, err
		}
	}
	if s.accounts != nil && !user.PersonalTenantID.Valid {
		updated, _, err := s.accounts.EnsurePersonalTenant(ctx, user)
		if err != nil {
			return db.User{}, err
		}
		user = updated
	}
	return user, nil
}

func (s *Service) upsertMembershipRecord(ctx context.Context, tenantID pgtype.UUID, userID pgtype.UUID, role db.MembershipRole) (db.TenantMembership, error) {
	existing, err := s.queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.queries.AddTenantMembership(ctx, db.AddTenantMembershipParams{
				TenantID: tenantID,
				UserID:   userID,
				Role:     role,
			})
		}
		return db.TenantMembership{}, err
	}
	if existing.Role == role {
		return existing, nil
	}
	return s.queries.UpdateTenantMembershipRole(ctx, db.UpdateTenantMembershipRoleParams{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
	})
}

func (s *Service) computeUsageUSD(ctx context.Context, tenantID uuid.UUID, schedule string, now time.Time) (float64, error) {
	start, end := budgetWindowBounds(now, schedule)
	row, err := s.queries.SumUsageForTenant(ctx, db.SumUsageForTenantParams{
		TenantID: toPgUUID(tenantID),
		Ts:       pgtype.Timestamptz{Time: start, Valid: true},
		Ts_2:     pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	return microsToUSD(row.TotalCostUsdMicros), nil
}

func budgetWindowBounds(now time.Time, schedule string) (time.Time, time.Time) {
	nowUTC := now.UTC()
	normalized := config.NormalizeBudgetRefreshSchedule(schedule)
	switch normalized {
	case "weekly":
		year, month, day := nowUTC.Date()
		weekday := int(nowUTC.Weekday())
		delta := (weekday + 6) % 7
		start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -delta)
		return start, start.AddDate(0, 0, 7)
	case "calendar_month":
		year, month, _ := nowUTC.Date()
		start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		if days, ok := config.BudgetRollingWindowDays(normalized); ok && days > 0 {
			start := nowUTC.AddDate(0, 0, -days)
			return start, nowUTC
		}
	}
	return nowUTC.AddDate(0, 0, -30), nowUTC
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func microsToUSD(micros int64) float64 {
	return float64(micros) / 1_000_000
}

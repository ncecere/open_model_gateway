package usage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
)

var (
	ErrInvalidPeriod        = timeutil.ErrInvalidPeriod
	ErrInvalidBreakdownType = errors.New("invalid breakdown group")
	ErrScopeForbidden       = errors.New("scope unavailable")
	ErrInvalidTimezone      = errors.New("invalid timezone")
	ErrNoEntitiesSelected   = errors.New("at least one entity must be requested")
	ErrEntityLimitExceeded  = errors.New("too many entities requested")
	ErrInvalidRange         = errors.New("invalid comparison range")
)

// Service exposes usage aggregation helpers shared across admin and user surfaces.
type Service struct {
	queries  *db.Queries
	timezone *time.Location
}

const (
	maxUsageCompareSeries  = 10
	maxCustomCompareDays   = 180
	maxCustomCompareWindow = time.Duration(maxCustomCompareDays) * 24 * time.Hour
)

func NewService(queries *db.Queries, timezone *time.Location) *Service {
	return &Service{queries: queries, timezone: timezone}
}

func (s *Service) location() *time.Location {
	if s == nil || s.timezone == nil {
		return time.UTC
	}
	return s.timezone
}

func (s *Service) newWindow(period string, overrideTZ string) (timeutil.Window, error) {
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(overrideTZ); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return timeutil.Window{}, ErrInvalidTimezone
		}
		loc = custom
	}
	now := time.Now().In(loc)
	return timeutil.NewWindow(period, now, loc)
}

// UserSummary aggregates usage for a single user across their personal tenant and memberships.
type UserSummary struct {
	Period         string              `json:"period"`
	Start          string              `json:"start"`
	End            string              `json:"end"`
	Timezone       string              `json:"timezone"`
	Totals         UsageTotals         `json:"totals"`
	Personal       *UserTenantUsage    `json:"personal,omitempty"`
	PersonalSeries []UsagePoint        `json:"personal_series,omitempty"`
	Memberships    []UserTenantUsage   `json:"memberships"`
	PersonalKeys   []APIKeyUsageDigest `json:"personal_api_keys,omitempty"`
	RecentRequests []RecentRequest     `json:"recent_requests,omitempty"`
	Scopes         []UsageScope        `json:"scopes,omitempty"`
	SelectedScope  *UserScopeDetail    `json:"selected_scope,omitempty"`
}

// UsageTotals describes aggregate request/token/cost counts.
type UsageTotals struct {
	Requests  int64   `json:"requests"`
	Tokens    int64   `json:"tokens"`
	CostCents int64   `json:"cost_cents"`
	CostUSD   float64 `json:"cost_usd"`
}

// UserTenantUsage represents usage for a tenant the user belongs to.
type UserTenantUsage struct {
	TenantID   string  `json:"tenant_id"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	Status     string  `json:"status"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	CostCents  int64   `json:"cost_cents"`
	CostUSD    float64 `json:"cost_usd"`
	IsPersonal bool    `json:"is_personal"`
}

type UsageScopeKind string

const (
	UsageScopePersonal UsageScopeKind = "personal"
	UsageScopeTenant   UsageScopeKind = "tenant"
)

type UsageScope struct {
	ID       string         `json:"id"`
	Kind     UsageScopeKind `json:"kind"`
	TenantID *string        `json:"tenant_id,omitempty"`
	Name     string         `json:"name"`
	Role     string         `json:"role,omitempty"`
	Status   string         `json:"status,omitempty"`
	Totals   UsageTotals    `json:"totals"`
}

type UserScopeDetail struct {
	Scope          UsageScope          `json:"scope"`
	Series         []UsagePoint        `json:"series"`
	APIKeys        []APIKeyUsageDigest `json:"api_keys"`
	RecentRequests []RecentRequest     `json:"recent_requests"`
}

type scopeEntry struct {
	scope  UsageScope
	tenant uuid.UUID
}

// UsagePoint is a daily time-series datapoint.
type UsagePoint struct {
	Date      string  `json:"date"`
	Requests  int64   `json:"requests"`
	Tokens    int64   `json:"tokens"`
	CostCents int64   `json:"cost_cents"`
	CostUSD   float64 `json:"cost_usd"`
}

type UsageCompareSeriesKind string

const (
	UsageCompareSeriesTenant UsageCompareSeriesKind = "tenant"
	UsageCompareSeriesModel  UsageCompareSeriesKind = "model"
	UsageCompareSeriesUser   UsageCompareSeriesKind = "user"
)

type UsageCompareSeries struct {
	Kind         UsageCompareSeriesKind `json:"kind"`
	ID           string                 `json:"id"`
	Label        string                 `json:"label"`
	TenantID     *string                `json:"tenant_id,omitempty"`
	TenantStatus string                 `json:"tenant_status,omitempty"`
	TenantKind   string                 `json:"tenant_kind,omitempty"`
	UserID       *string                `json:"user_id,omitempty"`
	UserEmail    string                 `json:"user_email,omitempty"`
	UserName     string                 `json:"user_name,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Totals       UsageTotals            `json:"totals"`
	Points       []UsagePoint           `json:"points"`
	ActiveStart  *string                `json:"active_start,omitempty"`
	ActiveEnd    *string                `json:"active_end,omitempty"`
}

type MultiEntityUsage struct {
	Period   string               `json:"period"`
	Start    string               `json:"start"`
	End      string               `json:"end"`
	Timezone string               `json:"timezone"`
	Series   []UsageCompareSeries `json:"series"`
}

type TenantDailyUsageResponse struct {
	TenantID string                `json:"tenant_id"`
	Start    string                `json:"start"`
	End      string                `json:"end"`
	Timezone string                `json:"timezone"`
	Days     []TenantDailyUsageDay `json:"days"`
}

type TenantDailyUsageDay struct {
	Date      string                `json:"date"`
	Requests  int64                 `json:"requests"`
	Tokens    int64                 `json:"tokens"`
	CostCents int64                 `json:"cost_cents"`
	CostUSD   float64               `json:"cost_usd"`
	Keys      []TenantDailyKeyUsage `json:"keys"`
}

type TenantDailyKeyUsage struct {
	APIKeyID     string  `json:"api_key_id"`
	APIKeyName   string  `json:"api_key_name"`
	APIKeyPrefix string  `json:"api_key_prefix"`
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	CostCents    int64   `json:"cost_cents"`
	CostUSD      float64 `json:"cost_usd"`
}

type UserDailyUsageResponse struct {
	UserID   string              `json:"user_id"`
	Start    string              `json:"start"`
	End      string              `json:"end"`
	Timezone string              `json:"timezone"`
	Days     []UserDailyUsageDay `json:"days"`
}

type UserDailyUsageDay struct {
	Date      string                 `json:"date"`
	Requests  int64                  `json:"requests"`
	Tokens    int64                  `json:"tokens"`
	CostCents int64                  `json:"cost_cents"`
	CostUSD   float64                `json:"cost_usd"`
	Tenants   []UserDailyTenantUsage `json:"tenants"`
}

type UserDailyTenantUsage struct {
	TenantID   string  `json:"tenant_id"`
	TenantName string  `json:"tenant_name"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	CostCents  int64   `json:"cost_cents"`
	CostUSD    float64 `json:"cost_usd"`
}

type ModelDailyUsageResponse struct {
	ModelAlias string               `json:"model_alias"`
	Start      string               `json:"start"`
	End        string               `json:"end"`
	Timezone   string               `json:"timezone"`
	Days       []ModelDailyUsageDay `json:"days"`
}

type ModelDailyUsageDay struct {
	Date      string                  `json:"date"`
	Requests  int64                   `json:"requests"`
	Tokens    int64                   `json:"tokens"`
	CostCents int64                   `json:"cost_cents"`
	CostUSD   float64                 `json:"cost_usd"`
	Tenants   []ModelDailyTenantUsage `json:"tenants"`
}

type ModelDailyTenantUsage struct {
	TenantID   string  `json:"tenant_id"`
	TenantName string  `json:"tenant_name"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	CostCents  int64   `json:"cost_cents"`
	CostUSD    float64 `json:"cost_usd"`
}

type CompareUsageParams struct {
	Period       string
	Timezone     string
	Start        *time.Time
	End          *time.Time
	TenantIDs    []uuid.UUID
	ModelAliases []string
	UserIDs      []uuid.UUID
	TenantScope  []uuid.UUID
}

// AdminUsageSummary mirrors the admin usage summary payload.
type AdminUsageSummary struct {
	Period         string       `json:"period"`
	Start          string       `json:"start"`
	End            string       `json:"end"`
	Timezone       string       `json:"timezone"`
	TotalRequests  int64        `json:"total_requests"`
	TotalTokens    int64        `json:"total_tokens"`
	TotalCostCents int64        `json:"total_cost_cents"`
	TotalCostUSD   float64      `json:"total_cost_usd"`
	Points         []UsagePoint `json:"points"`
	TenantID       *string      `json:"tenant_id,omitempty"`
}

// AdminBreakdownParams configures the admin usage breakdown query.
type AdminBreakdownParams struct {
	Group         string
	Period        string
	Limit         int
	EntityID      string
	Timezone      string
	StartOverride *time.Time
	EndOverride   *time.Time
}

// AdminBreakdownItem represents an item row in the breakdown response.
type AdminBreakdownItem struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Requests  int64   `json:"requests"`
	Tokens    int64   `json:"tokens"`
	CostCents int64   `json:"cost_cents"`
	CostUSD   float64 `json:"cost_usd"`
}

// AdminBreakdownSeries captures the time-series for the selected entity.
type AdminBreakdownSeries struct {
	ID       string       `json:"id"`
	Label    string       `json:"label"`
	Points   []UsagePoint `json:"points"`
	Timezone string       `json:"timezone"`
}

// AdminBreakdown is the full response structure for usage breakdown.
type AdminBreakdown struct {
	Group    string               `json:"group"`
	Period   string               `json:"period"`
	Start    string               `json:"start"`
	End      string               `json:"end"`
	Timezone string               `json:"timezone"`
	Items    []AdminBreakdownItem `json:"items"`
	Series   AdminBreakdownSeries `json:"series"`
}

// APIKeyUsageSummary aggregates usage for a single API key over a time window.
type APIKeyUsageSummary struct {
	APIKeyID string       `json:"api_key_id"`
	Period   string       `json:"period"`
	Start    string       `json:"start"`
	End      string       `json:"end"`
	Timezone string       `json:"timezone"`
	Totals   UsageTotals  `json:"totals"`
	Series   []UsagePoint `json:"series"`
}

func (t *UsageTotals) addTotals(other UsageTotals) {
	t.Requests += other.Requests
	t.Tokens += other.Tokens
	t.CostCents += other.CostCents
	t.CostUSD += other.CostUSD
}

// SummarizeUserUsage returns usage aggregates for the provided user and period (e.g., "7d", "30d") or a custom range when start/end overrides are supplied.
func (s *Service) SummarizeUserUsage(ctx context.Context, user db.User, period string, tenantFilter *uuid.UUID, timezone string, startOverride, endOverride *time.Time) (UserSummary, error) {
	if s == nil || s.queries == nil {
		return UserSummary{}, errors.New("usage service not initialized")
	}

	var (
		window timeutil.Window
		err    error
	)
	if startOverride != nil && endOverride != nil {
		loc := timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(timezone); tz != "" {
			custom, tzErr := time.LoadLocation(tz)
			if tzErr != nil {
				return UserSummary{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start := startOverride.In(loc)
		end := endOverride.In(loc)
		if !end.After(start) {
			return UserSummary{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return UserSummary{}, ErrInvalidRange
		}
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		label := fmt.Sprintf("custom_%dd", days)
		window, err = timeutil.NewWindowFromRange(start, end, loc, label)
		if err != nil {
			return UserSummary{}, err
		}
	} else {
		window, err = s.newWindow(period, timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return UserSummary{}, ErrInvalidTimezone
			}
			return UserSummary{}, ErrInvalidPeriod
		}
	}
	zone := window.Timezone()

	summary := UserSummary{
		Period:      window.Period(),
		Start:       window.StartString(),
		End:         window.EndString(),
		Timezone:    zone,
		Totals:      UsageTotals{},
		Memberships: make([]UserTenantUsage, 0),
	}

	entries := make([]scopeEntry, 0)

	var personalTenant uuid.UUID
	if user.PersonalTenantID.Valid {
		if id, err := uuidFromPg(user.PersonalTenantID); err == nil {
			personalTenant = id
		}
	}

	if personalTenant != uuid.Nil {
		personalTotals, err := s.sumUsageForUserTenant(ctx, user, personalTenant, window)
		if err != nil {
			return UserSummary{}, err
		}
		personalUsage := UserTenantUsage{
			TenantID:   personalTenant.String(),
			Name:       user.Email,
			Role:       string(db.MembershipRoleOwner),
			Status:     "active",
			Requests:   personalTotals.Requests,
			Tokens:     personalTotals.Tokens,
			CostCents:  personalTotals.CostCents,
			CostUSD:    personalTotals.CostUSD,
			IsPersonal: true,
		}
		summary.Personal = &personalUsage
		scope := UsageScope{
			ID:     "personal",
			Kind:   UsageScopePersonal,
			Name:   "Personal",
			Totals: personalTotals,
		}
		idCopy := personalTenant.String()
		scope.TenantID = &idCopy
		scope.Role = string(db.MembershipRoleOwner)
		scope.Status = "active"
		entries = append(entries, scopeEntry{scope: scope, tenant: personalTenant})
	} else {
		zeroUsage := UserTenantUsage{
			TenantID:   "",
			Name:       user.Email,
			Role:       string(db.MembershipRoleOwner),
			Status:     "active",
			IsPersonal: true,
		}
		summary.Personal = &zeroUsage
		scope := UsageScope{
			ID:     "personal",
			Kind:   UsageScopePersonal,
			Name:   "Personal",
			Role:   string(db.MembershipRoleOwner),
			Status: "active",
			Totals: UsageTotals{},
		}
		entries = append(entries, scopeEntry{scope: scope, tenant: uuid.Nil})
	}

	ownedTenants, err := s.queries.ListUserOwnedTenants(ctx, user.ID)
	if err != nil {
		return UserSummary{}, err
	}

	for _, row := range ownedTenants {
		tenantUUID, err := uuidFromPg(row.TenantID)
		if err != nil {
			continue
		}
		tenantTotals, err := s.sumUsageForUserTenant(ctx, user, tenantUUID, window)
		if err != nil {
			return UserSummary{}, err
		}
		tenantIDStr := tenantUUID.String()
		scope := UsageScope{
			ID:       tenantIDStr,
			Kind:     UsageScopeTenant,
			TenantID: &tenantIDStr,
			Name:     row.Name,
			Role:     string(row.Role),
			Status:   string(row.Status),
			Totals:   tenantTotals,
		}
		entries = append(entries, scopeEntry{scope: scope, tenant: tenantUUID})

		summary.Memberships = append(summary.Memberships, UserTenantUsage{
			TenantID:   tenantIDStr,
			Name:       row.Name,
			Role:       string(row.Role),
			Status:     string(row.Status),
			Requests:   tenantTotals.Requests,
			Tokens:     tenantTotals.Tokens,
			CostCents:  tenantTotals.CostCents,
			CostUSD:    tenantTotals.CostUSD,
			IsPersonal: false,
		})
	}

	selectedEntry := entries[0]
	if tenantFilter != nil && *tenantFilter != uuid.Nil {
		found := false
		for _, entry := range entries {
			if entry.tenant == *tenantFilter {
				selectedEntry = entry
				found = true
				break
			}
		}
		if !found {
			return UserSummary{}, ErrScopeForbidden
		}
	}

	for _, entry := range entries {
		summary.Totals.addTotals(entry.scope.Totals)
	}

	detail, err := s.buildScopeDetail(ctx, user, selectedEntry, window)
	if err != nil {
		return UserSummary{}, err
	}

	scopes := make([]UsageScope, 0, len(entries))
	for _, entry := range entries {
		scopes = append(scopes, entry.scope)
	}
	summary.Scopes = scopes
	summary.SelectedScope = &detail

	summary.PersonalSeries = detail.Series
	summary.PersonalKeys = detail.APIKeys
	summary.RecentRequests = detail.RecentRequests

	return summary, nil
}

// SummarizeAPIKeyUsage aggregates usage for a single API key owned by the caller over the requested period.
func (s *Service) SummarizeAPIKeyUsage(ctx context.Context, key db.ApiKey, period, timezone string) (APIKeyUsageSummary, error) {
	if s == nil || s.queries == nil {
		return APIKeyUsageSummary{}, errors.New("usage service not initialized")
	}
	window, err := s.newWindow(period, timezone)
	if err != nil {
		if errors.Is(err, ErrInvalidTimezone) {
			return APIKeyUsageSummary{}, ErrInvalidTimezone
		}
		return APIKeyUsageSummary{}, ErrInvalidPeriod
	}
	loc := window.Location()
	zone := window.Timezone()
	start, end := window.Bounds()
	sum, err := s.queries.SumUsageForAPIKey(ctx, db.SumUsageForAPIKeyParams{
		ApiKeyID: key.ID,
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return APIKeyUsageSummary{}, err
	}
	rows, err := s.queries.AggregateAPIKeyUsageDaily(ctx, db.AggregateAPIKeyUsageDailyParams{
		ApiKeyID: key.ID,
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
		Column4:  zone,
	})
	if err != nil {
		return APIKeyUsageSummary{}, err
	}
	series := buildAPIKeyUsagePoints(start, end, rows, loc)
	return APIKeyUsageSummary{
		APIKeyID: pgUUIDString(key.ID),
		Period:   window.Period(),
		Start:    window.StartString(),
		End:      window.EndString(),
		Timezone: zone,
		Totals: UsageTotals{
			Requests:  sum.TotalRequests,
			Tokens:    sum.TotalTokens,
			CostCents: sum.TotalCostCents,
			CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		},
		Series: series,
	}, nil
}

func (s *Service) buildTenantCompareSeries(ctx context.Context, tenantIDs []uuid.UUID, start, end time.Time, zone string, loc *time.Location) ([]UsageCompareSeries, error) {
	if len(tenantIDs) == 0 {
		return nil, nil
	}
	pgIDs := toPgUUIDArray(tenantIDs)
	metaRows, err := s.queries.ListTenantsByIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ListTenantsByIDsRow, len(metaRows))
	for _, row := range metaRows {
		meta[pgUUIDString(row.ID)] = row
	}
	totalRows, err := s.queries.SumUsageByTenants(ctx, db.SumUsageByTenantsParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		id := pgUUIDString(row.TenantID)
		totalMap[id] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByTenants(ctx, db.AggregateUsageDailyByTenantsParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return nil, err
	}
	daily := make(map[string]map[int64]dailyAggregate, len(tenantIDs))
	for _, row := range dailyRows {
		id := pgUUIDString(row.TenantID)
		if id == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := daily[id]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			daily[id] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	series := make([]UsageCompareSeries, 0, len(tenantIDs))
	for _, id := range tenantIDs {
		idStr := id.String()
		label := idStr
		var tenantStatus, tenantKind string
		if row, ok := meta[idStr]; ok {
			label = row.Name
			tenantStatus = string(row.Status)
			tenantKind = string(row.Kind)
			if row.Kind == db.TenantKindPersonal {
				label = "Personal"
			}
		}
		totals := totalMap[idStr]
		points := buildUsagePointsFromDailyMap(start, end, loc, daily[idStr])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		idCopy := idStr
		series = append(series, UsageCompareSeries{
			Kind:         UsageCompareSeriesTenant,
			ID:           idStr,
			Label:        label,
			TenantID:     &idCopy,
			TenantStatus: tenantStatus,
			TenantKind:   tenantKind,
			Totals:       totals,
			Points:       points,
			ActiveStart:  activeStart,
			ActiveEnd:    activeEnd,
		})
	}
	return series, nil
}

func (s *Service) buildModelCompareSeries(ctx context.Context, aliases []string, start, end time.Time, zone string, loc *time.Location, scope []pgtype.UUID) ([]UsageCompareSeries, error) {
	if len(aliases) == 0 {
		return nil, nil
	}
	metaRows, err := s.queries.ListModelCatalogByAliases(ctx, aliases)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ModelCatalog, len(metaRows))
	for _, row := range metaRows {
		meta[row.Alias] = row
	}
	totalRows, err := s.queries.SumUsageByModels(ctx, db.SumUsageByModelsParams{
		Column1: aliases,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: scope,
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		totalMap[row.ModelAlias] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByModels(ctx, db.AggregateUsageDailyByModelsParams{
		Column1: aliases,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
		Column5: scope,
	})
	if err != nil {
		return nil, err
	}
	daily := make(map[string]map[int64]dailyAggregate, len(aliases))
	for _, row := range dailyRows {
		alias := row.ModelAlias
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := daily[alias]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			daily[alias] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	series := make([]UsageCompareSeries, 0, len(aliases))
	for _, alias := range aliases {
		label := alias
		provider := ""
		if row, ok := meta[alias]; ok {
			provider = catalog.NormalizeProviderSlug(row.Provider)
		}
		totals := totalMap[alias]
		points := buildUsagePointsFromDailyMap(start, end, loc, daily[alias])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		series = append(series, UsageCompareSeries{
			Kind:        UsageCompareSeriesModel,
			ID:          alias,
			Label:       label,
			Provider:    provider,
			Totals:      totals,
			Points:      points,
			ActiveStart: activeStart,
			ActiveEnd:   activeEnd,
		})
	}
	return series, nil
}

func (s *Service) buildUserCompareSeries(ctx context.Context, userIDs []uuid.UUID, start, end time.Time, zone string, loc *time.Location) ([]UsageCompareSeries, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	pgIDs := toPgUUIDArray(userIDs)
	metaRows, err := s.queries.ListUsersByIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ListUsersByIDsRow, len(metaRows))
	for _, row := range metaRows {
		meta[pgUUIDString(row.ID)] = row
	}
	totalRows, err := s.queries.SumUsageByUsers(ctx, db.SumUsageByUsersParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		id := pgUUIDString(row.UserID)
		totalMap[id] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return nil, err
	}
	userDaily := groupUserDailyAggregates(dailyRows, loc)
	series := make([]UsageCompareSeries, 0, len(userIDs))
	for _, id := range userIDs {
		idStr := id.String()
		metaRow := meta[idStr]
		label := strings.TrimSpace(metaRow.Name)
		if label == "" {
			label = strings.TrimSpace(metaRow.Email)
		}
		if label == "" {
			label = idStr
		}
		totals := totalMap[idStr]
		points := buildUsagePointsFromDailyMap(start, end, loc, userDaily[idStr])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		idCopy := idStr
		series = append(series, UsageCompareSeries{
			Kind:        UsageCompareSeriesUser,
			ID:          idStr,
			Label:       label,
			UserID:      &idCopy,
			UserEmail:   metaRow.Email,
			UserName:    metaRow.Name,
			Totals:      totals,
			Points:      points,
			ActiveStart: activeStart,
			ActiveEnd:   activeEnd,
		})
	}
	return series, nil
}

// CompareUsage returns parallel usage series for the requested tenants/models so dashboards can overlay them.
func (s *Service) CompareUsage(ctx context.Context, params CompareUsageParams) (MultiEntityUsage, error) {
	if s == nil || s.queries == nil {
		return MultiEntityUsage{}, errors.New("usage service not initialized")
	}
	period := strings.TrimSpace(params.Period)
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(params.Timezone)
	tenants := dedupUUIDs(params.TenantIDs)
	aliases := dedupStrings(params.ModelAliases)
	users := dedupUUIDs(params.UserIDs)
	totalRequested := len(tenants) + len(aliases) + len(users)
	if totalRequested == 0 {
		return MultiEntityUsage{}, ErrNoEntitiesSelected
	}
	if totalRequested > maxUsageCompareSeries {
		return MultiEntityUsage{}, ErrEntityLimitExceeded
	}
	var (
		loc         *time.Location
		zone        string
		start, end  time.Time
		periodLabel string
	)
	if params.Start != nil || params.End != nil {
		if params.Start == nil || params.End == nil {
			return MultiEntityUsage{}, ErrInvalidRange
		}
		loc = timeutil.EnsureLocation(s.location())
		if timezone != "" {
			custom, err := time.LoadLocation(timezone)
			if err != nil {
				return MultiEntityUsage{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start = params.Start.In(loc)
		end = params.End.In(loc)
		if !end.After(start) {
			return MultiEntityUsage{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return MultiEntityUsage{}, ErrInvalidRange
		}
		zone = loc.String()
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		periodLabel = fmt.Sprintf("custom_%dd", days)
	} else {
		window, err := s.newWindow(period, timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return MultiEntityUsage{}, ErrInvalidTimezone
			}
			return MultiEntityUsage{}, ErrInvalidPeriod
		}
		loc = window.Location()
		zone = window.Timezone()
		start, end = window.Bounds()
		periodLabel = window.Period()
	}
	scopeIDs := dedupUUIDs(params.TenantScope)
	scopeParam := toPgUUIDArray(scopeIDs)
	result := MultiEntityUsage{
		Period:   periodLabel,
		Start:    windowTimeString(start, loc),
		End:      windowTimeString(end, loc),
		Timezone: zone,
		Series:   make([]UsageCompareSeries, 0, totalRequested),
	}
	if len(tenants) > 0 {
		series, err := s.buildTenantCompareSeries(ctx, tenants, start, end, zone, loc)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	if len(aliases) > 0 {
		series, err := s.buildModelCompareSeries(ctx, aliases, start, end, zone, loc, scopeParam)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	if len(users) > 0 {
		series, err := s.buildUserCompareSeries(ctx, users, start, end, zone, loc)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	return result, nil
}

// TenantDailyUsage aggregates per-day totals and per-key breakdowns for a tenant between start and end.
func (s *Service) TenantDailyUsage(ctx context.Context, tenantID uuid.UUID, start, end time.Time, timezone string) (TenantDailyUsageResponse, error) {
	if s == nil || s.queries == nil {
		return TenantDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	if tenantID == uuid.Nil {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}

	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return TenantDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	dailyRows, err := s.queries.AggregateTenantUsageDaily(ctx, db.AggregateTenantUsageDailyParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(startDay),
		Ts_2:     toPgTime(endDay),
		Column4:  zone,
	})
	if err != nil {
		return TenantDailyUsageResponse{}, err
	}
	keyRows, err := s.queries.AggregateTenantUsageDailyByAPIKeys(ctx, db.AggregateTenantUsageDailyByAPIKeysParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(startDay),
		Ts_2:     toPgTime(endDay),
		Column4:  zone,
	})
	if err != nil {
		return TenantDailyUsageResponse{}, err
	}

	dailyBuckets := make(map[int64]dailyAggregate, len(dailyRows))
	for _, row := range dailyRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		dailyBuckets[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}

	keyBuckets := make(map[int64][]TenantDailyKeyUsage)
	for _, row := range keyRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		keyBuckets[day.Unix()] = append(keyBuckets[day.Unix()], TenantDailyKeyUsage{
			APIKeyID:     pgUUIDString(row.ApiKeyID),
			APIKeyName:   row.ApiKeyName,
			APIKeyPrefix: row.ApiKeyPrefix,
			Requests:     row.Requests,
			Tokens:       row.Tokens,
			CostCents:    row.CostCents,
			CostUSD:      microsToUSD(row.CostUsdMicros),
		})
	}

	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]TenantDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		aggregate := dailyBuckets[key]
		keys := keyBuckets[key]
		if keys == nil {
			keys = make([]TenantDailyKeyUsage, 0)
		}
		if len(keys) > 1 {
			sort.Slice(keys, func(i, j int) bool {
				if keys[i].CostCents == keys[j].CostCents {
					return keys[i].Requests > keys[j].Requests
				}
				return keys[i].CostCents > keys[j].CostCents
			})
		}
		days = append(days, TenantDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Keys:      keys,
		})
	}

	return TenantDailyUsageResponse{
		TenantID: tenantID.String(),
		Start:    windowTimeString(startDay, loc),
		End:      windowTimeString(endDay, loc),
		Timezone: zone,
		Days:     days,
	}, nil
}

// UserDailyUsage aggregates daily totals for a user with per-tenant breakdowns.
func (s *Service) UserDailyUsage(ctx context.Context, userID uuid.UUID, start, end time.Time, timezone string) (UserDailyUsageResponse, error) {
	if s == nil || s.queries == nil {
		return UserDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	if userID == uuid.Nil {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return UserDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	userRows, err := s.queries.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
		Column1: toPgUUIDArray([]uuid.UUID{userID}),
		Ts:      toPgTime(startDay),
		Ts_2:    toPgTime(endDay),
		Column4: zone,
	})
	if err != nil {
		return UserDailyUsageResponse{}, err
	}
	userDaily := groupUserDailyAggregates(userRows, loc)
	dailyTotals := userDaily[userID.String()]

	tenantRows, err := s.queries.AggregateUserUsageDailyByTenants(ctx, db.AggregateUserUsageDailyByTenantsParams{
		OwnerUserID: toPgUUID(userID),
		Ts:          toPgTime(startDay),
		Ts_2:        toPgTime(endDay),
		Column4:     zone,
	})
	if err != nil {
		return UserDailyUsageResponse{}, err
	}
	tenantBuckets := make(map[int64][]UserDailyTenantUsage)
	for _, row := range tenantRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		tenantBuckets[day.Unix()] = append(tenantBuckets[day.Unix()], UserDailyTenantUsage{
			TenantID:   pgUUIDString(row.TenantID),
			TenantName: row.TenantName,
			Requests:   row.Requests,
			Tokens:     row.Tokens,
			CostCents:  row.CostCents,
			CostUSD:    microsToUSD(row.CostUsdMicros),
		})
	}

	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]UserDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		tenants := tenantBuckets[key]
		if tenants == nil {
			tenants = make([]UserDailyTenantUsage, 0)
		} else if len(tenants) > 1 {
			sort.Slice(tenants, func(i, j int) bool {
				if tenants[i].CostCents == tenants[j].CostCents {
					return tenants[i].Requests > tenants[j].Requests
				}
				return tenants[i].CostCents > tenants[j].CostCents
			})
		}
		aggregate, ok := dailyTotals[key]
		if (!ok || aggregate == (dailyAggregate{})) && len(tenants) > 0 {
			aggregate = sumUserTenantUsage(tenants)
		}
		days = append(days, UserDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Tenants:   tenants,
		})
	}

	return UserDailyUsageResponse{
		UserID:   userID.String(),
		Start:    windowTimeString(startDay, loc),
		End:      windowTimeString(endDay, loc),
		Timezone: zone,
		Days:     days,
	}, nil
}

// ModelDailyUsage aggregates daily totals for a model alias with per-tenant breakdowns.
func (s *Service) ModelDailyUsage(ctx context.Context, alias string, start, end time.Time, timezone string, tenantScope []uuid.UUID) (ModelDailyUsageResponse, error) {
	if s == nil || s.queries == nil {
		return ModelDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return ModelDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	scopeIDs := dedupUUIDs(tenantScope)
	scopeParam := toPgUUIDArray(scopeIDs)
	allowedTenants := make(map[string]struct{})
	for _, id := range scopeIDs {
		allowedTenants[id.String()] = struct{}{}
	}
	modelRows, err := s.queries.AggregateUsageDailyByModels(ctx, db.AggregateUsageDailyByModelsParams{
		Column1: []string{alias},
		Ts:      toPgTime(startDay),
		Ts_2:    toPgTime(endDay),
		Column4: zone,
		Column5: scopeParam,
	})
	if err != nil {
		return ModelDailyUsageResponse{}, err
	}
	modelDaily := groupModelDailyAggregates(modelRows, loc)
	dailyTotals := modelDaily[alias]

	tenantRows, err := s.queries.AggregateModelUsageDailyByTenants(ctx, db.AggregateModelUsageDailyByTenantsParams{
		ModelAlias: alias,
		Ts:         toPgTime(startDay),
		Ts_2:       toPgTime(endDay),
		Column4:    zone,
	})
	if err != nil {
		return ModelDailyUsageResponse{}, err
	}
	tenantBuckets := make(map[int64][]ModelDailyTenantUsage)
	for _, row := range tenantRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		tenantID := pgUUIDString(row.TenantID)
		if len(allowedTenants) > 0 {
			if _, ok := allowedTenants[tenantID]; !ok {
				continue
			}
		}
		if tenantID == "" {
			continue
		}
		tenantBuckets[day.Unix()] = append(tenantBuckets[day.Unix()], ModelDailyTenantUsage{
			TenantID:   tenantID,
			TenantName: row.TenantName,
			Requests:   row.Requests,
			Tokens:     row.Tokens,
			CostCents:  row.CostCents,
			CostUSD:    microsToUSD(row.CostUsdMicros),
		})
	}
	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]ModelDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		tenants := tenantBuckets[key]
		if tenants == nil {
			tenants = make([]ModelDailyTenantUsage, 0)
		} else if len(tenants) > 1 {
			sort.Slice(tenants, func(i, j int) bool {
				if tenants[i].CostCents == tenants[j].CostCents {
					return tenants[i].Requests > tenants[j].Requests
				}
				return tenants[i].CostCents > tenants[j].CostCents
			})
		}
		aggregate, ok := dailyTotals[key]
		if (!ok || aggregate == (dailyAggregate{})) && len(tenants) > 0 {
			aggregate = sumModelTenantUsage(tenants)
		}
		days = append(days, ModelDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Tenants:   tenants,
		})
	}

	return ModelDailyUsageResponse{
		ModelAlias: alias,
		Start:      windowTimeString(startDay, loc),
		End:        windowTimeString(endDay, loc),
		Timezone:   zone,
		Days:       days,
	}, nil
}

// SummarizeAdminUsage aggregates system-wide or tenant-scoped usage for admin dashboards.
func (s *Service) SummarizeAdminUsage(ctx context.Context, period string, tenantID *uuid.UUID, timezone string, startOverride, endOverride *time.Time) (AdminUsageSummary, error) {
	if s == nil || s.queries == nil {
		return AdminUsageSummary{}, errors.New("usage service not initialized")
	}
	var (
		start, end  time.Time
		loc         *time.Location
		zone        string
		periodLabel string
	)
	if startOverride != nil && endOverride != nil {
		loc = timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(timezone); tz != "" {
			custom, err := time.LoadLocation(tz)
			if err != nil {
				return AdminUsageSummary{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start = startOverride.In(loc)
		end = endOverride.In(loc)
		if !end.After(start) {
			return AdminUsageSummary{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return AdminUsageSummary{}, ErrInvalidRange
		}
		zone = loc.String()
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		periodLabel = fmt.Sprintf("custom_%dd", days)
	} else {
		window, err := s.newWindow(period, timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return AdminUsageSummary{}, ErrInvalidTimezone
			}
			return AdminUsageSummary{}, ErrInvalidPeriod
		}
		loc = window.Location()
		zone = window.Timezone()
		start, end = window.Bounds()
		periodLabel = window.Period()
	}

	var tenantParam pgtype.UUID
	var tenantRef *string
	if tenantID != nil && *tenantID != uuid.Nil {
		tenantParam = toPgUUID(*tenantID)
		idCopy := tenantID.String()
		tenantRef = &idCopy
	}

	sum, err := s.queries.SumUsage(ctx, db.SumUsageParams{
		Column1: tenantParam,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AdminUsageSummary{}, err
	}

	dailyRows, err := s.queries.AggregateUsageDaily(ctx, db.AggregateUsageDailyParams{
		Column1: tenantParam,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return AdminUsageSummary{}, err
	}

	points := buildAggregateUsagePoints(start, end, dailyRows, loc)
	return AdminUsageSummary{
		Period:         periodLabel,
		Start:          start.In(loc).Format(time.RFC3339),
		End:            end.In(loc).Format(time.RFC3339),
		Timezone:       zone,
		TotalRequests:  sum.TotalRequests,
		TotalTokens:    sum.TotalTokens,
		TotalCostCents: sum.TotalCostCents,
		TotalCostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		Points:         points,
		TenantID:       tenantRef,
	}, nil
}

// BreakdownAdminUsage returns aggregate lists + series grouped by tenant or model.
func (s *Service) BreakdownAdminUsage(ctx context.Context, params AdminBreakdownParams) (AdminBreakdown, error) {
	if s == nil || s.queries == nil {
		return AdminBreakdown{}, errors.New("usage service not initialized")
	}
	var (
		window timeutil.Window
		err    error
	)
	if params.StartOverride != nil && params.EndOverride != nil {
		loc := timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(params.Timezone); tz != "" {
			custom, tzErr := time.LoadLocation(tz)
			if tzErr != nil {
				return AdminBreakdown{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start := params.StartOverride.In(loc)
		end := params.EndOverride.In(loc)
		if !end.After(start) {
			return AdminBreakdown{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return AdminBreakdown{}, ErrInvalidRange
		}
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		window, err = timeutil.NewWindowFromRange(start, end, loc, fmt.Sprintf("custom_%dd", days))
		if err != nil {
			return AdminBreakdown{}, err
		}
	} else {
		window, err = s.newWindow(params.Period, params.Timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return AdminBreakdown{}, ErrInvalidTimezone
			}
			return AdminBreakdown{}, ErrInvalidPeriod
		}
	}
	loc := window.Location()
	zone := window.Timezone()
	start, end := window.Bounds()

	group := strings.TrimSpace(strings.ToLower(params.Group))
	if group == "" {
		group = "tenant"
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 5
	}

	result := AdminBreakdown{
		Group:    group,
		Period:   window.Period(),
		Start:    window.StartString(),
		End:      window.EndString(),
		Timezone: zone,
		Items:    make([]AdminBreakdownItem, 0),
		Series: AdminBreakdownSeries{
			Timezone: zone,
		},
	}

	selected := strings.TrimSpace(params.EntityID)

	switch group {
	case "tenant":
		rows, err := s.queries.AggregateUsageByTenant(ctx, db.AggregateUsageByTenantParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			tenantID, err := uuidFromPg(row.TenantID)
			if err != nil {
				continue
			}
			id := tenantID.String()
			labelMap[id] = row.Name
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        id,
				Label:     row.Name,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			if tenantUUID, err := uuid.Parse(selected); err == nil {
				dailyRows, err := s.queries.AggregateTenantUsageDaily(ctx, db.AggregateTenantUsageDailyParams{
					TenantID: toPgUUID(tenantUUID),
					Ts:       toPgTime(start),
					Ts_2:     toPgTime(end),
					Column4:  zone,
				})
				if err != nil {
					return AdminBreakdown{}, err
				}
				result.Series.ID = selected
				label := labelMap[selected]
				if label == "" {
					label = selected
				}
				result.Series.Label = label
				result.Series.Points = buildTenantUsagePoints(start, end, dailyRows, loc)
			}
		}
	case "model":
		rows, err := s.queries.AggregateUsageByModel(ctx, db.AggregateUsageByModelParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			label := strings.TrimSpace(row.ModelAlias)
			if label == "" {
				label = "unknown"
			}
			labelMap[label] = label
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        label,
				Label:     label,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			dailyRows, err := s.queries.AggregateModelUsageDaily(ctx, db.AggregateModelUsageDailyParams{
				ModelAlias: selected,
				Ts:         toPgTime(start),
				Ts_2:       toPgTime(end),
				Column4:    zone,
			})
			if err != nil {
				return AdminBreakdown{}, err
			}
			result.Series.ID = selected
			label := labelMap[selected]
			if label == "" {
				label = selected
			}
			result.Series.Label = label
			result.Series.Points = buildModelUsagePoints(start, end, dailyRows, loc)
		}
	case "user":
		rows, err := s.queries.AggregateUsageByUser(ctx, db.AggregateUsageByUserParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			userID, err := uuidFromPg(row.UserID)
			if err != nil {
				continue
			}
			id := userID.String()
			label := strings.TrimSpace(row.Name)
			if label == "" {
				label = strings.TrimSpace(row.Email)
			}
			if label == "" {
				label = id
			}
			labelMap[id] = label
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        id,
				Label:     label,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			if userUUID, err := uuid.Parse(selected); err == nil {
				pgIDs := toPgUUIDArray([]uuid.UUID{userUUID})
				dailyRows, err := s.queries.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
					Column1: pgIDs,
					Ts:      toPgTime(start),
					Ts_2:    toPgTime(end),
					Column4: zone,
				})
				if err != nil {
					return AdminBreakdown{}, err
				}
				userDaily := groupUserDailyAggregates(dailyRows, loc)
				result.Series.ID = selected
				label := labelMap[selected]
				if label == "" {
					label = selected
				}
				result.Series.Label = label
				result.Series.Points = buildUsagePointsFromDailyMap(start, end, loc, userDaily[selected])
			}
		}
	default:
		return AdminBreakdown{}, ErrInvalidBreakdownType
	}

	return result, nil
}

func (s *Service) sumUsageForUserTenant(ctx context.Context, user db.User, tenantID uuid.UUID, window timeutil.Window) (UsageTotals, error) {
	if tenantID == uuid.Nil {
		return UsageTotals{}, nil
	}
	start, end := window.Bounds()
	sum, err := s.queries.SumUsageForUserTenant(ctx, db.SumUsageForUserTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(tenantID),
		Ts:          toPgTime(start),
		Ts_2:        toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return UsageTotals{}, err
	}
	return UsageTotals{
		Requests:  sum.TotalRequests,
		Tokens:    sum.TotalTokens,
		CostCents: sum.TotalCostCents,
		CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
	}, nil
}

func (s *Service) buildScopeDetail(ctx context.Context, user db.User, entry scopeEntry, window timeutil.Window) (UserScopeDetail, error) {
	detail := UserScopeDetail{
		Scope:          entry.scope,
		Series:         make([]UsagePoint, 0),
		APIKeys:        make([]APIKeyUsageDigest, 0),
		RecentRequests: make([]RecentRequest, 0),
	}
	if entry.tenant == uuid.Nil {
		return detail, nil
	}
	start, end := window.Bounds()
	zone := window.Timezone()
	loc := window.Location()

	rows, err := s.queries.AggregateUsageDailyForUserTenant(ctx, db.AggregateUsageDailyForUserTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(entry.tenant),
		Ts:          toPgTime(start),
		Ts_2:        toPgTime(end),
		Column5:     zone,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return detail, err
	}
	detail.Series = buildUserScopePoints(start, end, rows, loc)

	apiKeys, reqs, err := s.summarizeKeysForTenant(ctx, user, entry.tenant, window)
	if err != nil {
		return detail, err
	}
	detail.APIKeys = apiKeys
	detail.RecentRequests = reqs
	return detail, nil
}

func buildUserScopePoints(start, end time.Time, rows []db.AggregateUsageDailyForUserTenantRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateUsageDailyForUserTenantRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildTenantUsagePoints(start, end time.Time, rows []db.AggregateTenantUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateTenantUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildAggregateUsagePoints(start, end time.Time, rows []db.AggregateUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildModelUsagePoints(start, end time.Time, rows []db.AggregateModelUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateModelUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildAPIKeyUsagePoints(start, end time.Time, rows []db.AggregateAPIKeyUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateAPIKeyUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

type dailyAggregate struct {
	Requests      int64
	Tokens        int64
	CostCents     int64
	CostUsdMicros int64
}

func buildUsagePointsFromDailyMap(start, end time.Time, loc *time.Location, daily map[int64]dailyAggregate) []UsagePoint {
	if daily == nil {
		daily = map[int64]dailyAggregate{}
	}
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end.Add(-time.Nanosecond), loc)
	if endDay.Before(startDay) {
		endDay = startDay
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var record dailyAggregate
		if row, ok := daily[key]; ok {
			record = row
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  record.Requests,
			Tokens:    record.Tokens,
			CostCents: record.CostCents,
			CostUSD:   microsToUSD(record.CostUsdMicros),
		})
	}
	return points
}

func deriveUsageActiveRange(points []UsagePoint) (*string, *string) {
	var start *string
	var end *string
	for _, point := range points {
		if point.Requests > 0 || point.Tokens > 0 || point.CostCents > 0 || point.CostUSD > 0 {
			if start == nil {
				date := point.Date
				start = &date
			}
			date := point.Date
			end = &date
		}
	}
	return start, end
}

func findPoint(points []UsagePoint, date string) UsagePoint {
	for _, point := range points {
		if point.Date == date {
			return point
		}
	}
	return UsagePoint{Date: date}
}

func groupUserDailyAggregates(rows []db.AggregateUsageDailyByUsersRow, loc *time.Location) map[string]map[int64]dailyAggregate {
	loc = timeutil.EnsureLocation(loc)
	data := make(map[string]map[int64]dailyAggregate, len(rows))
	for _, row := range rows {
		id := pgUUIDString(row.UserID)
		if id == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := data[id]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			data[id] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	return data
}

func groupModelDailyAggregates(rows []db.AggregateUsageDailyByModelsRow, loc *time.Location) map[string]map[int64]dailyAggregate {
	loc = timeutil.EnsureLocation(loc)
	data := make(map[string]map[int64]dailyAggregate, len(rows))
	for _, row := range rows {
		alias := strings.TrimSpace(row.ModelAlias)
		if alias == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := data[alias]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			data[alias] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	return data
}

func sumUserTenantUsage(tenants []UserDailyTenantUsage) dailyAggregate {
	var agg dailyAggregate
	for _, tenant := range tenants {
		agg.Requests += tenant.Requests
		agg.Tokens += tenant.Tokens
		agg.CostCents += tenant.CostCents
		agg.CostUsdMicros += tenant.CostCents * 10_000
	}
	return agg
}

func sumModelTenantUsage(tenants []ModelDailyTenantUsage) dailyAggregate {
	var agg dailyAggregate
	for _, tenant := range tenants {
		agg.Requests += tenant.Requests
		agg.Tokens += tenant.Tokens
		agg.CostCents += tenant.CostCents
		agg.CostUsdMicros += tenant.CostCents * 10_000
	}
	return agg
}

func windowTimeString(ts time.Time, loc *time.Location) string {
	return ts.In(timeutil.EnsureLocation(loc)).Format(time.RFC3339)
}

func (s *Service) summarizeKeysForTenant(ctx context.Context, user db.User, tenantID uuid.UUID, window timeutil.Window) ([]APIKeyUsageDigest, []RecentRequest, error) {
	start, end := window.Bounds()
	loc := window.Location()
	keys, err := s.queries.ListAPIKeysByOwnerAndTenant(ctx, db.ListAPIKeysByOwnerAndTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(tenantID),
	})
	if err != nil {
		return nil, nil, err
	}
	if len(keys) == 0 {
		return nil, nil, nil
	}
	digests := make([]APIKeyUsageDigest, 0, len(keys))
	keyIDs := make([]pgtype.UUID, 0, len(keys))
	nameMap := make(map[string]string, len(keys))
	for _, key := range keys {
		sum, err := s.queries.SumUsageForAPIKey(ctx, db.SumUsageForAPIKeyParams{
			ApiKeyID: key.ID,
			Ts:       toPgTime(start),
			Ts_2:     toPgTime(end),
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, err
		}
		digest := APIKeyUsageDigest{
			APIKeyID:  pgUUIDString(key.ID),
			Name:      key.Name,
			Prefix:    key.Prefix,
			Requests:  sum.TotalRequests,
			Tokens:    sum.TotalTokens,
			CostCents: sum.TotalCostCents,
			CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		}
		if key.LastUsedAt.Valid {
			if ts, err := timeFromPg(key.LastUsedAt); err == nil {
				digest.LastUsedAt = &ts
			}
		}
		digests = append(digests, digest)
		keyIDs = append(keyIDs, key.ID)
		nameMap[pgUUIDString(key.ID)] = key.Name
	}
	if len(keyIDs) == 0 {
		return digests, nil, nil
	}
	reqs, err := s.queries.ListRecentRequestsByAPIKeys(ctx, db.ListRecentRequestsByAPIKeysParams{
		Column1: keyIDs,
		Limit:   10,
	})
	if err != nil {
		return digests, nil, err
	}
	if len(reqs) == 0 {
		return digests, nil, nil
	}
	recent := make([]RecentRequest, 0, len(reqs))
	for _, req := range reqs {
		idStr := pgUUIDString(req.ID)
		keyStr := pgUUIDString(req.ApiKeyID)
		var ts time.Time
		if req.Ts.Valid {
			ts = req.Ts.Time.In(loc)
		} else {
			ts = time.Time{}
		}
		var errCode *string
		if req.ErrorCode.Valid {
			v := req.ErrorCode.String
			errCode = &v
		}
		record := RecentRequest{
			ID:         idStr,
			APIKeyID:   keyStr,
			APIKeyName: nameMap[keyStr],
			ModelAlias: req.ModelAlias,
			Provider:   req.Provider,
			Status:     req.Status,
			LatencyMS:  req.LatencyMs,
			CostCents:  req.CostCents,
			CostUSD:    microsToUSD(req.CostUsdMicros),
			Timestamp:  ts,
			ErrorCode:  errCode,
		}
		recent = append(recent, record)
	}
	return digests, recent, nil
}

func dedupUUIDs(ids []uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func dedupStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		alias := strings.TrimSpace(value)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		result = append(result, alias)
	}
	return result
}

func toPgUUIDArray(ids []uuid.UUID) []pgtype.UUID {
	if len(ids) == 0 {
		return []pgtype.UUID{}
	}
	arr := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		arr = append(arr, toPgUUID(id))
	}
	return arr
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func microsToUSD(micros int64) float64 {
	return float64(micros) / 1_000_000
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

func int64FromAny(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case int:
		return int64(v)
	case uint64:
		if v > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case pgtype.Int8:
		if v.Valid {
			return v.Int64
		}
		return 0
	case pgtype.Numeric:
		if !v.Valid {
			return 0
		}
		floatVal, err := v.Float64Value()
		if err != nil || !floatVal.Valid {
			return 0
		}
		return int64(floatVal.Float64)
	case nil:
		return 0
	default:
		return 0
	}
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgUUIDString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	val, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return ""
	}
	return val.String()
}

// APIKeyUsageDigest summarizes totals for list responses.
type APIKeyUsageDigest struct {
	APIKeyID   string     `json:"api_key_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Requests   int64      `json:"requests"`
	Tokens     int64      `json:"tokens"`
	CostCents  int64      `json:"cost_cents"`
	CostUSD    float64    `json:"cost_usd"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// RecentRequest captures metadata for latest requests across personal keys.
type RecentRequest struct {
	ID         string    `json:"id"`
	APIKeyID   string    `json:"api_key_id"`
	APIKeyName string    `json:"api_key_name"`
	ModelAlias string    `json:"model_alias"`
	Provider   string    `json:"provider"`
	Status     int32     `json:"status"`
	LatencyMS  int32     `json:"latency_ms"`
	CostCents  int64     `json:"cost_cents"`
	CostUSD    float64   `json:"cost_usd"`
	Timestamp  time.Time `json:"timestamp"`
	ErrorCode  *string   `json:"error_code,omitempty"`
}

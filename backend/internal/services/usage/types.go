package usage

import (
	"errors"
	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	"time"
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

// ModelPerformanceStats captures recent performance data for a model alias.
type ModelPerformanceStats struct {
	Alias                  string  `json:"alias"`
	Requests               int64   `json:"requests"`
	Tokens                 int64   `json:"tokens"`
	ThroughputTokensPerSec float64 `json:"throughput_tokens_per_sec"`
	AvgLatencyMs           float64 `json:"avg_latency_ms"`
	P95LatencyMs           float64 `json:"p95_latency_ms"`
	LatencySampleCount     int64   `json:"latency_sample_count"`
}

const (
	maxUsageCompareSeries  = 10
	maxCustomCompareDays   = 180
	maxCustomCompareWindow = time.Duration(maxCustomCompareDays) * 24 * time.Hour
)

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

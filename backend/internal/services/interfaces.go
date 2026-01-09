// Package services provides business logic interfaces for the Open Model Gateway.
// These interfaces allow for dependency injection and easier unit testing.
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// TenantService defines the interface for tenant management operations.
type TenantService interface {
	GetTenantByID(ctx context.Context, id uuid.UUID) (*db.Tenant, error)
	GetTenantByName(ctx context.Context, name string) (*db.Tenant, error)
}

// UsageService defines the interface for usage tracking operations.
type UsageService interface {
	RecordUsage(ctx context.Context, params RecordUsageParams) error
	GetTenantUsage(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (UsageSummary, error)
}

// RecordUsageParams holds parameters for recording usage.
type RecordUsageParams struct {
	TenantID      uuid.UUID
	APIKeyID      uuid.UUID
	UserID        pgtype.UUID
	Model         string
	Provider      string
	InputTokens   int64
	OutputTokens  int64
	TotalTokens   int64
	CostUSD       decimal.Decimal
	RequestType   string
	StatusCode    int
	LatencyMs     int64
	IdempotencyID string
}

// UsageSummary represents aggregated usage data.
type UsageSummary struct {
	TotalRequests   int64
	TotalTokens     int64
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCostUSD    decimal.Decimal
}

// APIKeyService defines the interface for API key management operations.
type APIKeyService interface {
	GetAPIKeyByPrefix(ctx context.Context, prefix string) (*db.ApiKey, error)
	ValidateAPIKey(ctx context.Context, prefix, hash string) (*db.ApiKey, error)
	ListAPIKeysForTenant(ctx context.Context, tenantID uuid.UUID) ([]db.ApiKey, error)
}

// BudgetService defines the interface for budget management operations.
type BudgetService interface {
	CheckBudget(ctx context.Context, tenantID uuid.UUID) (BudgetStatus, error)
	GetTenantBudget(ctx context.Context, tenantID uuid.UUID) (*BudgetInfo, error)
}

// BudgetStatus represents the current budget status for a tenant.
type BudgetStatus struct {
	LimitUSD     decimal.Decimal
	UsedUSD      decimal.Decimal
	RemainingUSD decimal.Decimal
	IsWarning    bool
	IsExceeded   bool
}

// BudgetInfo holds budget configuration for a tenant.
type BudgetInfo struct {
	TenantID         uuid.UUID
	LimitUSD         decimal.Decimal
	WarningThreshold decimal.Decimal
	UsedUSD          decimal.Decimal
}

// RateLimitService defines the interface for rate limiting operations.
type RateLimitService interface {
	CheckRateLimit(ctx context.Context, tenantID uuid.UUID, apiKeyID uuid.UUID) (bool, error)
	GetTenantRateLimit(ctx context.Context, tenantID uuid.UUID) (*RateLimitInfo, error)
	GetAPIKeyRateLimit(ctx context.Context, apiKeyID uuid.UUID) (*RateLimitInfo, error)
}

// RateLimitInfo holds rate limit configuration.
type RateLimitInfo struct {
	RequestsPerMinute int32
	TokensPerMinute   int64
	RequestsPerDay    int32
	TokensPerDay      int64
}

// AuditService defines the interface for audit logging operations.
type AuditService interface {
	LogEvent(ctx context.Context, params AuditEventParams) error
	ListEvents(ctx context.Context, params ListAuditParams) ([]AuditEvent, error)
}

// AuditEventParams holds parameters for logging an audit event.
type AuditEventParams struct {
	UserID       pgtype.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]interface{}
}

// ListAuditParams holds parameters for listing audit events.
type ListAuditParams struct {
	UserID       *uuid.UUID
	Action       *string
	ResourceType *string
	Limit        int32
	Offset       int32
}

// AuditEvent represents an audit log entry.
type AuditEvent struct {
	ID           uuid.UUID
	UserID       pgtype.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]interface{}
	CreatedAt    time.Time
}

// ModelCatalogService defines the interface for model catalog operations.
type ModelCatalogService interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
	GetModel(ctx context.Context, alias string) (*ModelInfo, error)
	IsModelEnabled(ctx context.Context, alias string) (bool, error)
}

// ModelInfo represents a model in the catalog.
type ModelInfo struct {
	Alias           string
	Provider        string
	ProviderModel   string
	ModelType       string
	ContextWindow   int32
	MaxOutputTokens int32
	SupportsTools   bool
	PriceInput      decimal.Decimal
	PriceOutput     decimal.Decimal
	Enabled         bool
}

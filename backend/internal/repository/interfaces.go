package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// TenantRepository defines the interface for tenant data access.
type TenantRepository interface {
	GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error)
	GetTenantByName(ctx context.Context, name string) (db.Tenant, error)
	ListTenants(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error)
	ListPersonalTenants(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error)
	CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error)
	UpdateTenantName(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error)
	UpdateTenantStatus(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error)
}

// TenantModelRepository defines the interface for tenant model data access.
type TenantModelRepository interface {
	ListTenantModels(ctx context.Context, tenantID pgtype.UUID) ([]string, error)
	InsertTenantModel(ctx context.Context, params db.InsertTenantModelParams) error
	DeleteTenantModels(ctx context.Context, tenantID pgtype.UUID) error
}

// ApiKeyRepository defines the interface for API key data access.
type ApiKeyRepository interface {
	GetAPIKeyByPrefix(ctx context.Context, prefix string) (db.ApiKey, error)
	GetAPIKeyByID(ctx context.Context, id pgtype.UUID) (db.ApiKey, error)
	ListAPIKeysByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error)
	CreateAPIKey(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error)
	RevokeAPIKey(ctx context.Context, id pgtype.UUID) (db.ApiKey, error)
}

// UsageRepository defines the interface for usage data access.
type UsageRepository interface {
	InsertUsageRecord(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error)
	SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)
}

// BudgetRepository defines the interface for budget data access.
type BudgetRepository interface {
	GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error)
	ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error)
	UpsertTenantBudgetOverride(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error)
	DeleteTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) error
}

// RateLimitRepository defines the interface for rate limit data access.
type RateLimitRepository interface {
	GetAPIKeyRateLimit(ctx context.Context, apiKeyID pgtype.UUID) (db.GetAPIKeyRateLimitRow, error)
	UpsertAPIKeyRateLimit(ctx context.Context, params db.UpsertAPIKeyRateLimitParams) (db.UpsertAPIKeyRateLimitRow, error)
	DeleteAPIKeyRateLimit(ctx context.Context, apiKeyID pgtype.UUID) (int64, error)
	GetTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) (db.TenantRateLimit, error)
	UpsertTenantRateLimit(ctx context.Context, params db.UpsertTenantRateLimitParams) (db.TenantRateLimit, error)
	DeleteTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) error
}

// MembershipRepository defines the interface for membership data access.
type MembershipRepository interface {
	GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error)
	ListTenantMembers(ctx context.Context, tenantID pgtype.UUID) ([]db.ListTenantMembersRow, error)
	ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error)
	AddTenantMembership(ctx context.Context, params db.AddTenantMembershipParams) (db.TenantMembership, error)
	UpdateTenantMembershipRole(ctx context.Context, params db.UpdateTenantMembershipRoleParams) (db.TenantMembership, error)
	RemoveTenantMembership(ctx context.Context, params db.RemoveTenantMembershipParams) error
}

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)
}

// AuditRepository defines the interface for audit log data access.
type AuditRepository interface {
	InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error)
	ListAuditLogs(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error)
}

// ModelCatalogRepository defines the interface for model catalog data access.
type ModelCatalogRepository interface {
	ListModelCatalog(ctx context.Context) ([]db.ModelCatalog, error)
	GetModelByAlias(ctx context.Context, alias string) (db.ModelCatalog, error)
	UpsertModelCatalogEntry(ctx context.Context, params db.UpsertModelCatalogEntryParams) (db.ModelCatalog, error)
	DeleteModelCatalogEntry(ctx context.Context, alias string) error
}

// QueriesAdapter wraps *db.Queries and implements repository interfaces.
// This allows gradual migration - services can depend on specific repository
// interfaces while still using the concrete QueriesAdapter implementation.
type QueriesAdapter struct {
	Q *db.Queries
}

// NewQueriesAdapter creates a new QueriesAdapter wrapping the given Queries.
func NewQueriesAdapter(q *db.Queries) *QueriesAdapter {
	return &QueriesAdapter{Q: q}
}

// TenantRepository implementation

func (a *QueriesAdapter) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	return a.Q.GetTenantByID(ctx, id)
}

func (a *QueriesAdapter) GetTenantByName(ctx context.Context, name string) (db.Tenant, error) {
	return a.Q.GetTenantByName(ctx, name)
}

func (a *QueriesAdapter) ListTenants(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error) {
	return a.Q.ListTenants(ctx, params)
}

func (a *QueriesAdapter) CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error) {
	return a.Q.CreateTenant(ctx, params)
}

func (a *QueriesAdapter) UpdateTenantStatus(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error) {
	return a.Q.UpdateTenantStatus(ctx, params)
}

func (a *QueriesAdapter) ListPersonalTenants(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error) {
	return a.Q.ListPersonalTenants(ctx, params)
}

func (a *QueriesAdapter) UpdateTenantName(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error) {
	return a.Q.UpdateTenantName(ctx, params)
}

// TenantModelRepository implementation

func (a *QueriesAdapter) ListTenantModels(ctx context.Context, tenantID pgtype.UUID) ([]string, error) {
	return a.Q.ListTenantModels(ctx, tenantID)
}

func (a *QueriesAdapter) InsertTenantModel(ctx context.Context, params db.InsertTenantModelParams) error {
	return a.Q.InsertTenantModel(ctx, params)
}

func (a *QueriesAdapter) DeleteTenantModels(ctx context.Context, tenantID pgtype.UUID) error {
	return a.Q.DeleteTenantModels(ctx, tenantID)
}

// ApiKeyRepository implementation

func (a *QueriesAdapter) GetAPIKeyByPrefix(ctx context.Context, prefix string) (db.ApiKey, error) {
	return a.Q.GetAPIKeyByPrefix(ctx, prefix)
}

func (a *QueriesAdapter) GetAPIKeyByID(ctx context.Context, id pgtype.UUID) (db.ApiKey, error) {
	return a.Q.GetAPIKeyByID(ctx, id)
}

func (a *QueriesAdapter) ListAPIKeysByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error) {
	return a.Q.ListAPIKeysByTenant(ctx, tenantID)
}

func (a *QueriesAdapter) CreateAPIKey(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error) {
	return a.Q.CreateAPIKey(ctx, params)
}

func (a *QueriesAdapter) RevokeAPIKey(ctx context.Context, id pgtype.UUID) (db.ApiKey, error) {
	return a.Q.RevokeAPIKey(ctx, id)
}

// UsageRepository implementation

func (a *QueriesAdapter) InsertUsageRecord(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error) {
	return a.Q.InsertUsageRecord(ctx, params)
}

func (a *QueriesAdapter) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	return a.Q.SumUsageForTenant(ctx, params)
}

// BudgetRepository implementation

func (a *QueriesAdapter) GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error) {
	return a.Q.GetTenantBudgetOverride(ctx, tenantID)
}

func (a *QueriesAdapter) ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error) {
	return a.Q.ListTenantBudgetOverrides(ctx)
}

func (a *QueriesAdapter) UpsertTenantBudgetOverride(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error) {
	return a.Q.UpsertTenantBudgetOverride(ctx, params)
}

func (a *QueriesAdapter) DeleteTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) error {
	return a.Q.DeleteTenantBudgetOverride(ctx, tenantID)
}

// RateLimitRepository implementation

func (a *QueriesAdapter) GetAPIKeyRateLimit(ctx context.Context, apiKeyID pgtype.UUID) (db.GetAPIKeyRateLimitRow, error) {
	return a.Q.GetAPIKeyRateLimit(ctx, apiKeyID)
}

func (a *QueriesAdapter) UpsertAPIKeyRateLimit(ctx context.Context, params db.UpsertAPIKeyRateLimitParams) (db.UpsertAPIKeyRateLimitRow, error) {
	return a.Q.UpsertAPIKeyRateLimit(ctx, params)
}

func (a *QueriesAdapter) DeleteAPIKeyRateLimit(ctx context.Context, apiKeyID pgtype.UUID) (int64, error) {
	return a.Q.DeleteAPIKeyRateLimit(ctx, apiKeyID)
}

func (a *QueriesAdapter) GetTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) (db.TenantRateLimit, error) {
	return a.Q.GetTenantRateLimit(ctx, tenantID)
}

func (a *QueriesAdapter) UpsertTenantRateLimit(ctx context.Context, params db.UpsertTenantRateLimitParams) (db.TenantRateLimit, error) {
	return a.Q.UpsertTenantRateLimit(ctx, params)
}

func (a *QueriesAdapter) DeleteTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) error {
	return a.Q.DeleteTenantRateLimit(ctx, tenantID)
}

// MembershipRepository implementation

func (a *QueriesAdapter) GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error) {
	return a.Q.GetTenantMembership(ctx, params)
}

func (a *QueriesAdapter) ListTenantMembers(ctx context.Context, tenantID pgtype.UUID) ([]db.ListTenantMembersRow, error) {
	return a.Q.ListTenantMembers(ctx, tenantID)
}

func (a *QueriesAdapter) ListUserTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserTenantsRow, error) {
	return a.Q.ListUserTenants(ctx, userID)
}

func (a *QueriesAdapter) AddTenantMembership(ctx context.Context, params db.AddTenantMembershipParams) (db.TenantMembership, error) {
	return a.Q.AddTenantMembership(ctx, params)
}

func (a *QueriesAdapter) UpdateTenantMembershipRole(ctx context.Context, params db.UpdateTenantMembershipRoleParams) (db.TenantMembership, error) {
	return a.Q.UpdateTenantMembershipRole(ctx, params)
}

func (a *QueriesAdapter) RemoveTenantMembership(ctx context.Context, params db.RemoveTenantMembershipParams) error {
	return a.Q.RemoveTenantMembership(ctx, params)
}

// UserRepository implementation

func (a *QueriesAdapter) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return a.Q.GetUserByID(ctx, id)
}

func (a *QueriesAdapter) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return a.Q.GetUserByEmail(ctx, email)
}

func (a *QueriesAdapter) ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error) {
	return a.Q.ListUsers(ctx, params)
}

func (a *QueriesAdapter) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	return a.Q.CreateUser(ctx, params)
}

// AuditRepository implementation

func (a *QueriesAdapter) InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error) {
	return a.Q.InsertAuditLog(ctx, params)
}

func (a *QueriesAdapter) ListAuditLogs(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error) {
	return a.Q.ListAuditLogs(ctx, params)
}

// ModelCatalogRepository implementation

func (a *QueriesAdapter) ListModelCatalog(ctx context.Context) ([]db.ModelCatalog, error) {
	return a.Q.ListModelCatalog(ctx)
}

func (a *QueriesAdapter) GetModelByAlias(ctx context.Context, alias string) (db.ModelCatalog, error) {
	return a.Q.GetModelByAlias(ctx, alias)
}

func (a *QueriesAdapter) UpsertModelCatalogEntry(ctx context.Context, params db.UpsertModelCatalogEntryParams) (db.ModelCatalog, error) {
	return a.Q.UpsertModelCatalogEntry(ctx, params)
}

func (a *QueriesAdapter) DeleteModelCatalogEntry(ctx context.Context, alias string) error {
	return a.Q.DeleteModelCatalogEntry(ctx, alias)
}

// Compile-time checks that QueriesAdapter implements all interfaces
var (
	_ TenantRepository       = (*QueriesAdapter)(nil)
	_ TenantModelRepository  = (*QueriesAdapter)(nil)
	_ ApiKeyRepository       = (*QueriesAdapter)(nil)
	_ UsageRepository        = (*QueriesAdapter)(nil)
	_ BudgetRepository       = (*QueriesAdapter)(nil)
	_ RateLimitRepository    = (*QueriesAdapter)(nil)
	_ MembershipRepository   = (*QueriesAdapter)(nil)
	_ UserRepository         = (*QueriesAdapter)(nil)
	_ AuditRepository        = (*QueriesAdapter)(nil)
	_ ModelCatalogRepository = (*QueriesAdapter)(nil)
)

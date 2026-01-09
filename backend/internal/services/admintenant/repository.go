package admintenant

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin tenant operations.
// This interface aggregates all the database queries needed by the Service.
type Repository interface {
	// Tenant operations
	GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error)
	ListTenants(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error)
	ListPersonalTenants(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error)
	CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error)
	UpdateTenantName(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error)
	UpdateTenantStatus(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error)

	// Tenant model operations
	ListTenantModels(ctx context.Context, tenantID pgtype.UUID) ([]string, error)
	DeleteTenantModels(ctx context.Context, tenantID pgtype.UUID) error
	InsertTenantModel(ctx context.Context, params db.InsertTenantModelParams) error

	// Model catalog operations
	GetModelByAlias(ctx context.Context, alias string) (db.ModelCatalog, error)

	// Budget operations
	ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error)

	// API key operations
	CreateAPIKey(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error)
	ListAPIKeysByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error)
	RevokeAPIKey(ctx context.Context, id pgtype.UUID) (db.ApiKey, error)

	// Rate limit operations
	GetTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) (db.TenantRateLimit, error)
	UpsertTenantRateLimit(ctx context.Context, params db.UpsertTenantRateLimitParams) (db.TenantRateLimit, error)
	DeleteTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) error
	UpsertAPIKeyRateLimit(ctx context.Context, params db.UpsertAPIKeyRateLimitParams) (db.UpsertAPIKeyRateLimitRow, error)

	// Membership operations
	ListTenantMembers(ctx context.Context, tenantID pgtype.UUID) ([]db.ListTenantMembersRow, error)
	GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error)
	AddTenantMembership(ctx context.Context, params db.AddTenantMembershipParams) (db.TenantMembership, error)
	UpdateTenantMembershipRole(ctx context.Context, params db.UpdateTenantMembershipRoleParams) (db.TenantMembership, error)
	RemoveTenantMembership(ctx context.Context, params db.RemoveTenantMembershipParams) error

	// User operations
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)

	// Usage operations
	SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)

	// Transaction support - returns a new Repository scoped to the transaction
	WithTx(tx pgx.Tx) Repository
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

// Tenant operations

func (a *queriesAdapter) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	return a.q.GetTenantByID(ctx, id)
}

func (a *queriesAdapter) ListTenants(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error) {
	return a.q.ListTenants(ctx, params)
}

func (a *queriesAdapter) ListPersonalTenants(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error) {
	return a.q.ListPersonalTenants(ctx, params)
}

func (a *queriesAdapter) CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error) {
	return a.q.CreateTenant(ctx, params)
}

func (a *queriesAdapter) UpdateTenantName(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error) {
	return a.q.UpdateTenantName(ctx, params)
}

func (a *queriesAdapter) UpdateTenantStatus(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error) {
	return a.q.UpdateTenantStatus(ctx, params)
}

// Tenant model operations

func (a *queriesAdapter) ListTenantModels(ctx context.Context, tenantID pgtype.UUID) ([]string, error) {
	return a.q.ListTenantModels(ctx, tenantID)
}

func (a *queriesAdapter) DeleteTenantModels(ctx context.Context, tenantID pgtype.UUID) error {
	return a.q.DeleteTenantModels(ctx, tenantID)
}

func (a *queriesAdapter) InsertTenantModel(ctx context.Context, params db.InsertTenantModelParams) error {
	return a.q.InsertTenantModel(ctx, params)
}

// Model catalog operations

func (a *queriesAdapter) GetModelByAlias(ctx context.Context, alias string) (db.ModelCatalog, error) {
	return a.q.GetModelByAlias(ctx, alias)
}

// Budget operations

func (a *queriesAdapter) ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error) {
	return a.q.ListTenantBudgetOverrides(ctx)
}

// API key operations

func (a *queriesAdapter) CreateAPIKey(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error) {
	return a.q.CreateAPIKey(ctx, params)
}

func (a *queriesAdapter) ListAPIKeysByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error) {
	return a.q.ListAPIKeysByTenant(ctx, tenantID)
}

func (a *queriesAdapter) RevokeAPIKey(ctx context.Context, id pgtype.UUID) (db.ApiKey, error) {
	return a.q.RevokeAPIKey(ctx, id)
}

// Rate limit operations

func (a *queriesAdapter) GetTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) (db.TenantRateLimit, error) {
	return a.q.GetTenantRateLimit(ctx, tenantID)
}

func (a *queriesAdapter) UpsertTenantRateLimit(ctx context.Context, params db.UpsertTenantRateLimitParams) (db.TenantRateLimit, error) {
	return a.q.UpsertTenantRateLimit(ctx, params)
}

func (a *queriesAdapter) DeleteTenantRateLimit(ctx context.Context, tenantID pgtype.UUID) error {
	return a.q.DeleteTenantRateLimit(ctx, tenantID)
}

func (a *queriesAdapter) UpsertAPIKeyRateLimit(ctx context.Context, params db.UpsertAPIKeyRateLimitParams) (db.UpsertAPIKeyRateLimitRow, error) {
	return a.q.UpsertAPIKeyRateLimit(ctx, params)
}

// Membership operations

func (a *queriesAdapter) ListTenantMembers(ctx context.Context, tenantID pgtype.UUID) ([]db.ListTenantMembersRow, error) {
	return a.q.ListTenantMembers(ctx, tenantID)
}

func (a *queriesAdapter) GetTenantMembership(ctx context.Context, params db.GetTenantMembershipParams) (db.TenantMembership, error) {
	return a.q.GetTenantMembership(ctx, params)
}

func (a *queriesAdapter) AddTenantMembership(ctx context.Context, params db.AddTenantMembershipParams) (db.TenantMembership, error) {
	return a.q.AddTenantMembership(ctx, params)
}

func (a *queriesAdapter) UpdateTenantMembershipRole(ctx context.Context, params db.UpdateTenantMembershipRoleParams) (db.TenantMembership, error) {
	return a.q.UpdateTenantMembershipRole(ctx, params)
}

func (a *queriesAdapter) RemoveTenantMembership(ctx context.Context, params db.RemoveTenantMembershipParams) error {
	return a.q.RemoveTenantMembership(ctx, params)
}

// User operations

func (a *queriesAdapter) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return a.q.GetUserByEmail(ctx, email)
}

func (a *queriesAdapter) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	return a.q.CreateUser(ctx, params)
}

// Usage operations

func (a *queriesAdapter) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	return a.q.SumUsageForTenant(ctx, params)
}

// Transaction support

func (a *queriesAdapter) WithTx(tx pgx.Tx) Repository {
	return &queriesAdapter{q: a.q.WithTx(tx)}
}

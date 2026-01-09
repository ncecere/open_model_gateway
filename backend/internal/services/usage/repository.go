package usage

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for usage operations.
// This interface aggregates all the database queries needed by the Service.
type Repository interface {
	// Aggregation operations - daily
	AggregateTenantUsageDaily(ctx context.Context, params db.AggregateTenantUsageDailyParams) ([]db.AggregateTenantUsageDailyRow, error)
	AggregateTenantUsageDailyByAPIKeys(ctx context.Context, params db.AggregateTenantUsageDailyByAPIKeysParams) ([]db.AggregateTenantUsageDailyByAPIKeysRow, error)
	AggregateUsageDailyByUsers(ctx context.Context, params db.AggregateUsageDailyByUsersParams) ([]db.AggregateUsageDailyByUsersRow, error)
	AggregateUserUsageDailyByTenants(ctx context.Context, params db.AggregateUserUsageDailyByTenantsParams) ([]db.AggregateUserUsageDailyByTenantsRow, error)
	AggregateUsageDailyByModels(ctx context.Context, params db.AggregateUsageDailyByModelsParams) ([]db.AggregateUsageDailyByModelsRow, error)
	AggregateModelUsageDailyByTenants(ctx context.Context, params db.AggregateModelUsageDailyByTenantsParams) ([]db.AggregateModelUsageDailyByTenantsRow, error)
	AggregateAPIKeyUsageDaily(ctx context.Context, params db.AggregateAPIKeyUsageDailyParams) ([]db.AggregateAPIKeyUsageDailyRow, error)
	AggregateUsageDailyForUserTenant(ctx context.Context, params db.AggregateUsageDailyForUserTenantParams) ([]db.AggregateUsageDailyForUserTenantRow, error)
	AggregateUsageDaily(ctx context.Context, params db.AggregateUsageDailyParams) ([]db.AggregateUsageDailyRow, error)
	AggregateModelUsageDaily(ctx context.Context, params db.AggregateModelUsageDailyParams) ([]db.AggregateModelUsageDailyRow, error)
	AggregateUsageDailyByTenants(ctx context.Context, params db.AggregateUsageDailyByTenantsParams) ([]db.AggregateUsageDailyByTenantsRow, error)

	// Aggregation operations - by entity
	AggregateUsageByTenant(ctx context.Context, params db.AggregateUsageByTenantParams) ([]db.AggregateUsageByTenantRow, error)
	AggregateUsageByModel(ctx context.Context, params db.AggregateUsageByModelParams) ([]db.AggregateUsageByModelRow, error)
	AggregateUsageByUser(ctx context.Context, params db.AggregateUsageByUserParams) ([]db.AggregateUsageByUserRow, error)

	// Sum operations
	SumUsage(ctx context.Context, params db.SumUsageParams) (db.SumUsageRow, error)
	SumUsageForAPIKey(ctx context.Context, params db.SumUsageForAPIKeyParams) (db.SumUsageForAPIKeyRow, error)
	SumUsageForUserTenant(ctx context.Context, params db.SumUsageForUserTenantParams) (db.SumUsageForUserTenantRow, error)
	SumUsageByTenants(ctx context.Context, params db.SumUsageByTenantsParams) ([]db.SumUsageByTenantsRow, error)
	SumUsageByModels(ctx context.Context, params db.SumUsageByModelsParams) ([]db.SumUsageByModelsRow, error)
	SumUsageByUsers(ctx context.Context, params db.SumUsageByUsersParams) ([]db.SumUsageByUsersRow, error)

	// Request metrics operations
	AggregateRequestMetricsByModel(ctx context.Context, params db.AggregateRequestMetricsByModelParams) ([]db.AggregateRequestMetricsByModelRow, error)
	AggregateLatencyByModel(ctx context.Context, params db.AggregateLatencyByModelParams) ([]db.AggregateLatencyByModelRow, error)

	// List operations
	ListUserOwnedTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserOwnedTenantsRow, error)
	ListAPIKeysByOwnerAndTenant(ctx context.Context, params db.ListAPIKeysByOwnerAndTenantParams) ([]db.ApiKey, error)
	ListRecentRequestsByAPIKeys(ctx context.Context, params db.ListRecentRequestsByAPIKeysParams) ([]db.Request, error)
	ListTenantsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.ListTenantsByIDsRow, error)
	ListModelCatalogByAliases(ctx context.Context, aliases []string) ([]db.ModelCatalog, error)
	ListUsersByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.ListUsersByIDsRow, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

// Aggregation operations - daily

func (a *queriesAdapter) AggregateTenantUsageDaily(ctx context.Context, params db.AggregateTenantUsageDailyParams) ([]db.AggregateTenantUsageDailyRow, error) {
	return a.q.AggregateTenantUsageDaily(ctx, params)
}

func (a *queriesAdapter) AggregateTenantUsageDailyByAPIKeys(ctx context.Context, params db.AggregateTenantUsageDailyByAPIKeysParams) ([]db.AggregateTenantUsageDailyByAPIKeysRow, error) {
	return a.q.AggregateTenantUsageDailyByAPIKeys(ctx, params)
}

func (a *queriesAdapter) AggregateUsageDailyByUsers(ctx context.Context, params db.AggregateUsageDailyByUsersParams) ([]db.AggregateUsageDailyByUsersRow, error) {
	return a.q.AggregateUsageDailyByUsers(ctx, params)
}

func (a *queriesAdapter) AggregateUserUsageDailyByTenants(ctx context.Context, params db.AggregateUserUsageDailyByTenantsParams) ([]db.AggregateUserUsageDailyByTenantsRow, error) {
	return a.q.AggregateUserUsageDailyByTenants(ctx, params)
}

func (a *queriesAdapter) AggregateUsageDailyByModels(ctx context.Context, params db.AggregateUsageDailyByModelsParams) ([]db.AggregateUsageDailyByModelsRow, error) {
	return a.q.AggregateUsageDailyByModels(ctx, params)
}

func (a *queriesAdapter) AggregateModelUsageDailyByTenants(ctx context.Context, params db.AggregateModelUsageDailyByTenantsParams) ([]db.AggregateModelUsageDailyByTenantsRow, error) {
	return a.q.AggregateModelUsageDailyByTenants(ctx, params)
}

func (a *queriesAdapter) AggregateAPIKeyUsageDaily(ctx context.Context, params db.AggregateAPIKeyUsageDailyParams) ([]db.AggregateAPIKeyUsageDailyRow, error) {
	return a.q.AggregateAPIKeyUsageDaily(ctx, params)
}

func (a *queriesAdapter) AggregateUsageDailyForUserTenant(ctx context.Context, params db.AggregateUsageDailyForUserTenantParams) ([]db.AggregateUsageDailyForUserTenantRow, error) {
	return a.q.AggregateUsageDailyForUserTenant(ctx, params)
}

func (a *queriesAdapter) AggregateUsageDaily(ctx context.Context, params db.AggregateUsageDailyParams) ([]db.AggregateUsageDailyRow, error) {
	return a.q.AggregateUsageDaily(ctx, params)
}

func (a *queriesAdapter) AggregateModelUsageDaily(ctx context.Context, params db.AggregateModelUsageDailyParams) ([]db.AggregateModelUsageDailyRow, error) {
	return a.q.AggregateModelUsageDaily(ctx, params)
}

func (a *queriesAdapter) AggregateUsageDailyByTenants(ctx context.Context, params db.AggregateUsageDailyByTenantsParams) ([]db.AggregateUsageDailyByTenantsRow, error) {
	return a.q.AggregateUsageDailyByTenants(ctx, params)
}

// Aggregation operations - by entity

func (a *queriesAdapter) AggregateUsageByTenant(ctx context.Context, params db.AggregateUsageByTenantParams) ([]db.AggregateUsageByTenantRow, error) {
	return a.q.AggregateUsageByTenant(ctx, params)
}

func (a *queriesAdapter) AggregateUsageByModel(ctx context.Context, params db.AggregateUsageByModelParams) ([]db.AggregateUsageByModelRow, error) {
	return a.q.AggregateUsageByModel(ctx, params)
}

func (a *queriesAdapter) AggregateUsageByUser(ctx context.Context, params db.AggregateUsageByUserParams) ([]db.AggregateUsageByUserRow, error) {
	return a.q.AggregateUsageByUser(ctx, params)
}

// Sum operations

func (a *queriesAdapter) SumUsage(ctx context.Context, params db.SumUsageParams) (db.SumUsageRow, error) {
	return a.q.SumUsage(ctx, params)
}

func (a *queriesAdapter) SumUsageForAPIKey(ctx context.Context, params db.SumUsageForAPIKeyParams) (db.SumUsageForAPIKeyRow, error) {
	return a.q.SumUsageForAPIKey(ctx, params)
}

func (a *queriesAdapter) SumUsageForUserTenant(ctx context.Context, params db.SumUsageForUserTenantParams) (db.SumUsageForUserTenantRow, error) {
	return a.q.SumUsageForUserTenant(ctx, params)
}

func (a *queriesAdapter) SumUsageByTenants(ctx context.Context, params db.SumUsageByTenantsParams) ([]db.SumUsageByTenantsRow, error) {
	return a.q.SumUsageByTenants(ctx, params)
}

func (a *queriesAdapter) SumUsageByModels(ctx context.Context, params db.SumUsageByModelsParams) ([]db.SumUsageByModelsRow, error) {
	return a.q.SumUsageByModels(ctx, params)
}

func (a *queriesAdapter) SumUsageByUsers(ctx context.Context, params db.SumUsageByUsersParams) ([]db.SumUsageByUsersRow, error) {
	return a.q.SumUsageByUsers(ctx, params)
}

// Request metrics operations

func (a *queriesAdapter) AggregateRequestMetricsByModel(ctx context.Context, params db.AggregateRequestMetricsByModelParams) ([]db.AggregateRequestMetricsByModelRow, error) {
	return a.q.AggregateRequestMetricsByModel(ctx, params)
}

func (a *queriesAdapter) AggregateLatencyByModel(ctx context.Context, params db.AggregateLatencyByModelParams) ([]db.AggregateLatencyByModelRow, error) {
	return a.q.AggregateLatencyByModel(ctx, params)
}

// List operations

func (a *queriesAdapter) ListUserOwnedTenants(ctx context.Context, userID pgtype.UUID) ([]db.ListUserOwnedTenantsRow, error) {
	return a.q.ListUserOwnedTenants(ctx, userID)
}

func (a *queriesAdapter) ListAPIKeysByOwnerAndTenant(ctx context.Context, params db.ListAPIKeysByOwnerAndTenantParams) ([]db.ApiKey, error) {
	return a.q.ListAPIKeysByOwnerAndTenant(ctx, params)
}

func (a *queriesAdapter) ListRecentRequestsByAPIKeys(ctx context.Context, params db.ListRecentRequestsByAPIKeysParams) ([]db.Request, error) {
	return a.q.ListRecentRequestsByAPIKeys(ctx, params)
}

func (a *queriesAdapter) ListTenantsByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.ListTenantsByIDsRow, error) {
	return a.q.ListTenantsByIDs(ctx, ids)
}

func (a *queriesAdapter) ListModelCatalogByAliases(ctx context.Context, aliases []string) ([]db.ModelCatalog, error) {
	return a.q.ListModelCatalogByAliases(ctx, aliases)
}

func (a *queriesAdapter) ListUsersByIDs(ctx context.Context, ids []pgtype.UUID) ([]db.ListUsersByIDsRow, error) {
	return a.q.ListUsersByIDs(ctx, ids)
}

package billinghooks

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for billing webhook operations.
type Repository interface {
	// Webhook CRUD
	CreateBillingWebhook(ctx context.Context, params db.CreateBillingWebhookParams) (db.BillingWebhook, error)
	UpdateBillingWebhook(ctx context.Context, params db.UpdateBillingWebhookParams) (db.BillingWebhook, error)
	DeleteBillingWebhook(ctx context.Context, id pgtype.UUID) error
	GetBillingWebhook(ctx context.Context, id pgtype.UUID) (db.BillingWebhook, error)
	ListBillingWebhooksAdmin(ctx context.Context) ([]db.BillingWebhook, error)
	ListBillingWebhooksByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.BillingWebhook, error)

	// Event operations
	GetBillingWebhookEvent(ctx context.Context, id pgtype.UUID) (db.BillingWebhookEvent, error)
	ListBillingWebhookEvents(ctx context.Context, params db.ListBillingWebhookEventsParams) ([]db.BillingWebhookEvent, error)
	InsertBillingWebhookEvent(ctx context.Context, params db.InsertBillingWebhookEventParams) (db.BillingWebhookEvent, error)

	// Usage aggregation (for building summaries)
	SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)
	AggregateUsageByModelForTenant(ctx context.Context, params db.AggregateUsageByModelForTenantParams) ([]db.AggregateUsageByModelForTenantRow, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) CreateBillingWebhook(ctx context.Context, params db.CreateBillingWebhookParams) (db.BillingWebhook, error) {
	return a.q.CreateBillingWebhook(ctx, params)
}

func (a *queriesAdapter) UpdateBillingWebhook(ctx context.Context, params db.UpdateBillingWebhookParams) (db.BillingWebhook, error) {
	return a.q.UpdateBillingWebhook(ctx, params)
}

func (a *queriesAdapter) DeleteBillingWebhook(ctx context.Context, id pgtype.UUID) error {
	return a.q.DeleteBillingWebhook(ctx, id)
}

func (a *queriesAdapter) GetBillingWebhook(ctx context.Context, id pgtype.UUID) (db.BillingWebhook, error) {
	return a.q.GetBillingWebhook(ctx, id)
}

func (a *queriesAdapter) ListBillingWebhooksAdmin(ctx context.Context) ([]db.BillingWebhook, error) {
	return a.q.ListBillingWebhooksAdmin(ctx)
}

func (a *queriesAdapter) ListBillingWebhooksByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.BillingWebhook, error) {
	return a.q.ListBillingWebhooksByTenant(ctx, tenantID)
}

func (a *queriesAdapter) GetBillingWebhookEvent(ctx context.Context, id pgtype.UUID) (db.BillingWebhookEvent, error) {
	return a.q.GetBillingWebhookEvent(ctx, id)
}

func (a *queriesAdapter) ListBillingWebhookEvents(ctx context.Context, params db.ListBillingWebhookEventsParams) ([]db.BillingWebhookEvent, error) {
	return a.q.ListBillingWebhookEvents(ctx, params)
}

func (a *queriesAdapter) InsertBillingWebhookEvent(ctx context.Context, params db.InsertBillingWebhookEventParams) (db.BillingWebhookEvent, error) {
	return a.q.InsertBillingWebhookEvent(ctx, params)
}

func (a *queriesAdapter) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	return a.q.SumUsageForTenant(ctx, params)
}

func (a *queriesAdapter) AggregateUsageByModelForTenant(ctx context.Context, params db.AggregateUsageByModelForTenantParams) ([]db.AggregateUsageByModelForTenantRow, error) {
	return a.q.AggregateUsageByModelForTenant(ctx, params)
}

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Fixtures provides methods for creating test data.
type Fixtures struct {
	t       *testing.T
	queries *db.Queries
}

// NewFixtures creates a new Fixtures helper.
func NewFixtures(t *testing.T, queries *db.Queries) *Fixtures {
	return &Fixtures{t: t, queries: queries}
}

// CreateTenant creates a test tenant.
func (f *Fixtures) CreateTenant(ctx context.Context, name string) db.Tenant {
	f.t.Helper()

	tenant, err := f.queries.CreateTenant(ctx, db.CreateTenantParams{
		Name:   name,
		Status: db.TenantStatusActive,
		Kind:   db.TenantKindOrganization,
	})
	if err != nil {
		f.t.Fatalf("create test tenant: %v", err)
	}
	return tenant
}

// CreateAPIKey creates a test API key for a tenant.
func (f *Fixtures) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, name string) db.ApiKey {
	f.t.Helper()

	// Generate unique prefix to avoid conflicts
	prefix := "omg_" + uuid.New().String()[:8]

	key, err := f.queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		Prefix:     prefix,
		SecretHash: "test-hash-" + uuid.New().String(),
		Name:       name,
		ScopesJson: []byte("[]"),
		QuotaJson:  []byte("{}"),
		Kind:       db.ApiKeyKindService,
	})
	if err != nil {
		f.t.Fatalf("create test api key: %v", err)
	}
	return key
}

// CreateUser creates a test user.
func (f *Fixtures) CreateUser(ctx context.Context, email, name string) db.User {
	f.t.Helper()

	user, err := f.queries.CreateUser(ctx, db.CreateUserParams{
		Email: email,
		Name:  name,
	})
	if err != nil {
		f.t.Fatalf("create test user: %v", err)
	}
	return user
}

// CreateMembership creates a tenant membership for a user.
func (f *Fixtures) CreateMembership(ctx context.Context, tenantID, userID uuid.UUID, role db.MembershipRole) db.TenantMembership {
	f.t.Helper()

	membership, err := f.queries.AddTenantMembership(ctx, db.AddTenantMembershipParams{
		TenantID: pgtype.UUID{Bytes: tenantID, Valid: true},
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
		Role:     role,
	})
	if err != nil {
		f.t.Fatalf("create test membership: %v", err)
	}
	return membership
}

// CreateUsageRecord creates a test usage record.
func (f *Fixtures) CreateUsageRecord(ctx context.Context, tenantID uuid.UUID, alias string, tokens int64) db.UsageRecord {
	f.t.Helper()

	now := time.Now().UTC()
	record, err := f.queries.InsertUsageRecord(ctx, db.InsertUsageRecordParams{
		TenantID:      pgtype.UUID{Bytes: tenantID, Valid: true},
		Ts:            pgtype.Timestamptz{Time: now, Valid: true},
		ModelAlias:    alias,
		Provider:      "openai",
		InputTokens:   tokens / 2,
		OutputTokens:  tokens / 2,
		Requests:      1,
		CostCents:     int64(float64(tokens) * 0.001),
		CostUsdMicros: int64(float64(tokens) * 10),
	})
	if err != nil {
		f.t.Fatalf("create test usage record: %v", err)
	}
	return record
}

// TenantFixture holds a complete tenant test setup.
type TenantFixture struct {
	Tenant db.Tenant
	User   db.User
	APIKey db.ApiKey
}

// CreateTenantFixture creates a complete tenant setup with user and API key.
func (f *Fixtures) CreateTenantFixture(ctx context.Context, tenantName, userEmail string) TenantFixture {
	f.t.Helper()

	tenant := f.CreateTenant(ctx, tenantName)
	tenantID, _ := uuid.FromBytes(tenant.ID.Bytes[:])

	user := f.CreateUser(ctx, userEmail, "Test User")
	userID, _ := uuid.FromBytes(user.ID.Bytes[:])

	f.CreateMembership(ctx, tenantID, userID, db.MembershipRoleAdmin)
	apiKey := f.CreateAPIKey(ctx, tenantID, "test-key")

	return TenantFixture{
		Tenant: tenant,
		User:   user,
		APIKey: apiKey,
	}
}

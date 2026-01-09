package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

func TestStartTestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	cleanup, dsn := StartTestPostgres(t)
	defer cleanup()

	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test postgres: %v", err)
	}
}

func TestStartTestRedis(t *testing.T) {
	t.Parallel()

	cleanup, url := StartTestRedis(t)
	defer cleanup()

	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestStartMiniRedis(t *testing.T) {
	t.Parallel()

	mr := StartMiniRedis(t)

	if mr.URL() == "" {
		t.Fatal("expected non-empty URL")
	}

	// Test basic operations
	if err := mr.Set("test-key", "test-value"); err != nil {
		t.Fatalf("set key: %v", err)
	}

	val, err := mr.Get("test-key")
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if val != "test-value" {
		t.Fatalf("expected 'test-value', got '%s'", val)
	}
}

func TestApplySchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	cleanup, dsn := StartTestPostgres(t)
	defer cleanup()

	if err := ApplySchema(ctx, dsn); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	// Verify schema was applied by checking for tenants table
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'tenants'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("check table: %v", err)
	}
	if !exists {
		t.Fatal("expected tenants table to exist")
	}
}

func TestNewTestConfig(t *testing.T) {
	t.Parallel()

	cfg := NewTestConfig(TestConfigOptions{
		PostgresURL: "postgres://localhost/test",
		RedisURL:    "redis://localhost",
		FilesDir:    "/tmp/files",
	})

	if cfg.Database.URL != "postgres://localhost/test" {
		t.Fatalf("expected postgres URL, got %s", cfg.Database.URL)
	}
	if cfg.Redis.URL != "redis://localhost" {
		t.Fatalf("expected redis URL, got %s", cfg.Redis.URL)
	}
	if cfg.Files.Local.Directory != "/tmp/files" {
		t.Fatalf("expected files dir, got %s", cfg.Files.Local.Directory)
	}
	if len(cfg.ModelCatalog) == 0 {
		t.Fatal("expected model catalog entries")
	}
}

func TestFixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ctx := context.Background()

	cleanup, dsn := StartTestPostgres(t)
	defer cleanup()

	if err := ApplySchema(ctx, dsn); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	fixtures := NewFixtures(t, queries)

	// Test CreateTenant
	tenant := fixtures.CreateTenant(ctx, "Test Tenant")
	if tenant.Name != "Test Tenant" {
		t.Fatalf("expected tenant name 'Test Tenant', got '%s'", tenant.Name)
	}

	tenantID, _ := uuid.FromBytes(tenant.ID.Bytes[:])

	// Test CreateUser
	user := fixtures.CreateUser(ctx, "test@example.com", "Test User")
	if user.Email != "test@example.com" {
		t.Fatalf("expected email 'test@example.com', got '%s'", user.Email)
	}

	userID, _ := uuid.FromBytes(user.ID.Bytes[:])

	// Test CreateMembership
	membership := fixtures.CreateMembership(ctx, tenantID, userID, db.MembershipRoleAdmin)
	if membership.Role != db.MembershipRoleAdmin {
		t.Fatalf("expected role 'admin', got '%s'", membership.Role)
	}

	// Test CreateAPIKey
	apiKey := fixtures.CreateAPIKey(ctx, tenantID, "Test Key")
	if apiKey.Name != "Test Key" {
		t.Fatalf("expected key name 'Test Key', got '%s'", apiKey.Name)
	}

	// Test CreateTenantFixture
	fixture := fixtures.CreateTenantFixture(ctx, "Fixture Tenant", "fixture@example.com")
	if fixture.Tenant.Name != "Fixture Tenant" {
		t.Fatalf("expected tenant name, got '%s'", fixture.Tenant.Name)
	}
	if fixture.User.Email != "fixture@example.com" {
		t.Fatalf("expected user email, got '%s'", fixture.User.Email)
	}
}

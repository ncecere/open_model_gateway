package user

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
)

type permissionStubRow struct {
	values []any
	err    error
}

func (r permissionStubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan mismatch: %d dest vs %d values", len(dest), len(r.values))
	}
	for i, val := range r.values {
		switch d := dest[i].(type) {
		case *pgtype.UUID:
			*d = val.(pgtype.UUID)
		case *db.MembershipRole:
			*d = val.(db.MembershipRole)
		case *decimal.Decimal:
			*d = val.(decimal.Decimal)
		case *int64:
			*d = val.(int64)
		case *string:
			*d = val.(string)
		case *pgtype.Timestamptz:
			*d = val.(pgtype.Timestamptz)
		case *db.TenantStatus:
			*d = val.(db.TenantStatus)
		case *db.TenantKind:
			*d = val.(db.TenantKind)
		case *interface{}:
			*d = val
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

type permissionStubDB struct {
	tenantID      uuid.UUID
	userID        uuid.UUID
	role          db.MembershipRole
	hasMembership bool
	now           time.Time
}

func (s permissionStubDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, fmt.Errorf("unexpected Exec call")
}

func (s permissionStubDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, fmt.Errorf("unexpected Query call")
}

func (s permissionStubDB) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM tenant_memberships"):
		if !s.hasMembership {
			return permissionStubRow{err: pgx.ErrNoRows}
		}
		return permissionStubRow{values: []any{
			pgtype.UUID{Bytes: uuid.New(), Valid: true},
			pgtype.UUID{Bytes: s.tenantID, Valid: true},
			pgtype.UUID{Bytes: s.userID, Valid: true},
			s.role,
			decimal.NewFromInt(0),
			decimal.NewFromInt(0),
			int64(0),
			pgtype.Timestamptz{Time: s.now, Valid: true},
		}}
	case strings.Contains(sql, "FROM tenants"):
		return permissionStubRow{values: []any{
			pgtype.UUID{Bytes: s.tenantID, Valid: true},
			"Test Tenant",
			db.TenantStatusActive,
			db.TenantKindOrganization,
			pgtype.Timestamptz{Time: s.now, Valid: true},
		}}
	case strings.Contains(sql, "FROM tenant_budget_overrides"):
		return permissionStubRow{err: pgx.ErrNoRows}
	case strings.Contains(sql, "FROM usage_records"):
		return permissionStubRow{values: []any{
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			int64(0),
		}}
	default:
		return permissionStubRow{err: fmt.Errorf("unexpected query")}
	}
}

func TestRequireTenantCapabilityBoundary(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	userRecord := db.User{
		ID:    toPgUUID(userID),
		Email: "tester@example.com",
	}

	cases := []rbac.Capability{
		rbac.CapabilityManageMemberships,
		rbac.CapabilityManageTenantKeys,
		rbac.CapabilityManageBillingWebhooks,
		rbac.CapabilityManageBatches,
		rbac.CapabilityManageMemberBudgets,
		rbac.CapabilityManageTenantLimits,
		rbac.CapabilityAttachModels,
		rbac.CapabilityManageTenantGuardrails,
	}

	for _, capability := range cases {
		capability := capability
		t.Run(string(capability)+"/viewer", func(t *testing.T) {
			status := exerciseCapabilityBoundary(t, tenantID, userID, userRecord, capability, db.MembershipRoleViewer)
			if status != fiber.StatusForbidden {
				t.Fatalf("expected 403 for %s viewer, got %d", capability, status)
			}
		})
		t.Run(string(capability)+"/admin", func(t *testing.T) {
			status := exerciseCapabilityBoundary(t, tenantID, userID, userRecord, capability, db.MembershipRoleAdmin)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 for %s admin, got %d", capability, status)
			}
		})
	}
}

func exerciseCapabilityBoundary(
	t *testing.T,
	tenantID uuid.UUID,
	userID uuid.UUID,
	userRecord db.User,
	capability rbac.Capability,
	role db.MembershipRole,
) int {
	t.Helper()

	cfg := &config.Config{
		Budgets: config.BudgetConfig{
			DefaultUSD:           100,
			WarningThresholdPerc: 0.8,
			RefreshSchedule:      "calendar_month",
		},
	}
	queries := db.New(permissionStubDB{
		tenantID:      tenantID,
		userID:        userID,
		role:          role,
		hasMembership: true,
		now:           time.Now(),
	})
	handler := &userHandler{
		tenantSvc: tenantservice.NewService(cfg, queries, time.UTC),
	}

	app := fiber.New()
	app.Get("/tenants/:tenantID/test",
		func(c *fiber.Ctx) error {
			attachUserContext(c, userRecord, userID)
			return c.Next()
		},
		handler.requireTenantCapability("tenantID", capability),
		func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/tenants/"+tenantID.String()+"/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return resp.StatusCode
}

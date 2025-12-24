package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

type stubRow struct {
	values []any
	err    error
}

func (r stubRow) Scan(dest ...any) error {
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
		case *int32:
			*d = val.(int32)
		case *string:
			*d = val.(string)
		case *[]string:
			*d = val.([]string)
		case *pgtype.Timestamptz:
			*d = val.(pgtype.Timestamptz)
		case *pgtype.Text:
			*d = val.(pgtype.Text)
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

type stubDB struct {
	membership db.TenantMembership
	hasMember  bool
}

func (s stubDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, fmt.Errorf("unexpected Exec call")
}

func (s stubDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, fmt.Errorf("unexpected Query call")
}

func (s stubDB) QueryRow(_ context.Context, sql string, _ ...interface{}) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM tenant_budget_overrides"):
		return stubRow{err: pgx.ErrNoRows}
	case strings.Contains(sql, "FROM tenant_memberships"):
		if !s.hasMember {
			return stubRow{err: pgx.ErrNoRows}
		}
		return stubRow{values: []any{
			s.membership.ID,
			s.membership.TenantID,
			s.membership.UserID,
			s.membership.Role,
			s.membership.BudgetUsd,
			s.membership.WarningThreshold,
			s.membership.TokenCap,
			s.membership.CreatedAt,
		}}
	default:
		return stubRow{err: fmt.Errorf("unexpected query")}
	}
}

func TestBuildRequestContextCapsBudgetByMember(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	keyID := uuid.New()

	membership := db.TenantMembership{
		ID:               toPgUUID(uuid.New()),
		TenantID:         toPgUUID(tenantID),
		UserID:           toPgUUID(userID),
		Role:             db.MembershipRoleAdmin,
		BudgetUsd:        decimal.NewFromFloat(25.0),
		WarningThreshold: decimal.NewFromFloat(0.7),
		TokenCap:         0,
		CreatedAt:        pgtype.Timestamptz{},
	}

	queries := db.New(stubDB{membership: membership, hasMember: true})
	cfg := &config.Config{
		Budgets: config.BudgetConfig{
			DefaultUSD:           100,
			WarningThresholdPerc: 0.8,
			RefreshSchedule:      "calendar_month",
		},
	}

	container := &Container{
		Config:  cfg,
		Queries: queries,
	}

	record := db.ApiKey{
		ID:          toPgUUID(keyID),
		TenantID:    toPgUUID(tenantID),
		OwnerUserID: toPgUUID(userID),
		QuotaJson:   []byte(`{"budget_usd":50}`),
		Prefix:      "test",
	}

	rc, err := BuildRequestContext(context.Background(), container, record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rc.BudgetLimitCents != 2500 {
		t.Fatalf("expected budget limit 2500, got %d", rc.BudgetLimitCents)
	}
	if rc.WarningThreshold != 0.7 {
		t.Fatalf("expected warning threshold 0.7, got %v", rc.WarningThreshold)
	}
}

package tenant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Service centralizes tenant-related read operations consumed by HTTP handlers.
type Service struct {
	queries  *db.Queries
	cfg      *config.Config
	timezone *time.Location
}

func NewService(cfg *config.Config, queries *db.Queries, timezone *time.Location) *Service {
	if timezone == nil {
		timezone = time.UTC
	}
	return &Service{cfg: cfg, queries: queries, timezone: timezone}
}

// Membership represents a tenant membership for a specific user.
type Membership struct {
	TenantID   uuid.UUID
	Name       string
	Status     db.TenantStatus
	Role       db.MembershipRole
	JoinedAt   time.Time
	CreatedAt  time.Time
	IsPersonal bool
}

// Summary captures tenant metadata and budget context for a member.
type Summary struct {
	TenantID  uuid.UUID
	Name      string
	Status    db.TenantStatus
	Role      db.MembershipRole
	CreatedAt time.Time
	Budget    BudgetSummary
}

// BudgetSummary exposes limit/usage metadata for a tenant.
type BudgetSummary struct {
	LimitUSD         float64
	UsedUSD          float64
	RemainingUSD     float64
	WarningThreshold float64
	RefreshSchedule  string
}

// ListUserMemberships returns all tenants the user belongs to (including personal tenant).
func (s *Service) ListUserMemberships(ctx context.Context, user db.User) ([]Membership, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("tenant service not initialized")
	}
	rows, err := s.queries.ListUserTenants(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Membership, 0, len(rows))
	var personal uuid.UUID
	if user.PersonalTenantID.Valid {
		personal, _ = uuidFromPg(user.PersonalTenantID)
	}
	for _, row := range rows {
		tenantID, err := uuidFromPg(row.TenantID)
		if err != nil {
			continue
		}
		joined, err := timeFromPg(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		created, err := timeFromPg(row.TenantCreatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, Membership{
			TenantID:   tenantID,
			Name:       row.TenantName,
			Status:     row.TenantStatus,
			Role:       row.Role,
			JoinedAt:   joined.In(s.timezone),
			CreatedAt:  created.In(s.timezone),
			IsPersonal: personal != uuid.Nil && tenantID == personal,
		})
	}
	return out, nil
}

// GetTenantSummary returns tenant metadata/budget info for the provided tenant if the user is a member.
func (s *Service) GetTenantSummary(ctx context.Context, user db.User, tenantID uuid.UUID) (Summary, error) {
	if s == nil || s.queries == nil {
		return Summary{}, errors.New("tenant service not initialized")
	}
	membership, err := s.queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
		TenantID: toPgUUID(tenantID),
		UserID:   user.ID,
	})
	if err != nil {
		return Summary{}, err
	}
	tenant, err := s.queries.GetTenantByID(ctx, toPgUUID(tenantID))
	if err != nil {
		return Summary{}, err
	}
	created, err := timeFromPg(tenant.CreatedAt)
	if err != nil {
		return Summary{}, err
	}
	budget, err := s.buildBudgetSummary(ctx, tenantID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		TenantID:  tenantID,
		Name:      tenant.Name,
		Status:    tenant.Status,
		Role:      membership.Role,
		CreatedAt: created.In(s.timezone),
		Budget:    budget,
	}, nil
}

func (s *Service) buildBudgetSummary(ctx context.Context, tenantID uuid.UUID) (BudgetSummary, error) {
	limit := s.cfg.Budgets.DefaultUSD
	warn := s.cfg.Budgets.WarningThresholdPerc
	schedule := s.cfg.Budgets.RefreshSchedule
	override, err := s.queries.GetTenantBudgetOverride(ctx, toPgUUID(tenantID))
	if err == nil {
		if val, ok := override.BudgetUsd.Float64(); ok && val > 0 {
			limit = val
		}
		if val, ok := override.WarningThreshold.Float64(); ok && val > 0 {
			warn = val
		}
		if sched := strings.TrimSpace(override.RefreshSchedule); sched != "" {
			schedule = sched
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return BudgetSummary{}, err
	}
	used, err := s.calcUsageUSD(ctx, tenantID, schedule)
	if err != nil {
		return BudgetSummary{}, err
	}
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return BudgetSummary{
		LimitUSD:         limit,
		UsedUSD:          used,
		RemainingUSD:     remaining,
		WarningThreshold: warn,
		RefreshSchedule:  config.NormalizeBudgetRefreshSchedule(schedule),
	}, nil
}

func (s *Service) calcUsageUSD(ctx context.Context, tenantID uuid.UUID, schedule string) (float64, error) {
	start, end := budgetWindowBounds(time.Now().In(s.timezone), schedule)
	row, err := s.queries.SumUsageForTenant(ctx, db.SumUsageForTenantParams{
		TenantID: toPgUUID(tenantID),
		Ts:       pgtype.Timestamptz{Time: start, Valid: true},
		Ts_2:     pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return float64(row.TotalCostCents) / 100.0, nil
}

func budgetWindowBounds(now time.Time, schedule string) (time.Time, time.Time) {
	nowUTC := now.UTC()
	normalized := config.NormalizeBudgetRefreshSchedule(schedule)
	switch normalized {
	case "weekly":
		year, month, day := nowUTC.Date()
		weekday := int(nowUTC.Weekday())
		delta := (weekday + 6) % 7
		start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -delta)
		return start, start.AddDate(0, 0, 7)
	case "calendar_month":
		year, month, _ := nowUTC.Date()
		start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		if days, ok := config.BudgetRollingWindowDays(normalized); ok && days > 0 {
			start := nowUTC.AddDate(0, 0, -days)
			return start, nowUTC
		}
		start := time.Date(nowUTC.Year(), nowUTC.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.Nil, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

func timeFromPg(ts pgtype.Timestamptz) (time.Time, error) {
	if !ts.Valid {
		return time.Time{}, errors.New("invalid timestamp")
	}
	return ts.Time, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

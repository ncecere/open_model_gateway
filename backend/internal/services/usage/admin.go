package usage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	"math"
	"strings"
	"time"
)

func (s *Service) SummarizeAdminUsage(ctx context.Context, period string, tenantID *uuid.UUID, timezone string, startOverride, endOverride *time.Time) (AdminUsageSummary, error) {
	if s == nil || s.queries == nil {
		return AdminUsageSummary{}, errors.New("usage service not initialized")
	}
	var (
		start, end  time.Time
		loc         *time.Location
		zone        string
		periodLabel string
	)
	if startOverride != nil && endOverride != nil {
		loc = timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(timezone); tz != "" {
			custom, err := time.LoadLocation(tz)
			if err != nil {
				return AdminUsageSummary{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start = startOverride.In(loc)
		end = endOverride.In(loc)
		if !end.After(start) {
			return AdminUsageSummary{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return AdminUsageSummary{}, ErrInvalidRange
		}
		zone = loc.String()
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		periodLabel = fmt.Sprintf("custom_%dd", days)
	} else {
		window, err := s.newWindow(period, timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return AdminUsageSummary{}, ErrInvalidTimezone
			}
			return AdminUsageSummary{}, ErrInvalidPeriod
		}
		loc = window.Location()
		zone = window.Timezone()
		start, end = window.Bounds()
		periodLabel = window.Period()
	}

	var tenantParam pgtype.UUID
	var tenantRef *string
	if tenantID != nil && *tenantID != uuid.Nil {
		tenantParam = toPgUUID(*tenantID)
		idCopy := tenantID.String()
		tenantRef = &idCopy
	}

	sum, err := s.queries.SumUsage(ctx, db.SumUsageParams{
		Column1: tenantParam,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AdminUsageSummary{}, err
	}

	dailyRows, err := s.queries.AggregateUsageDaily(ctx, db.AggregateUsageDailyParams{
		Column1: tenantParam,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return AdminUsageSummary{}, err
	}

	points := buildAggregateUsagePoints(start, end, dailyRows, loc)
	return AdminUsageSummary{
		Period:         periodLabel,
		Start:          start.In(loc).Format(time.RFC3339),
		End:            end.In(loc).Format(time.RFC3339),
		Timezone:       zone,
		TotalRequests:  sum.TotalRequests,
		TotalTokens:    sum.TotalTokens,
		TotalCostCents: sum.TotalCostCents,
		TotalCostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		Points:         points,
		TenantID:       tenantRef,
	}, nil
}

// BreakdownAdminUsage returns aggregate lists + series grouped by tenant or model.

func (s *Service) BreakdownAdminUsage(ctx context.Context, params AdminBreakdownParams) (AdminBreakdown, error) {
	if s == nil || s.queries == nil {
		return AdminBreakdown{}, errors.New("usage service not initialized")
	}
	var (
		window timeutil.Window
		err    error
	)
	if params.StartOverride != nil && params.EndOverride != nil {
		loc := timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(params.Timezone); tz != "" {
			custom, tzErr := time.LoadLocation(tz)
			if tzErr != nil {
				return AdminBreakdown{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start := params.StartOverride.In(loc)
		end := params.EndOverride.In(loc)
		if !end.After(start) {
			return AdminBreakdown{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return AdminBreakdown{}, ErrInvalidRange
		}
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		window, err = timeutil.NewWindowFromRange(start, end, loc, fmt.Sprintf("custom_%dd", days))
		if err != nil {
			return AdminBreakdown{}, err
		}
	} else {
		window, err = s.newWindow(params.Period, params.Timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return AdminBreakdown{}, ErrInvalidTimezone
			}
			return AdminBreakdown{}, ErrInvalidPeriod
		}
	}
	loc := window.Location()
	zone := window.Timezone()
	start, end := window.Bounds()

	group := strings.TrimSpace(strings.ToLower(params.Group))
	if group == "" {
		group = "tenant"
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 5
	}

	result := AdminBreakdown{
		Group:    group,
		Period:   window.Period(),
		Start:    window.StartString(),
		End:      window.EndString(),
		Timezone: zone,
		Items:    make([]AdminBreakdownItem, 0),
		Series: AdminBreakdownSeries{
			Timezone: zone,
		},
	}

	selected := strings.TrimSpace(params.EntityID)

	switch group {
	case "tenant":
		rows, err := s.queries.AggregateUsageByTenant(ctx, db.AggregateUsageByTenantParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			tenantID, err := uuidFromPg(row.TenantID)
			if err != nil {
				continue
			}
			id := tenantID.String()
			labelMap[id] = row.Name
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        id,
				Label:     row.Name,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			if tenantUUID, err := uuid.Parse(selected); err == nil {
				dailyRows, err := s.queries.AggregateTenantUsageDaily(ctx, db.AggregateTenantUsageDailyParams{
					TenantID: toPgUUID(tenantUUID),
					Ts:       toPgTime(start),
					Ts_2:     toPgTime(end),
					Column4:  zone,
				})
				if err != nil {
					return AdminBreakdown{}, err
				}
				result.Series.ID = selected
				label := labelMap[selected]
				if label == "" {
					label = selected
				}
				result.Series.Label = label
				result.Series.Points = buildTenantUsagePoints(start, end, dailyRows, loc)
			}
		}
	case "model":
		rows, err := s.queries.AggregateUsageByModel(ctx, db.AggregateUsageByModelParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			label := strings.TrimSpace(row.ModelAlias)
			if label == "" {
				label = "unknown"
			}
			labelMap[label] = label
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        label,
				Label:     label,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			dailyRows, err := s.queries.AggregateModelUsageDaily(ctx, db.AggregateModelUsageDailyParams{
				ModelAlias: selected,
				Ts:         toPgTime(start),
				Ts_2:       toPgTime(end),
				Column4:    zone,
			})
			if err != nil {
				return AdminBreakdown{}, err
			}
			result.Series.ID = selected
			label := labelMap[selected]
			if label == "" {
				label = selected
			}
			result.Series.Label = label
			result.Series.Points = buildModelUsagePoints(start, end, dailyRows, loc)
		}
	case "user":
		rows, err := s.queries.AggregateUsageByUser(ctx, db.AggregateUsageByUserParams{
			Ts:    toPgTime(start),
			Ts_2:  toPgTime(end),
			Limit: int32(limit),
		})
		if err != nil {
			return AdminBreakdown{}, err
		}
		labelMap := make(map[string]string, len(rows))
		for _, row := range rows {
			userID, err := uuidFromPg(row.UserID)
			if err != nil {
				continue
			}
			id := userID.String()
			label := strings.TrimSpace(row.Name)
			if label == "" {
				label = strings.TrimSpace(row.Email)
			}
			if label == "" {
				label = id
			}
			labelMap[id] = label
			result.Items = append(result.Items, AdminBreakdownItem{
				ID:        id,
				Label:     label,
				Requests:  row.Requests,
				Tokens:    row.Tokens,
				CostCents: row.CostCents,
				CostUSD:   microsToUSD(row.CostUsdMicros),
			})
		}
		if selected == "" && len(result.Items) > 0 {
			selected = result.Items[0].ID
		}
		if selected != "" {
			if userUUID, err := uuid.Parse(selected); err == nil {
				pgIDs := toPgUUIDArray([]uuid.UUID{userUUID})
				dailyRows, err := s.queries.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
					Column1: pgIDs,
					Ts:      toPgTime(start),
					Ts_2:    toPgTime(end),
					Column4: zone,
				})
				if err != nil {
					return AdminBreakdown{}, err
				}
				userDaily := groupUserDailyAggregates(dailyRows, loc)
				result.Series.ID = selected
				label := labelMap[selected]
				if label == "" {
					label = selected
				}
				result.Series.Label = label
				result.Series.Points = buildUsagePointsFromDailyMap(start, end, loc, userDaily[selected])
			}
		}
	default:
		return AdminBreakdown{}, ErrInvalidBreakdownType
	}

	return result, nil
}

func buildTenantUsagePoints(start, end time.Time, rows []db.AggregateTenantUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateTenantUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildAggregateUsagePoints(start, end time.Time, rows []db.AggregateUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

func buildModelUsagePoints(start, end time.Time, rows []db.AggregateModelUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateModelUsageDailyRow, len(rows))
	for _, row := range rows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		daily[day.Unix()] = row
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var requests, tokens, cost int64
		var costUSD float64
		if row, ok := daily[key]; ok {
			requests = row.Requests
			tokens = row.Tokens
			cost = row.CostCents
			costUSD = microsToUSD(row.CostUsdMicros)
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  requests,
			Tokens:    tokens,
			CostCents: cost,
			CostUSD:   costUSD,
		})
	}
	return points
}

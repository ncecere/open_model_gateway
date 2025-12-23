package usage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	"math"
	"strings"
	"time"
)

func (s *Service) buildTenantCompareSeries(ctx context.Context, tenantIDs []uuid.UUID, start, end time.Time, zone string, loc *time.Location) ([]UsageCompareSeries, error) {
	if len(tenantIDs) == 0 {
		return nil, nil
	}
	pgIDs := toPgUUIDArray(tenantIDs)
	metaRows, err := s.queries.ListTenantsByIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ListTenantsByIDsRow, len(metaRows))
	for _, row := range metaRows {
		meta[pgUUIDString(row.ID)] = row
	}
	totalRows, err := s.queries.SumUsageByTenants(ctx, db.SumUsageByTenantsParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		id := pgUUIDString(row.TenantID)
		totalMap[id] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByTenants(ctx, db.AggregateUsageDailyByTenantsParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return nil, err
	}
	daily := make(map[string]map[int64]dailyAggregate, len(tenantIDs))
	for _, row := range dailyRows {
		id := pgUUIDString(row.TenantID)
		if id == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := daily[id]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			daily[id] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	series := make([]UsageCompareSeries, 0, len(tenantIDs))
	for _, id := range tenantIDs {
		idStr := id.String()
		label := idStr
		var tenantStatus, tenantKind string
		if row, ok := meta[idStr]; ok {
			label = row.Name
			tenantStatus = string(row.Status)
			tenantKind = string(row.Kind)
			if row.Kind == db.TenantKindPersonal {
				label = "Personal"
			}
		}
		totals := totalMap[idStr]
		points := buildUsagePointsFromDailyMap(start, end, loc, daily[idStr])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		idCopy := idStr
		series = append(series, UsageCompareSeries{
			Kind:         UsageCompareSeriesTenant,
			ID:           idStr,
			Label:        label,
			TenantID:     &idCopy,
			TenantStatus: tenantStatus,
			TenantKind:   tenantKind,
			Totals:       totals,
			Points:       points,
			ActiveStart:  activeStart,
			ActiveEnd:    activeEnd,
		})
	}
	return series, nil
}

func (s *Service) buildModelCompareSeries(ctx context.Context, aliases []string, start, end time.Time, zone string, loc *time.Location, scope []pgtype.UUID) ([]UsageCompareSeries, error) {
	if len(aliases) == 0 {
		return nil, nil
	}
	metaRows, err := s.queries.ListModelCatalogByAliases(ctx, aliases)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ModelCatalog, len(metaRows))
	for _, row := range metaRows {
		meta[row.Alias] = row
	}
	totalRows, err := s.queries.SumUsageByModels(ctx, db.SumUsageByModelsParams{
		Column1: aliases,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: scope,
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		totalMap[row.ModelAlias] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByModels(ctx, db.AggregateUsageDailyByModelsParams{
		Column1: aliases,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
		Column5: scope,
	})
	if err != nil {
		return nil, err
	}
	daily := make(map[string]map[int64]dailyAggregate, len(aliases))
	for _, row := range dailyRows {
		alias := row.ModelAlias
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := daily[alias]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			daily[alias] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	series := make([]UsageCompareSeries, 0, len(aliases))
	for _, alias := range aliases {
		label := alias
		provider := ""
		if row, ok := meta[alias]; ok {
			provider = catalog.NormalizeProviderSlug(row.Provider)
		}
		totals := totalMap[alias]
		points := buildUsagePointsFromDailyMap(start, end, loc, daily[alias])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		series = append(series, UsageCompareSeries{
			Kind:        UsageCompareSeriesModel,
			ID:          alias,
			Label:       label,
			Provider:    provider,
			Totals:      totals,
			Points:      points,
			ActiveStart: activeStart,
			ActiveEnd:   activeEnd,
		})
	}
	return series, nil
}

// ModelPerformance aggregates recent throughput and latency metrics per model alias.

func (s *Service) ModelPerformance(ctx context.Context, start, end time.Time) (map[string]ModelPerformanceStats, error) {
	result := make(map[string]ModelPerformanceStats)
	if s == nil || s.queries == nil {
		return result, nil
	}
	if end.Before(start) {
		start, end = end, start
	}
	windowSeconds := end.Sub(start).Seconds()
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	metricRows, err := s.queries.AggregateRequestMetricsByModel(ctx, db.AggregateRequestMetricsByModelParams{
		Ts:   toPgTime(start),
		Ts_2: toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	for _, row := range metricRows {
		stats := ModelPerformanceStats{
			Alias:                  row.ModelAlias,
			Requests:               row.Requests,
			Tokens:                 row.Tokens,
			ThroughputTokensPerSec: float64(row.Tokens) / windowSeconds,
		}
		result[row.ModelAlias] = stats
	}
	latencyRows, err := s.queries.AggregateLatencyByModel(ctx, db.AggregateLatencyByModelParams{
		Ts:   toPgTime(start),
		Ts_2: toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	for _, row := range latencyRows {
		stats := result[row.ModelAlias]
		stats.Alias = row.ModelAlias
		stats.AvgLatencyMs = row.AvgLatencyMs
		stats.P95LatencyMs = row.P95LatencyMs
		stats.LatencySampleCount = row.SampleCount
		result[row.ModelAlias] = stats
	}
	return result, nil
}

func (s *Service) buildUserCompareSeries(ctx context.Context, userIDs []uuid.UUID, start, end time.Time, zone string, loc *time.Location) ([]UsageCompareSeries, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	pgIDs := toPgUUIDArray(userIDs)
	metaRows, err := s.queries.ListUsersByIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	meta := make(map[string]db.ListUsersByIDsRow, len(metaRows))
	for _, row := range metaRows {
		meta[pgUUIDString(row.ID)] = row
	}
	totalRows, err := s.queries.SumUsageByUsers(ctx, db.SumUsageByUsersParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
	})
	if err != nil {
		return nil, err
	}
	totalMap := make(map[string]UsageTotals, len(totalRows))
	for _, row := range totalRows {
		id := pgUUIDString(row.UserID)
		totalMap[id] = UsageTotals{
			Requests:  row.TotalRequests,
			Tokens:    row.TotalTokens,
			CostCents: row.TotalCostCents,
			CostUSD:   microsToUSD(row.TotalCostUsdMicros),
		}
	}
	dailyRows, err := s.queries.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
		Column1: pgIDs,
		Ts:      toPgTime(start),
		Ts_2:    toPgTime(end),
		Column4: zone,
	})
	if err != nil {
		return nil, err
	}
	userDaily := groupUserDailyAggregates(dailyRows, loc)
	series := make([]UsageCompareSeries, 0, len(userIDs))
	for _, id := range userIDs {
		idStr := id.String()
		metaRow := meta[idStr]
		label := strings.TrimSpace(metaRow.Name)
		if label == "" {
			label = strings.TrimSpace(metaRow.Email)
		}
		if label == "" {
			label = idStr
		}
		totals := totalMap[idStr]
		points := buildUsagePointsFromDailyMap(start, end, loc, userDaily[idStr])
		activeStart, activeEnd := deriveUsageActiveRange(points)
		idCopy := idStr
		series = append(series, UsageCompareSeries{
			Kind:        UsageCompareSeriesUser,
			ID:          idStr,
			Label:       label,
			UserID:      &idCopy,
			UserEmail:   metaRow.Email,
			UserName:    metaRow.Name,
			Totals:      totals,
			Points:      points,
			ActiveStart: activeStart,
			ActiveEnd:   activeEnd,
		})
	}
	return series, nil
}

// CompareUsage returns parallel usage series for the requested tenants/models so dashboards can overlay them.

func (s *Service) CompareUsage(ctx context.Context, params CompareUsageParams) (MultiEntityUsage, error) {
	if s == nil || s.queries == nil {
		return MultiEntityUsage{}, errors.New("usage service not initialized")
	}
	period := strings.TrimSpace(params.Period)
	if period == "" {
		period = "30d"
	}
	timezone := strings.TrimSpace(params.Timezone)
	tenants := dedupUUIDs(params.TenantIDs)
	aliases := dedupStrings(params.ModelAliases)
	users := dedupUUIDs(params.UserIDs)
	totalRequested := len(tenants) + len(aliases) + len(users)
	if totalRequested == 0 {
		return MultiEntityUsage{}, ErrNoEntitiesSelected
	}
	if totalRequested > maxUsageCompareSeries {
		return MultiEntityUsage{}, ErrEntityLimitExceeded
	}
	var (
		loc         *time.Location
		zone        string
		start, end  time.Time
		periodLabel string
	)
	if params.Start != nil || params.End != nil {
		if params.Start == nil || params.End == nil {
			return MultiEntityUsage{}, ErrInvalidRange
		}
		loc = timeutil.EnsureLocation(s.location())
		if timezone != "" {
			custom, err := time.LoadLocation(timezone)
			if err != nil {
				return MultiEntityUsage{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start = params.Start.In(loc)
		end = params.End.In(loc)
		if !end.After(start) {
			return MultiEntityUsage{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return MultiEntityUsage{}, ErrInvalidRange
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
				return MultiEntityUsage{}, ErrInvalidTimezone
			}
			return MultiEntityUsage{}, ErrInvalidPeriod
		}
		loc = window.Location()
		zone = window.Timezone()
		start, end = window.Bounds()
		periodLabel = window.Period()
	}
	scopeIDs := dedupUUIDs(params.TenantScope)
	scopeParam := toPgUUIDArray(scopeIDs)
	result := MultiEntityUsage{
		Period:   periodLabel,
		Start:    windowTimeString(start, loc),
		End:      windowTimeString(end, loc),
		Timezone: zone,
		Series:   make([]UsageCompareSeries, 0, totalRequested),
	}
	if len(tenants) > 0 {
		series, err := s.buildTenantCompareSeries(ctx, tenants, start, end, zone, loc)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	if len(aliases) > 0 {
		series, err := s.buildModelCompareSeries(ctx, aliases, start, end, zone, loc, scopeParam)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	if len(users) > 0 {
		series, err := s.buildUserCompareSeries(ctx, users, start, end, zone, loc)
		if err != nil {
			return MultiEntityUsage{}, err
		}
		result.Series = append(result.Series, series...)
	}
	return result, nil
}

// TenantDailyUsage aggregates per-day totals and per-key breakdowns for a tenant between start and end.

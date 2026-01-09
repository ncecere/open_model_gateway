package usage

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	"sort"
	"strings"
	"time"
)

func (s *Service) TenantDailyUsage(ctx context.Context, tenantID uuid.UUID, start, end time.Time, timezone string) (TenantDailyUsageResponse, error) {
	if s == nil || s.repo == nil {
		return TenantDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	if tenantID == uuid.Nil {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return TenantDailyUsageResponse{}, ErrInvalidRange
	}

	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return TenantDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	dailyRows, err := s.repo.AggregateTenantUsageDaily(ctx, db.AggregateTenantUsageDailyParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(startDay),
		Ts_2:     toPgTime(endDay),
		Column4:  zone,
	})
	if err != nil {
		return TenantDailyUsageResponse{}, err
	}
	keyRows, err := s.repo.AggregateTenantUsageDailyByAPIKeys(ctx, db.AggregateTenantUsageDailyByAPIKeysParams{
		TenantID: toPgUUID(tenantID),
		Ts:       toPgTime(startDay),
		Ts_2:     toPgTime(endDay),
		Column4:  zone,
	})
	if err != nil {
		return TenantDailyUsageResponse{}, err
	}

	dailyBuckets := make(map[int64]dailyAggregate, len(dailyRows))
	for _, row := range dailyRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		dailyBuckets[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}

	keyBuckets := make(map[int64][]TenantDailyKeyUsage)
	for _, row := range keyRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		keyBuckets[day.Unix()] = append(keyBuckets[day.Unix()], TenantDailyKeyUsage{
			APIKeyID:     pgUUIDString(row.ApiKeyID),
			APIKeyName:   row.ApiKeyName,
			APIKeyPrefix: row.ApiKeyPrefix,
			Requests:     row.Requests,
			Tokens:       row.Tokens,
			CostCents:    row.CostCents,
			CostUSD:      microsToUSD(row.CostUsdMicros),
		})
	}

	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]TenantDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		aggregate := dailyBuckets[key]
		keys := keyBuckets[key]
		if keys == nil {
			keys = make([]TenantDailyKeyUsage, 0)
		}
		if len(keys) > 1 {
			sort.Slice(keys, func(i, j int) bool {
				if keys[i].CostCents == keys[j].CostCents {
					return keys[i].Requests > keys[j].Requests
				}
				return keys[i].CostCents > keys[j].CostCents
			})
		}
		days = append(days, TenantDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Keys:      keys,
		})
	}

	return TenantDailyUsageResponse{
		TenantID: tenantID.String(),
		Start:    windowTimeString(startDay, loc),
		End:      windowTimeString(endDay, loc),
		Timezone: zone,
		Days:     days,
	}, nil
}

// UserDailyUsage aggregates daily totals for a user with per-tenant breakdowns.

func (s *Service) UserDailyUsage(ctx context.Context, userID uuid.UUID, start, end time.Time, timezone string) (UserDailyUsageResponse, error) {
	if s == nil || s.repo == nil {
		return UserDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	if userID == uuid.Nil {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return UserDailyUsageResponse{}, ErrInvalidRange
	}
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return UserDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	userRows, err := s.repo.AggregateUsageDailyByUsers(ctx, db.AggregateUsageDailyByUsersParams{
		Column1: toPgUUIDArray([]uuid.UUID{userID}),
		Ts:      toPgTime(startDay),
		Ts_2:    toPgTime(endDay),
		Column4: zone,
	})
	if err != nil {
		return UserDailyUsageResponse{}, err
	}
	userDaily := groupUserDailyAggregates(userRows, loc)
	dailyTotals := userDaily[userID.String()]

	tenantRows, err := s.repo.AggregateUserUsageDailyByTenants(ctx, db.AggregateUserUsageDailyByTenantsParams{
		OwnerUserID: toPgUUID(userID),
		Ts:          toPgTime(startDay),
		Ts_2:        toPgTime(endDay),
		Column4:     zone,
	})
	if err != nil {
		return UserDailyUsageResponse{}, err
	}
	tenantBuckets := make(map[int64][]UserDailyTenantUsage)
	for _, row := range tenantRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		tenantBuckets[day.Unix()] = append(tenantBuckets[day.Unix()], UserDailyTenantUsage{
			TenantID:   pgUUIDString(row.TenantID),
			TenantName: row.TenantName,
			Requests:   row.Requests,
			Tokens:     row.Tokens,
			CostCents:  row.CostCents,
			CostUSD:    microsToUSD(row.CostUsdMicros),
		})
	}

	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]UserDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		tenants := tenantBuckets[key]
		if tenants == nil {
			tenants = make([]UserDailyTenantUsage, 0)
		} else if len(tenants) > 1 {
			sort.Slice(tenants, func(i, j int) bool {
				if tenants[i].CostCents == tenants[j].CostCents {
					return tenants[i].Requests > tenants[j].Requests
				}
				return tenants[i].CostCents > tenants[j].CostCents
			})
		}
		aggregate, ok := dailyTotals[key]
		if (!ok || aggregate == (dailyAggregate{})) && len(tenants) > 0 {
			aggregate = sumUserTenantUsage(tenants)
		}
		days = append(days, UserDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Tenants:   tenants,
		})
	}

	return UserDailyUsageResponse{
		UserID:   userID.String(),
		Start:    windowTimeString(startDay, loc),
		End:      windowTimeString(endDay, loc),
		Timezone: zone,
		Days:     days,
	}, nil
}

// ModelDailyUsage aggregates daily totals for a model alias with per-tenant breakdowns.

func (s *Service) ModelDailyUsage(ctx context.Context, alias string, start, end time.Time, timezone string, tenantScope []uuid.UUID) (ModelDailyUsageResponse, error) {
	if s == nil || s.repo == nil {
		return ModelDailyUsageResponse{}, errors.New("usage service not initialized")
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	if !end.After(start) {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	if end.Sub(start) > maxCustomCompareWindow {
		return ModelDailyUsageResponse{}, ErrInvalidRange
	}
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(timezone); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return ModelDailyUsageResponse{}, ErrInvalidTimezone
		}
		loc = custom
	}
	zone := loc.String()
	startDay := timeutil.TruncateToDay(start.In(loc), loc)
	endDay := timeutil.TruncateToDay(end.In(loc), loc)
	if !endDay.After(startDay) {
		endDay = startDay.AddDate(0, 0, 1)
	}

	scopeIDs := dedupUUIDs(tenantScope)
	scopeParam := toPgUUIDArray(scopeIDs)
	allowedTenants := make(map[string]struct{})
	for _, id := range scopeIDs {
		allowedTenants[id.String()] = struct{}{}
	}
	modelRows, err := s.repo.AggregateUsageDailyByModels(ctx, db.AggregateUsageDailyByModelsParams{
		Column1: []string{alias},
		Ts:      toPgTime(startDay),
		Ts_2:    toPgTime(endDay),
		Column4: zone,
		Column5: scopeParam,
	})
	if err != nil {
		return ModelDailyUsageResponse{}, err
	}
	modelDaily := groupModelDailyAggregates(modelRows, loc)
	dailyTotals := modelDaily[alias]

	tenantRows, err := s.repo.AggregateModelUsageDailyByTenants(ctx, db.AggregateModelUsageDailyByTenantsParams{
		ModelAlias: alias,
		Ts:         toPgTime(startDay),
		Ts_2:       toPgTime(endDay),
		Column4:    zone,
	})
	if err != nil {
		return ModelDailyUsageResponse{}, err
	}
	tenantBuckets := make(map[int64][]ModelDailyTenantUsage)
	for _, row := range tenantRows {
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		tenantID := pgUUIDString(row.TenantID)
		if len(allowedTenants) > 0 {
			if _, ok := allowedTenants[tenantID]; !ok {
				continue
			}
		}
		if tenantID == "" {
			continue
		}
		tenantBuckets[day.Unix()] = append(tenantBuckets[day.Unix()], ModelDailyTenantUsage{
			TenantID:   tenantID,
			TenantName: row.TenantName,
			Requests:   row.Requests,
			Tokens:     row.Tokens,
			CostCents:  row.CostCents,
			CostUSD:    microsToUSD(row.CostUsdMicros),
		})
	}
	totalDays := int(endDay.Sub(startDay).Hours()/24 + 0.5)
	if totalDays < 1 {
		totalDays = 1
	}
	days := make([]ModelDailyUsageDay, 0, totalDays)
	for day := startDay; day.Before(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		tenants := tenantBuckets[key]
		if tenants == nil {
			tenants = make([]ModelDailyTenantUsage, 0)
		} else if len(tenants) > 1 {
			sort.Slice(tenants, func(i, j int) bool {
				if tenants[i].CostCents == tenants[j].CostCents {
					return tenants[i].Requests > tenants[j].Requests
				}
				return tenants[i].CostCents > tenants[j].CostCents
			})
		}
		aggregate, ok := dailyTotals[key]
		if (!ok || aggregate == (dailyAggregate{})) && len(tenants) > 0 {
			aggregate = sumModelTenantUsage(tenants)
		}
		days = append(days, ModelDailyUsageDay{
			Date:      day.Format(time.RFC3339),
			Requests:  aggregate.Requests,
			Tokens:    aggregate.Tokens,
			CostCents: aggregate.CostCents,
			CostUSD:   microsToUSD(aggregate.CostUsdMicros),
			Tenants:   tenants,
		})
	}

	return ModelDailyUsageResponse{
		ModelAlias: alias,
		Start:      windowTimeString(startDay, loc),
		End:        windowTimeString(endDay, loc),
		Timezone:   zone,
		Days:       days,
	}, nil
}

// SummarizeAdminUsage aggregates system-wide or tenant-scoped usage for admin dashboards.

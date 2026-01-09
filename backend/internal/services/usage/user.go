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

func (s *Service) SummarizeUserUsage(ctx context.Context, user db.User, period string, tenantFilter *uuid.UUID, timezone string, startOverride, endOverride *time.Time) (UserSummary, error) {
	if s == nil || s.repo == nil {
		return UserSummary{}, errors.New("usage service not initialized")
	}

	var (
		window timeutil.Window
		err    error
	)
	if startOverride != nil && endOverride != nil {
		loc := timeutil.EnsureLocation(s.location())
		if tz := strings.TrimSpace(timezone); tz != "" {
			custom, tzErr := time.LoadLocation(tz)
			if tzErr != nil {
				return UserSummary{}, ErrInvalidTimezone
			}
			loc = custom
		}
		start := startOverride.In(loc)
		end := endOverride.In(loc)
		if !end.After(start) {
			return UserSummary{}, ErrInvalidRange
		}
		if end.Sub(start) > maxCustomCompareWindow {
			return UserSummary{}, ErrInvalidRange
		}
		days := int(math.Ceil(end.Sub(start).Hours() / 24))
		if days <= 0 {
			days = 1
		}
		label := fmt.Sprintf("custom_%dd", days)
		window, err = timeutil.NewWindowFromRange(start, end, loc, label)
		if err != nil {
			return UserSummary{}, err
		}
	} else {
		window, err = s.newWindow(period, timezone)
		if err != nil {
			if errors.Is(err, ErrInvalidTimezone) {
				return UserSummary{}, ErrInvalidTimezone
			}
			return UserSummary{}, ErrInvalidPeriod
		}
	}
	zone := window.Timezone()

	summary := UserSummary{
		Period:      window.Period(),
		Start:       window.StartString(),
		End:         window.EndString(),
		Timezone:    zone,
		Totals:      UsageTotals{},
		Memberships: make([]UserTenantUsage, 0),
	}

	entries := make([]scopeEntry, 0)

	var personalTenant uuid.UUID
	if user.PersonalTenantID.Valid {
		if id, err := uuidFromPg(user.PersonalTenantID); err == nil {
			personalTenant = id
		}
	}

	if personalTenant != uuid.Nil {
		personalTotals, err := s.sumUsageForUserTenant(ctx, user, personalTenant, window)
		if err != nil {
			return UserSummary{}, err
		}
		personalUsage := UserTenantUsage{
			TenantID:   personalTenant.String(),
			Name:       user.Email,
			Role:       string(db.MembershipRoleOwner),
			Status:     "active",
			Requests:   personalTotals.Requests,
			Tokens:     personalTotals.Tokens,
			CostCents:  personalTotals.CostCents,
			CostUSD:    personalTotals.CostUSD,
			IsPersonal: true,
		}
		summary.Personal = &personalUsage
		scope := UsageScope{
			ID:     "personal",
			Kind:   UsageScopePersonal,
			Name:   "Personal",
			Totals: personalTotals,
		}
		idCopy := personalTenant.String()
		scope.TenantID = &idCopy
		scope.Role = string(db.MembershipRoleOwner)
		scope.Status = "active"
		entries = append(entries, scopeEntry{scope: scope, tenant: personalTenant})
	} else {
		zeroUsage := UserTenantUsage{
			TenantID:   "",
			Name:       user.Email,
			Role:       string(db.MembershipRoleOwner),
			Status:     "active",
			IsPersonal: true,
		}
		summary.Personal = &zeroUsage
		scope := UsageScope{
			ID:     "personal",
			Kind:   UsageScopePersonal,
			Name:   "Personal",
			Role:   string(db.MembershipRoleOwner),
			Status: "active",
			Totals: UsageTotals{},
		}
		entries = append(entries, scopeEntry{scope: scope, tenant: uuid.Nil})
	}

	ownedTenants, err := s.repo.ListUserOwnedTenants(ctx, user.ID)
	if err != nil {
		return UserSummary{}, err
	}

	for _, row := range ownedTenants {
		tenantUUID, err := uuidFromPg(row.TenantID)
		if err != nil {
			continue
		}
		tenantTotals, err := s.sumUsageForUserTenant(ctx, user, tenantUUID, window)
		if err != nil {
			return UserSummary{}, err
		}
		tenantIDStr := tenantUUID.String()
		scope := UsageScope{
			ID:       tenantIDStr,
			Kind:     UsageScopeTenant,
			TenantID: &tenantIDStr,
			Name:     row.Name,
			Role:     string(row.Role),
			Status:   string(row.Status),
			Totals:   tenantTotals,
		}
		entries = append(entries, scopeEntry{scope: scope, tenant: tenantUUID})

		summary.Memberships = append(summary.Memberships, UserTenantUsage{
			TenantID:   tenantIDStr,
			Name:       row.Name,
			Role:       string(row.Role),
			Status:     string(row.Status),
			Requests:   tenantTotals.Requests,
			Tokens:     tenantTotals.Tokens,
			CostCents:  tenantTotals.CostCents,
			CostUSD:    tenantTotals.CostUSD,
			IsPersonal: false,
		})
	}

	selectedEntry := entries[0]
	if tenantFilter != nil && *tenantFilter != uuid.Nil {
		found := false
		for _, entry := range entries {
			if entry.tenant == *tenantFilter {
				selectedEntry = entry
				found = true
				break
			}
		}
		if !found {
			return UserSummary{}, ErrScopeForbidden
		}
	}

	for _, entry := range entries {
		summary.Totals.addTotals(entry.scope.Totals)
	}

	detail, err := s.buildScopeDetail(ctx, user, selectedEntry, window)
	if err != nil {
		return UserSummary{}, err
	}

	scopes := make([]UsageScope, 0, len(entries))
	for _, entry := range entries {
		scopes = append(scopes, entry.scope)
	}
	summary.Scopes = scopes
	summary.SelectedScope = &detail

	summary.PersonalSeries = detail.Series
	summary.PersonalKeys = detail.APIKeys
	summary.RecentRequests = detail.RecentRequests

	return summary, nil
}

// SummarizeAPIKeyUsage aggregates usage for a single API key owned by the caller over the requested period.

func (s *Service) SummarizeAPIKeyUsage(ctx context.Context, key db.ApiKey, period, timezone string) (APIKeyUsageSummary, error) {
	if s == nil || s.repo == nil {
		return APIKeyUsageSummary{}, errors.New("usage service not initialized")
	}
	window, err := s.newWindow(period, timezone)
	if err != nil {
		if errors.Is(err, ErrInvalidTimezone) {
			return APIKeyUsageSummary{}, ErrInvalidTimezone
		}
		return APIKeyUsageSummary{}, ErrInvalidPeriod
	}
	loc := window.Location()
	zone := window.Timezone()
	start, end := window.Bounds()
	sum, err := s.repo.SumUsageForAPIKey(ctx, db.SumUsageForAPIKeyParams{
		ApiKeyID: key.ID,
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return APIKeyUsageSummary{}, err
	}
	rows, err := s.repo.AggregateAPIKeyUsageDaily(ctx, db.AggregateAPIKeyUsageDailyParams{
		ApiKeyID: key.ID,
		Ts:       toPgTime(start),
		Ts_2:     toPgTime(end),
		Column4:  zone,
	})
	if err != nil {
		return APIKeyUsageSummary{}, err
	}
	series := buildAPIKeyUsagePoints(start, end, rows, loc)
	return APIKeyUsageSummary{
		APIKeyID: pgUUIDString(key.ID),
		Period:   window.Period(),
		Start:    window.StartString(),
		End:      window.EndString(),
		Timezone: zone,
		Totals: UsageTotals{
			Requests:  sum.TotalRequests,
			Tokens:    sum.TotalTokens,
			CostCents: sum.TotalCostCents,
			CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		},
		Series: series,
	}, nil
}

func (s *Service) sumUsageForUserTenant(ctx context.Context, user db.User, tenantID uuid.UUID, window timeutil.Window) (UsageTotals, error) {
	if tenantID == uuid.Nil {
		return UsageTotals{}, nil
	}
	start, end := window.Bounds()
	sum, err := s.repo.SumUsageForUserTenant(ctx, db.SumUsageForUserTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(tenantID),
		Ts:          toPgTime(start),
		Ts_2:        toPgTime(end),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return UsageTotals{}, err
	}
	return UsageTotals{
		Requests:  sum.TotalRequests,
		Tokens:    sum.TotalTokens,
		CostCents: sum.TotalCostCents,
		CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
	}, nil
}

func (s *Service) buildScopeDetail(ctx context.Context, user db.User, entry scopeEntry, window timeutil.Window) (UserScopeDetail, error) {
	detail := UserScopeDetail{
		Scope:          entry.scope,
		Series:         make([]UsagePoint, 0),
		APIKeys:        make([]APIKeyUsageDigest, 0),
		RecentRequests: make([]RecentRequest, 0),
	}
	if entry.tenant == uuid.Nil {
		return detail, nil
	}
	start, end := window.Bounds()
	zone := window.Timezone()
	loc := window.Location()

	rows, err := s.repo.AggregateUsageDailyForUserTenant(ctx, db.AggregateUsageDailyForUserTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(entry.tenant),
		Ts:          toPgTime(start),
		Ts_2:        toPgTime(end),
		Column5:     zone,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return detail, err
	}
	detail.Series = buildUserScopePoints(start, end, rows, loc)

	apiKeys, reqs, err := s.summarizeKeysForTenant(ctx, user, entry.tenant, window)
	if err != nil {
		return detail, err
	}
	detail.APIKeys = apiKeys
	detail.RecentRequests = reqs
	return detail, nil
}

func buildUserScopePoints(start, end time.Time, rows []db.AggregateUsageDailyForUserTenantRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateUsageDailyForUserTenantRow, len(rows))
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

func buildAPIKeyUsagePoints(start, end time.Time, rows []db.AggregateAPIKeyUsageDailyRow, loc *time.Location) []UsagePoint {
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end, loc)
	daily := make(map[int64]db.AggregateAPIKeyUsageDailyRow, len(rows))
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

func (s *Service) summarizeKeysForTenant(ctx context.Context, user db.User, tenantID uuid.UUID, window timeutil.Window) ([]APIKeyUsageDigest, []RecentRequest, error) {
	start, end := window.Bounds()
	loc := window.Location()
	keys, err := s.repo.ListAPIKeysByOwnerAndTenant(ctx, db.ListAPIKeysByOwnerAndTenantParams{
		OwnerUserID: user.ID,
		TenantID:    toPgUUID(tenantID),
	})
	if err != nil {
		return nil, nil, err
	}
	if len(keys) == 0 {
		return nil, nil, nil
	}
	digests := make([]APIKeyUsageDigest, 0, len(keys))
	keyIDs := make([]pgtype.UUID, 0, len(keys))
	nameMap := make(map[string]string, len(keys))
	for _, key := range keys {
		sum, err := s.repo.SumUsageForAPIKey(ctx, db.SumUsageForAPIKeyParams{
			ApiKeyID: key.ID,
			Ts:       toPgTime(start),
			Ts_2:     toPgTime(end),
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, err
		}
		digest := APIKeyUsageDigest{
			APIKeyID:  pgUUIDString(key.ID),
			Name:      key.Name,
			Prefix:    key.Prefix,
			Requests:  sum.TotalRequests,
			Tokens:    sum.TotalTokens,
			CostCents: sum.TotalCostCents,
			CostUSD:   microsToUSD(sum.TotalCostUsdMicros),
		}
		if key.LastUsedAt.Valid {
			if ts, err := timeFromPg(key.LastUsedAt); err == nil {
				digest.LastUsedAt = &ts
			}
		}
		digests = append(digests, digest)
		keyIDs = append(keyIDs, key.ID)
		nameMap[pgUUIDString(key.ID)] = key.Name
	}
	if len(keyIDs) == 0 {
		return digests, nil, nil
	}
	reqs, err := s.repo.ListRecentRequestsByAPIKeys(ctx, db.ListRecentRequestsByAPIKeysParams{
		Column1: keyIDs,
		Limit:   10,
	})
	if err != nil {
		return digests, nil, err
	}
	if len(reqs) == 0 {
		return digests, nil, nil
	}
	recent := make([]RecentRequest, 0, len(reqs))
	for _, req := range reqs {
		idStr := pgUUIDString(req.ID)
		keyStr := pgUUIDString(req.ApiKeyID)
		var ts time.Time
		if req.Ts.Valid {
			ts = req.Ts.Time.In(loc)
		} else {
			ts = time.Time{}
		}
		var errCode *string
		if req.ErrorCode.Valid {
			v := req.ErrorCode.String
			errCode = &v
		}
		record := RecentRequest{
			ID:         idStr,
			APIKeyID:   keyStr,
			APIKeyName: nameMap[keyStr],
			ModelAlias: req.ModelAlias,
			Provider:   req.Provider,
			Status:     req.Status,
			LatencyMS:  req.LatencyMs,
			CostCents:  req.CostCents,
			CostUSD:    microsToUSD(req.CostUsdMicros),
			Timestamp:  ts,
			ErrorCode:  errCode,
		}
		recent = append(recent, record)
	}
	return digests, recent, nil
}

package usage

import (
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
	"math"
	"strings"
	"time"
)

type dailyAggregate struct {
	Requests      int64
	Tokens        int64
	CostCents     int64
	CostUsdMicros int64
}

func buildUsagePointsFromDailyMap(start, end time.Time, loc *time.Location, daily map[int64]dailyAggregate) []UsagePoint {
	if daily == nil {
		daily = map[int64]dailyAggregate{}
	}
	loc = timeutil.EnsureLocation(loc)
	startDay := timeutil.TruncateToDay(start, loc)
	endDay := timeutil.TruncateToDay(end.Add(-time.Nanosecond), loc)
	if endDay.Before(startDay) {
		endDay = startDay
	}
	points := make([]UsagePoint, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		key := day.Unix()
		var record dailyAggregate
		if row, ok := daily[key]; ok {
			record = row
		}
		points = append(points, UsagePoint{
			Date:      day.Format(time.RFC3339),
			Requests:  record.Requests,
			Tokens:    record.Tokens,
			CostCents: record.CostCents,
			CostUSD:   microsToUSD(record.CostUsdMicros),
		})
	}
	return points
}

func deriveUsageActiveRange(points []UsagePoint) (*string, *string) {
	var start *string
	var end *string
	for _, point := range points {
		if point.Requests > 0 || point.Tokens > 0 || point.CostCents > 0 || point.CostUSD > 0 {
			if start == nil {
				date := point.Date
				start = &date
			}
			date := point.Date
			end = &date
		}
	}
	return start, end
}

func findPoint(points []UsagePoint, date string) UsagePoint {
	for _, point := range points {
		if point.Date == date {
			return point
		}
	}
	return UsagePoint{Date: date}
}

func groupUserDailyAggregates(rows []db.AggregateUsageDailyByUsersRow, loc *time.Location) map[string]map[int64]dailyAggregate {
	loc = timeutil.EnsureLocation(loc)
	data := make(map[string]map[int64]dailyAggregate, len(rows))
	for _, row := range rows {
		id := pgUUIDString(row.UserID)
		if id == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := data[id]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			data[id] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	return data
}

func groupModelDailyAggregates(rows []db.AggregateUsageDailyByModelsRow, loc *time.Location) map[string]map[int64]dailyAggregate {
	loc = timeutil.EnsureLocation(loc)
	data := make(map[string]map[int64]dailyAggregate, len(rows))
	for _, row := range rows {
		alias := strings.TrimSpace(row.ModelAlias)
		if alias == "" {
			continue
		}
		dayTime, err := timeFromPg(row.Day)
		if err != nil {
			continue
		}
		day := timeutil.TruncateToDay(dayTime, loc)
		bucket := data[alias]
		if bucket == nil {
			bucket = make(map[int64]dailyAggregate)
			data[alias] = bucket
		}
		bucket[day.Unix()] = dailyAggregate{
			Requests:      row.Requests,
			Tokens:        row.Tokens,
			CostCents:     row.CostCents,
			CostUsdMicros: row.CostUsdMicros,
		}
	}
	return data
}

func sumUserTenantUsage(tenants []UserDailyTenantUsage) dailyAggregate {
	var agg dailyAggregate
	for _, tenant := range tenants {
		agg.Requests += tenant.Requests
		agg.Tokens += tenant.Tokens
		agg.CostCents += tenant.CostCents
		agg.CostUsdMicros += tenant.CostCents * 10_000
	}
	return agg
}

func sumModelTenantUsage(tenants []ModelDailyTenantUsage) dailyAggregate {
	var agg dailyAggregate
	for _, tenant := range tenants {
		agg.Requests += tenant.Requests
		agg.Tokens += tenant.Tokens
		agg.CostCents += tenant.CostCents
		agg.CostUsdMicros += tenant.CostCents * 10_000
	}
	return agg
}

func windowTimeString(ts time.Time, loc *time.Location) string {
	return ts.In(timeutil.EnsureLocation(loc)).Format(time.RFC3339)
}

func dedupUUIDs(ids []uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func dedupStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		alias := strings.TrimSpace(value)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		result = append(result, alias)
	}
	return result
}

func toPgUUIDArray(ids []uuid.UUID) []pgtype.UUID {
	if len(ids) == 0 {
		return []pgtype.UUID{}
	}
	arr := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		arr = append(arr, toPgUUID(id))
	}
	return arr
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func microsToUSD(micros int64) float64 {
	return float64(micros) / 1_000_000
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

func int64FromAny(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case int:
		return int64(v)
	case uint64:
		if v > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case pgtype.Int8:
		if v.Valid {
			return v.Int64
		}
		return 0
	case pgtype.Numeric:
		if !v.Valid {
			return 0
		}
		floatVal, err := v.Float64Value()
		if err != nil || !floatVal.Valid {
			return 0
		}
		return int64(floatVal.Float64)
	case nil:
		return 0
	default:
		return 0
	}
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgUUIDString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	val, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return ""
	}
	return val.String()
}

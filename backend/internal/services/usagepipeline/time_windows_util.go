package usagepipeline

import (
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func periodBounds(now time.Time, schedule string) (time.Time, time.Time) {
	nowUTC := now.UTC()
	normalized := config.NormalizeBudgetRefreshSchedule(schedule)

	switch normalized {
	case "weekly":
		year, month, day := nowUTC.Date()
		weekday := int(nowUTC.Weekday())
		delta := (weekday + 6) % 7 // Monday = 0
		start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -delta)
		return start, start.AddDate(0, 0, 7)
	default:
		if days, ok := config.BudgetRollingWindowDays(normalized); ok && days > 0 {
			start := nowUTC.AddDate(0, 0, -days)
			return start, nowUTC
		}
	}

	year, month, _ := nowUTC.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end
}

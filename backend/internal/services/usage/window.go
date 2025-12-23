package usage

import (
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/timeutil"
)

func (s *Service) location() *time.Location {
	if s == nil || s.timezone == nil {
		return time.UTC
	}
	return s.timezone
}

func (s *Service) newWindow(period string, overrideTZ string) (timeutil.Window, error) {
	loc := timeutil.EnsureLocation(s.location())
	if tz := strings.TrimSpace(overrideTZ); tz != "" {
		custom, err := time.LoadLocation(tz)
		if err != nil {
			return timeutil.Window{}, ErrInvalidTimezone
		}
		loc = custom
	}
	now := time.Now().In(loc)
	return timeutil.NewWindow(period, now, loc)
}

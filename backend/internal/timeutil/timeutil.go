package timeutil

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidPeriod = errors.New("invalid period")

// Window represents a normalized rolling time window anchored to a location.
type Window struct {
	period string
	start  time.Time
	end    time.Time
	loc    *time.Location
}

// EnsureLocation returns UTC when loc is nil.
func EnsureLocation(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

// NewWindow constructs a rolling window for the requested period (e.g., "7d", "24h").
func NewWindow(period string, now time.Time, loc *time.Location) (Window, error) {
	loc = EnsureLocation(loc)
	now = now.In(loc)
	dur, err := durationFromPeriod(period)
	if err != nil {
		return Window{}, err
	}
	start := now.Add(-dur)
	return Window{
		period: normalizePeriod(period),
		start:  start,
		end:    now,
		loc:    loc,
	}, nil
}

// NewWindowFromRange constructs a window covering the provided [start, end) bounds.
func NewWindowFromRange(start, end time.Time, loc *time.Location, label string) (Window, error) {
	loc = EnsureLocation(loc)
	start = start.In(loc)
	end = end.In(loc)
	if !end.After(start) {
		return Window{}, ErrInvalidPeriod
	}
	p := label
	if strings.TrimSpace(p) == "" {
		p = "custom"
	}
	return Window{
		period: normalizePeriod(p),
		start:  start,
		end:    end,
		loc:    loc,
	}, nil
}

// Period returns the normalized period string (e.g., "7d").
func (w Window) Period() string { return w.period }

// Start returns the inclusive start of the window.
func (w Window) Start() time.Time { return w.start }

// End returns the exclusive end of the window.
func (w Window) End() time.Time { return w.end }

// Bounds returns the start/end timestamps.
func (w Window) Bounds() (time.Time, time.Time) { return w.start, w.end }

// Location returns the reporting timezone for the window.
func (w Window) Location() *time.Location { return EnsureLocation(w.loc) }

// Timezone returns the location name for JSON responses.
func (w Window) Timezone() string { return w.Location().String() }

// StartString returns the start timestamp formatted as RFC3339 in the window's zone.
func (w Window) StartString() string { return w.start.In(w.Location()).Format(time.RFC3339) }

// EndString returns the end timestamp formatted as RFC3339 in the window's zone.
func (w Window) EndString() string { return w.end.In(w.Location()).Format(time.RFC3339) }

// Duration returns the window length.
func (w Window) Duration() time.Duration { return w.end.Sub(w.start) }

// Contains reports whether the timestamp falls within [start, end).
func (w Window) Contains(ts time.Time) bool {
	return !ts.Before(w.start) && ts.Before(w.end)
}

// TruncateToDay normalizes the timestamp to midnight in the provided zone.
func TruncateToDay(t time.Time, loc *time.Location) time.Time {
	loc = EnsureLocation(loc)
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// WindowFromPeriod returns the [start, end) timestamps for a rolling window (7d/30d/90d).
func WindowFromPeriod(period string, now time.Time, loc *time.Location) (time.Time, time.Time, error) {
	w, err := NewWindow(period, now, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return w.start, w.end, nil
}

func durationFromPeriod(period string) (time.Duration, error) {
	p := normalizePeriod(period)
	if len(p) < 2 {
		return 0, ErrInvalidPeriod
	}
	unit := p[len(p)-1]
	valueStr := p[:len(p)-1]
	value, err := strconv.Atoi(valueStr)
	if err != nil || value <= 0 {
		return 0, ErrInvalidPeriod
	}
	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	default:
		return 0, ErrInvalidPeriod
	}
}

func normalizePeriod(period string) string {
	p := strings.ToLower(strings.TrimSpace(period))
	if p == "" {
		return p
	}
	return p
}

package timeutil

import (
	"errors"
	"testing"
	"time"
)

func TestNewWindowDays(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2024, time.November, 7, 12, 0, 0, 0, time.UTC)
	win, err := NewWindow("7d", now, loc)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	if got := win.Period(); got != "7d" {
		t.Fatalf("unexpected period %s", got)
	}
	end := win.End()
	if !end.Equal(now.In(loc)) {
		t.Fatalf("unexpected end %v", end)
	}
	expectedStart := end.Add(-7 * 24 * time.Hour)
	if !win.Start().Equal(expectedStart) {
		t.Fatalf("unexpected start %v", win.Start())
	}
	if win.Timezone() != loc.String() {
		t.Fatalf("unexpected timezone %s", win.Timezone())
	}
	if win.StartString() == "" || win.EndString() == "" {
		t.Fatalf("expected formatted timestamps")
	}
}

func TestNewWindowHours(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, 1, 2, 15, 30, 0, 0, time.UTC)
	win, err := NewWindow("24h", now, loc)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	if win.Duration() != 24*time.Hour {
		t.Fatalf("unexpected duration %v", win.Duration())
	}
	if !win.Contains(now.Add(-12 * time.Hour)) {
		t.Fatalf("expected timestamp within window")
	}
	if win.Contains(now.Add(-25 * time.Hour)) {
		t.Fatalf("timestamp should be outside window")
	}
}

func TestWindowFromPeriodCompat(t *testing.T) {
	now := time.Date(2024, 5, 10, 10, 0, 0, 0, time.UTC)
	start, end, err := WindowFromPeriod("7d", now, time.UTC)
	if err != nil {
		t.Fatalf("window from period: %v", err)
	}
	if !end.Equal(now) {
		t.Fatalf("unexpected end %v", end)
	}
	if !start.Equal(now.Add(-7 * 24 * time.Hour)) {
		t.Fatalf("unexpected start %v", start)
	}
}

func TestNewWindowInvalid(t *testing.T) {
	if _, err := NewWindow("bad", time.Now(), time.UTC); !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("expected ErrInvalidPeriod")
	}
}

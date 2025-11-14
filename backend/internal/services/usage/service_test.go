package usage

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildUsagePointsFromDailyMap_FillsMissingDaysAndFormatsTimezone(t *testing.T) {
	start := time.Date(2025, time.January, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, time.January, 3, 15, 0, 0, 0, time.UTC)

	daily := map[int64]dailyAggregate{}
	day1 := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	day3 := time.Date(2025, time.January, 3, 0, 0, 0, 0, time.UTC).Unix()

	daily[day1] = dailyAggregate{Requests: 10, Tokens: 100, CostCents: 25, CostUsdMicros: 2_500_000}
	daily[day3] = dailyAggregate{Requests: 5, Tokens: 60, CostCents: 15, CostUsdMicros: 1_750_000}

	points := buildUsagePointsFromDailyMap(start, end, time.UTC, daily)

	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}

	tests := []struct {
		index         int
		wantDate      string
		wantRequests  int64
		wantTokens    int64
		wantCostCents int64
		wantCostUSD   float64
	}{
		{0, "2025-01-01T00:00:00Z", 10, 100, 25, 2.5},
		{1, "2025-01-02T00:00:00Z", 0, 0, 0, 0},
		{2, "2025-01-03T00:00:00Z", 5, 60, 15, 1.75},
	}

	for _, tt := range tests {
		point := points[tt.index]
		if point.Date != tt.wantDate {
			t.Errorf("index %d: want date %s, got %s", tt.index, tt.wantDate, point.Date)
		}
		if point.Requests != tt.wantRequests {
			t.Errorf("index %d: want requests %d, got %d", tt.index, tt.wantRequests, point.Requests)
		}
		if point.Tokens != tt.wantTokens {
			t.Errorf("index %d: want tokens %d, got %d", tt.index, tt.wantTokens, point.Tokens)
		}
		if point.CostCents != tt.wantCostCents {
			t.Errorf("index %d: want cost cents %d, got %d", tt.index, tt.wantCostCents, point.CostCents)
		}
		if point.CostUSD != tt.wantCostUSD {
			t.Errorf("index %d: want cost usd %.2f, got %.2f", tt.index, tt.wantCostUSD, point.CostUSD)
		}
	}
}

func TestBuildUsagePointsFromDailyMap_TimezoneNormalization(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	start := time.Date(2025, time.March, 8, 8, 0, 0, 0, loc)
	end := time.Date(2025, time.March, 9, 22, 0, 0, 0, loc)

	daily := map[int64]dailyAggregate{}
	daily[time.Date(2025, time.March, 8, 0, 0, 0, 0, loc).Unix()] = dailyAggregate{Requests: 1}
	daily[time.Date(2025, time.March, 9, 0, 0, 0, 0, loc).Unix()] = dailyAggregate{Requests: 2}

	points := buildUsagePointsFromDailyMap(start, end, loc, daily)

	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}

	if !strings.HasSuffix(points[0].Date, "-05:00") && !strings.HasSuffix(points[0].Date, "-04:00") {
		t.Errorf("expected RFC3339 date with timezone offset, got %s", points[0].Date)
	}
}

func TestDedupUUIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	duplicates := []uuid.UUID{id1, uuid.Nil, id1, id2, id2}
	result := dedupUUIDs(duplicates)
	if len(result) != 2 {
		t.Fatalf("expected 2 uuids, got %d", len(result))
	}
	if result[0] != id1 || result[1] != id2 {
		t.Fatalf("unexpected UUID order: %v", result)
	}
}

func TestDedupStrings(t *testing.T) {
	values := []string{"  alpha  ", "beta", "ALPHA", "", "beta"}
	result := dedupStrings(values)
	expected := []string{"alpha", "beta", "ALPHA"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(result))
	}
	for i, val := range expected {
		if result[i] != val {
			t.Fatalf("index %d: want %s, got %s", i, val, result[i])
		}
	}
}

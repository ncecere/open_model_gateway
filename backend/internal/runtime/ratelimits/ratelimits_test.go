package ratelimits

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

func TestFromRecord_OverridesDefaults(t *testing.T) {
	base := config.RateLimitConfig{
		DefaultRequestsPerMinute:      1000,
		DefaultTokensPerMinute:        100000,
		DefaultParallelRequestsKey:    10,
		DefaultParallelRequestsTenant: 100,
	}
	record := db.RateLimitDefault{
		RequestsPerMinute:      2000,
		TokensPerMinute:        200000,
		ParallelRequestsKey:    20,
		ParallelRequestsTenant: 200,
	}
	got := FromRecord(base, record)
	if got.DefaultRequestsPerMinute != 2000 {
		t.Fatalf("RPM: expected 2000, got %d", got.DefaultRequestsPerMinute)
	}
	if got.DefaultTokensPerMinute != 200000 {
		t.Fatalf("TPM: expected 200000, got %d", got.DefaultTokensPerMinute)
	}
	if got.DefaultParallelRequestsKey != 20 {
		t.Fatalf("ParallelKey: expected 20, got %d", got.DefaultParallelRequestsKey)
	}
	if got.DefaultParallelRequestsTenant != 200 {
		t.Fatalf("ParallelTenant: expected 200, got %d", got.DefaultParallelRequestsTenant)
	}
}

func TestFromRecord_NegativeValuesPreserveDefaults(t *testing.T) {
	base := config.RateLimitConfig{
		DefaultRequestsPerMinute:      1000,
		DefaultTokensPerMinute:        100000,
		DefaultParallelRequestsKey:    10,
		DefaultParallelRequestsTenant: 100,
	}
	record := db.RateLimitDefault{
		RequestsPerMinute:      -1,
		TokensPerMinute:        -1,
		ParallelRequestsKey:    -1,
		ParallelRequestsTenant: -1,
	}
	got := FromRecord(base, record)
	// Negative values still overwrite because the condition is >= 0
	// The implementation uses >= 0 so -1 does NOT overwrite
	if got.DefaultRequestsPerMinute != 1000 {
		t.Fatalf("expected base preserved at 1000 for negative, got %d", got.DefaultRequestsPerMinute)
	}
}

func TestFromRecord_ZeroValuesOverride(t *testing.T) {
	base := config.RateLimitConfig{
		DefaultRequestsPerMinute: 1000,
	}
	record := db.RateLimitDefault{
		RequestsPerMinute:      0,
		TokensPerMinute:        0,
		ParallelRequestsKey:    0,
		ParallelRequestsTenant: 0,
	}
	got := FromRecord(base, record)
	// Zero is >= 0, so it overwrites
	if got.DefaultRequestsPerMinute != 0 {
		t.Fatalf("expected 0 override, got %d", got.DefaultRequestsPerMinute)
	}
}

func TestLoadDefaults_NilInputs(t *testing.T) {
	// Should not panic with nil inputs
	if err := LoadDefaults(nil, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

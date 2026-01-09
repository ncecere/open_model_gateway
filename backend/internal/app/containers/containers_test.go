package containers

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

func TestNewDataContainer(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	dc := NewDataContainer(nil, redisClient)
	if dc == nil {
		t.Fatal("expected non-nil DataContainer")
	}
	if dc.Redis != redisClient {
		t.Error("expected Redis client to be set")
	}
	if dc.DBPool != nil {
		t.Error("expected nil DBPool when not provided")
	}
}

func TestNewRateLimitContainer(t *testing.T) {
	defaultKeyLimit := limits.LimitConfig{
		RequestsPerMinute: 100,
		TokensPerMinute:   10000,
		ParallelRequests:  10,
	}
	defaultTenantLimit := limits.LimitConfig{
		RequestsPerMinute: 1000,
		TokensPerMinute:   100000,
		ParallelRequests:  50,
	}

	rc := NewRateLimitContainer(nil, nil, nil, defaultKeyLimit, defaultTenantLimit)
	if rc == nil {
		t.Fatal("expected non-nil RateLimitContainer")
	}
	if rc.DefaultKeyLimit != defaultKeyLimit {
		t.Error("expected DefaultKeyLimit to be set")
	}
	if rc.DefaultTenantLimit != defaultTenantLimit {
		t.Error("expected DefaultTenantLimit to be set")
	}
	if rc.KeyRateLimits == nil {
		t.Error("expected KeyRateLimits to be initialized")
	}
	if rc.TenantRateLimits == nil {
		t.Error("expected TenantRateLimits to be initialized")
	}
}

func TestRateLimitContainer_UpdateTenantRateLimit(t *testing.T) {
	rc := NewRateLimitContainer(nil, nil, nil, limits.LimitConfig{}, limits.LimitConfig{})

	tenantID := uuid.New()
	cfg := &limits.LimitConfig{
		RequestsPerMinute: 500,
		TokensPerMinute:   50000,
		ParallelRequests:  25,
	}

	rc.UpdateTenantRateLimit(tenantID, cfg)

	keyLimit, tenantLimit := rc.EffectiveRateLimits("", tenantID)
	if tenantLimit.RequestsPerMinute != cfg.RequestsPerMinute {
		t.Errorf("expected tenant RequestsPerMinute=%d, got %d", cfg.RequestsPerMinute, tenantLimit.RequestsPerMinute)
	}
	_ = keyLimit
}

func TestRateLimitContainer_UpdateAPIKeyRateLimit(t *testing.T) {
	rc := NewRateLimitContainer(nil, nil, nil, limits.LimitConfig{}, limits.LimitConfig{})

	prefix := "sk_test"
	cfg := &limits.LimitConfig{
		RequestsPerMinute: 200,
		TokensPerMinute:   20000,
		ParallelRequests:  5,
	}

	rc.UpdateAPIKeyRateLimit(prefix, cfg)

	keyLimit, _ := rc.EffectiveRateLimits(prefix, uuid.Nil)
	if keyLimit.RequestsPerMinute != cfg.RequestsPerMinute {
		t.Errorf("expected key RequestsPerMinute=%d, got %d", cfg.RequestsPerMinute, keyLimit.RequestsPerMinute)
	}
}

func TestRoutingContainer_IsModelAllowed(t *testing.T) {
	rc := NewRoutingContainer(nil, nil, nil)

	tenantID := uuid.New()

	// Without any tenant models set, all models should be allowed
	if !rc.IsModelAllowed(tenantID, "gpt-4") {
		t.Error("expected model to be allowed when no restrictions set")
	}

	// Set allowed models for tenant
	rc.SetTenantModels(tenantID, []string{"gpt-3.5-turbo", "gpt-4"})

	if !rc.IsModelAllowed(tenantID, "gpt-4") {
		t.Error("expected gpt-4 to be allowed")
	}

	// Clear tenant models
	rc.ClearTenantModels(tenantID)

	// After clearing, all models should be allowed again
	if !rc.IsModelAllowed(tenantID, "gpt-4") {
		t.Error("expected model to be allowed after clearing restrictions")
	}
}

func TestTelemetryContainer_RecordProviderTelemetry(t *testing.T) {
	// Import providermetrics is needed for Sample type
	// Test that nil container doesn't panic
	var tc *TelemetryContainer
	tc.RecordProviderTelemetry(context.Background(), providermetrics.Sample{})

	// Test with empty container
	tc = &TelemetryContainer{}
	tc.RecordProviderTelemetry(context.Background(), providermetrics.Sample{})
}

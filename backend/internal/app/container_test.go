package app

import (
	"context"
	"errors"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

func TestEffectiveRateLimits_MergesOverrides(t *testing.T) {
	t.Helper()

	tenantID := uuid.New()
	container := &Container{
		KeyRateLimits: map[string]limits.LimitConfig{
			"tok-test": {
				RequestsPerMinute: 120,
			},
		},
		TenantRateLimits: map[uuid.UUID]limits.LimitConfig{
			tenantID: {
				TokensPerMinute: 50_000,
			},
		},
		DefaultKeyLimit: limits.LimitConfig{
			RequestsPerMinute: 60,
			TokensPerMinute:   10_000,
			ParallelRequests:  2,
		},
		DefaultTenantLimit: limits.LimitConfig{
			RequestsPerMinute: 100,
			TokensPerMinute:   20_000,
			ParallelRequests:  4,
		},
	}

	keyCfg, tenantCfg := container.EffectiveRateLimits("tok-test", tenantID)

	if keyCfg.RequestsPerMinute != 120 {
		t.Fatalf("expected key RPM override applied, got %d", keyCfg.RequestsPerMinute)
	}
	if keyCfg.TokensPerMinute != 10_000 || keyCfg.ParallelRequests != 2 {
		t.Fatalf("expected key defaults to remain for unset fields, got %+v", keyCfg)
	}

	if tenantCfg.TokensPerMinute != 50_000 {
		t.Fatalf("expected tenant TPM override applied, got %d", tenantCfg.TokensPerMinute)
	}
	if tenantCfg.RequestsPerMinute != 100 || tenantCfg.ParallelRequests != 4 {
		t.Fatalf("expected tenant defaults persisted for non-overridden fields, got %+v", tenantCfg)
	}
}

func TestAcquireRateLimits_RespectsTenantParallelOverrides(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	tenantID := uuid.New()
	container := &Container{
		RateLimiter:        limits.NewRateLimiter(client),
		KeyRateLimits:      map[string]limits.LimitConfig{},
		TenantRateLimits:   map[uuid.UUID]limits.LimitConfig{},
		DefaultKeyLimit:    limits.LimitConfig{},
		DefaultTenantLimit: limits.LimitConfig{ParallelRequests: 1},
	}

	container.UpdateTenantRateLimit(tenantID, &limits.LimitConfig{
		ParallelRequests: 1,
	})

	ctx := requestctx.WithContext(context.Background(), &requestctx.Context{
		TenantID:     tenantID,
		APIKeyPrefix: "tok-test",
	})

	_, _, _, _, release, err := container.AcquireRateLimits(ctx, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("first AcquireRateLimits returned error: %v", err)
	}

	if _, _, _, _, _, err := container.AcquireRateLimits(ctx, "gpt-4o-mini"); !errors.Is(err, limits.ErrLimitExceeded) {
		t.Fatalf("expected ErrLimitExceeded on concurrent tenant request, got %v", err)
	}

	release()

	_, _, _, _, releaseAgain, err := container.AcquireRateLimits(ctx, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("expected acquire to succeed after release, got %v", err)
	}
	releaseAgain()
}

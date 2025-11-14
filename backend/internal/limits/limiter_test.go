package limits

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestLimiter(t *testing.T) (*RateLimiter, func()) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	limiter := NewRateLimiter(client)
	cleanup := func() {
		client.Close()
		server.Close()
	}
	return limiter, cleanup
}

func TestRateLimiterAllowEnforcesParallel(t *testing.T) {
	limiter, cleanup := newTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	cfg := LimitConfig{ParallelRequests: 1}
	key := "parallel:test"

	if err := limiter.Allow(ctx, key, cfg); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}
	if err := limiter.Allow(ctx, key, cfg); err != ErrLimitExceeded {
		t.Fatalf("expected parallel limit error, got %v", err)
	}
	limiter.Release(ctx, key, cfg)
	if err := limiter.Allow(ctx, key, cfg); err != nil {
		t.Fatalf("request after release should pass: %v", err)
	}
}

func TestRateLimiterAllowEnforcesRPM(t *testing.T) {
	limiter, cleanup := newTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	cfg := LimitConfig{RequestsPerMinute: 2}
	key := "rpm:test"

	if err := limiter.Allow(ctx, key, cfg); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}
	if err := limiter.Allow(ctx, key, cfg); err != nil {
		t.Fatalf("second request should pass: %v", err)
	}
	if err := limiter.Allow(ctx, key, cfg); err != ErrLimitExceeded {
		t.Fatalf("expected rpm limit error, got %v", err)
	}
}

func TestTokenAllowanceRollsBackOnFailure(t *testing.T) {
	limiter, cleanup := newTestLimiter(t)
	defer cleanup()

	ctx := context.Background()
	cfg := LimitConfig{TokensPerMinute: 10}
	key := "tenant:tokens"

	if err := limiter.TokenAllowance(ctx, key, 6, cfg); err != nil {
		t.Fatalf("first token allowance should pass: %v", err)
	}
	if err := limiter.TokenAllowance(ctx, key, 6, cfg); err != ErrLimitExceeded {
		t.Fatalf("expected token limit error, got %v", err)
	}

	// Ensure the rollback removed the rejected increment.
	redisKey := currentMinuteKey("tpm:%s", key)
	used, err := limiter.client.Get(ctx, redisKey).Int()
	if err != nil {
		t.Fatalf("get redis value: %v", err)
	}
	if used != 6 {
		t.Fatalf("expected usage to stay at 6 after rollback, got %d", used)
	}
}

func currentMinuteKey(prefixFmt, key string) string {
	now := time.Now().UTC().Unix() / 60
	return fmt.Sprintf(prefixFmt+":%d", key, now)
}

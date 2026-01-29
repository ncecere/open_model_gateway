package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
)

// --- retryDelay tests ---

func TestRetryDelay_UsesRetryAfterWhenPresent(t *testing.T) {
	err := apperror.RateLimitedWithRetryAfter("test", "slow down", 5*time.Second)
	got := retryDelay(err, 500*time.Millisecond, 2.0, 1)
	if got != 5*time.Second {
		t.Fatalf("expected 5s from Retry-After, got %v", got)
	}
}

func TestRetryDelay_FallsBackToExponentialBackoff(t *testing.T) {
	err := apperror.ServiceUnavailable("test", "down")
	// attempt=1 → baseDelay
	got := retryDelay(err, 500*time.Millisecond, 2.0, 1)
	if got != 500*time.Millisecond {
		t.Fatalf("attempt 1: expected 500ms, got %v", got)
	}
	// attempt=2 → baseDelay * multiplier
	got = retryDelay(err, 500*time.Millisecond, 2.0, 2)
	if got != 1*time.Second {
		t.Fatalf("attempt 2: expected 1s, got %v", got)
	}
	// attempt=3 → baseDelay * multiplier^2
	got = retryDelay(err, 500*time.Millisecond, 2.0, 3)
	if got != 2*time.Second {
		t.Fatalf("attempt 3: expected 2s, got %v", got)
	}
}

func TestRetryDelay_NilError(t *testing.T) {
	got := retryDelay(nil, 500*time.Millisecond, 2.0, 1)
	if got != 500*time.Millisecond {
		t.Fatalf("nil error: expected 500ms fallback, got %v", got)
	}
}

// --- waitForRetry tests ---

func TestWaitForRetry_ZeroDelay(t *testing.T) {
	if err := waitForRetry(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForRetry_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := waitForRetry(ctx, 10*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWaitForRetry_ShortDelay(t *testing.T) {
	start := time.Now()
	if err := waitForRetry(context.Background(), 10*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	// Should wait at least 10ms but no more than 50ms (jitter adds up to 25%)
	if elapsed < 10*time.Millisecond {
		t.Fatalf("waited too little: %v", elapsed)
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("waited too long: %v", elapsed)
	}
}

// --- normalizedRetry tests ---

func TestNormalizedRetry_Defaults(t *testing.T) {
	route := providers.Route{}
	cfg := normalizedRetry(route)
	if cfg.MaxAttempts != 1 {
		t.Fatalf("expected MaxAttempts=1, got %d", cfg.MaxAttempts)
	}
	if cfg.BackoffMultiplier != 1 {
		t.Fatalf("expected BackoffMultiplier=1, got %f", cfg.BackoffMultiplier)
	}
}

func TestNormalizedRetry_PreservesConfigured(t *testing.T) {
	route := providers.Route{
		Retry: providers.RetryConfig{
			MaxAttempts:       3,
			InitialBackoff:    500 * time.Millisecond,
			BackoffMultiplier: 2.0,
		},
	}
	cfg := normalizedRetry(route)
	if cfg.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts=3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialBackoff != 500*time.Millisecond {
		t.Fatalf("expected InitialBackoff=500ms, got %v", cfg.InitialBackoff)
	}
}

// --- shouldRetryProvider tests ---

func TestShouldRetryProvider_NilError(t *testing.T) {
	if shouldRetryProvider(nil) {
		t.Fatal("nil error should not be retryable")
	}
}

func TestShouldRetryProvider_Cancelled(t *testing.T) {
	if shouldRetryProvider(context.Canceled) {
		t.Fatal("cancelled context should not be retryable")
	}
	if shouldRetryProvider(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded should not be retryable")
	}
}

func TestShouldRetryProvider_ImageUnsupported(t *testing.T) {
	if shouldRetryProvider(models.ErrImageOperationUnsupported) {
		t.Fatal("image unsupported should not be retryable")
	}
}

func TestShouldRetryProvider_RateLimit(t *testing.T) {
	err := apperror.RateLimited("test", "slow down")
	if !shouldRetryProvider(err) {
		t.Fatal("429 should be retryable")
	}
}

func TestShouldRetryProvider_ServerError(t *testing.T) {
	err := apperror.ServiceUnavailable("test", "down")
	if !shouldRetryProvider(err) {
		t.Fatal("503 should be retryable")
	}
}

func TestShouldRetryProvider_BadRequest(t *testing.T) {
	err := apperror.BadRequest("test", "invalid")
	if shouldRetryProvider(err) {
		t.Fatal("400 should not be retryable")
	}
}

func TestShouldRetryProvider_GenericError(t *testing.T) {
	if !shouldRetryProvider(errors.New("network timeout")) {
		t.Fatal("generic errors should be retryable")
	}
}

// --- retryReason tests ---

func TestRetryReason_Nil(t *testing.T) {
	if got := retryReason(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestRetryReason_ImageUnsupported(t *testing.T) {
	if got := retryReason(models.ErrImageOperationUnsupported); got != "unsupported" {
		t.Fatalf("expected unsupported, got %q", got)
	}
}

func TestRetryReason_LimitExceeded(t *testing.T) {
	if got := retryReason(limits.ErrLimitExceeded); got != "limit_exceeded" {
		t.Fatalf("expected limit_exceeded, got %q", got)
	}
}

func TestRetryReason_AppError(t *testing.T) {
	err := apperror.RateLimited("test", "slow")
	got := retryReason(err)
	if got != "http_429" {
		t.Fatalf("expected http_429, got %q", got)
	}
}

func TestRetryReason_GenericError(t *testing.T) {
	if got := retryReason(errors.New("boom")); got != "provider_error" {
		t.Fatalf("expected provider_error, got %q", got)
	}
}

// --- NewAPIError / AsAPIError tests ---

func TestNewAPIError_MapsToBadRequest(t *testing.T) {
	err := NewAPIError(fiber.StatusBadRequest, "bad input")
	status, msg, ok := AsAPIError(err)
	if !ok {
		t.Fatal("expected AsAPIError to return true")
	}
	if status != 400 {
		t.Fatalf("expected 400, got %d", status)
	}
	if msg != "bad input" {
		t.Fatalf("expected bad input, got %q", msg)
	}
}

func TestNewAPIError_MapsToRateLimited(t *testing.T) {
	err := NewAPIError(fiber.StatusTooManyRequests, "slow down")
	status, _, ok := AsAPIError(err)
	if !ok || status != 429 {
		t.Fatalf("expected 429, got %d (ok=%v)", status, ok)
	}
}

func TestAsAPIError_Nil(t *testing.T) {
	_, _, ok := AsAPIError(nil)
	if ok {
		t.Fatal("nil error should return false")
	}
}

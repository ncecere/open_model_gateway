package retryafter

import (
	"net/http"
	"testing"
	"time"
)

func TestParse_DeltaSeconds(t *testing.T) {
	d := Parse("30")
	if d != 30*time.Second {
		t.Errorf("expected 30s, got %v", d)
	}
}

func TestParse_FloatSeconds(t *testing.T) {
	d := Parse("1.5")
	if d != 1500*time.Millisecond {
		t.Errorf("expected 1.5s, got %v", d)
	}
}

func TestParse_Empty(t *testing.T) {
	d := Parse("")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParse_InvalidString(t *testing.T) {
	d := Parse("not-a-number")
	// Should not panic, might parse as HTTP-date (will fail) or return 0
	if d < 0 {
		t.Errorf("expected >= 0, got %v", d)
	}
}

func TestFromResponse_429WithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"5"}},
	}
	d := FromResponse(resp)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestFromResponse_NilResponse(t *testing.T) {
	d := FromResponse(nil)
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestRateLimitError_WithRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"10"}},
	}
	err := RateLimitError("test", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.RetryAfter != 10*time.Second {
		t.Errorf("expected RetryAfter 10s, got %v", err.RetryAfter)
	}
}

func TestRateLimitError_WithoutRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
	}
	err := RateLimitError("test", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.RetryAfter != 0 {
		t.Errorf("expected RetryAfter 0, got %v", err.RetryAfter)
	}
}

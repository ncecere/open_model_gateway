// Package retryafter provides utilities for parsing Retry-After headers
// from HTTP responses and converting them into apperror types.
package retryafter

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/apperror"
)

// FromResponse extracts a Retry-After duration from an HTTP response.
// Supports both delta-seconds and HTTP-date formats per RFC 7231 §7.1.3.
// Returns 0 if the header is absent or unparseable.
func FromResponse(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	return Parse(resp.Header.Get("Retry-After"))
}

// Parse interprets a Retry-After header value.
// Supports delta-seconds (e.g. "30") and HTTP-date formats.
func Parse(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	// Try delta-seconds first (most common)
	if secs, err := strconv.ParseFloat(value, 64); err == nil && secs > 0 {
		return time.Duration(secs * float64(time.Second))
	}

	// Try HTTP-date format
	if t, err := http.ParseTime(value); err == nil {
		delta := time.Until(t)
		if delta > 0 {
			return delta
		}
	}

	return 0
}

// RateLimitError creates an apperror with Retry-After from the response.
func RateLimitError(op string, resp *http.Response) *apperror.Error {
	ra := FromResponse(resp)
	if ra > 0 {
		return apperror.RateLimitedWithRetryAfter(op, "rate limited by provider", ra)
	}
	return apperror.RateLimited(op, "rate limited by provider")
}

// OverloadedError creates an apperror for 529/503 overloaded responses.
func OverloadedError(op string, resp *http.Response) *apperror.Error {
	ra := FromResponse(resp)
	if ra > 0 {
		return apperror.RateLimitedWithRetryAfter(op, "provider overloaded", ra)
	}
	return apperror.ServiceUnavailable(op, "provider overloaded")
}

package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver/httputil"
)

// AuthRateLimit returns a Fiber middleware that enforces sliding-window rate
// limiting on authentication endpoints to mitigate brute-force attacks.
// It uses an in-memory store keyed by client IP.
func AuthRateLimit(cfg config.AuthRateLimitConfig) fiber.Handler {
	if !cfg.Enabled || cfg.MaxAttempts <= 0 {
		return func(c *fiber.Ctx) error { return c.Next() }
	}

	window := cfg.Window
	if window <= 0 {
		window = 15 * time.Minute
	}
	maxAttempts := cfg.MaxAttempts

	store := &rateLimitStore{
		entries: make(map[string]*rateLimitEntry),
	}

	// Background cleanup every window period.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			store.cleanup(window)
		}
	}()

	return func(c *fiber.Ctx) error {
		key := c.IP()
		if !store.allow(key, maxAttempts, window) {
			c.Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			return httputil.WriteError(c, fiber.StatusTooManyRequests, "too many authentication attempts, try again later")
		}
		return c.Next()
	}
}

type rateLimitEntry struct {
	timestamps []time.Time
}

type rateLimitStore struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
}

func (s *rateLimitStore) allow(key string, max int, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	entry, ok := s.entries[key]
	if !ok {
		entry = &rateLimitEntry{}
		s.entries[key] = entry
	}

	// Prune expired timestamps.
	valid := entry.timestamps[:0]
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	entry.timestamps = valid

	if len(entry.timestamps) >= max {
		return false
	}

	entry.timestamps = append(entry.timestamps, now)
	return true
}

func (s *rateLimitStore) cleanup(window time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-window)
	for key, entry := range s.entries {
		valid := entry.timestamps[:0]
		for _, ts := range entry.timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		entry.timestamps = valid
		if len(entry.timestamps) == 0 {
			delete(s.entries, key)
		}
	}
}

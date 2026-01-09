// Package middleware provides HTTP middleware for the Open Model Gateway.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoggerConfig holds configuration for the request logger middleware.
type LoggerConfig struct {
	// Logger is the slog.Logger to use. Defaults to slog.Default().
	Logger *slog.Logger

	// SkipPaths contains paths to skip logging.
	SkipPaths []string
}

// Logger returns a Fiber middleware that logs requests using slog.
// It logs request method, path, status code, latency, and client IP.
func Logger(cfg ...LoggerConfig) fiber.Handler {
	var config LoggerConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip logging for configured paths
		if skipMap[path] {
			return c.Next()
		}

		start := time.Now()
		requestID := c.Locals("requestid")

		// Process request
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()

		// Build log attributes
		attrs := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("ip", c.IP()),
		}

		if requestID != nil {
			if rid, ok := requestID.(string); ok && rid != "" {
				attrs = append(attrs, slog.String("request_id", rid))
			}
		}

		// Add user agent for non-health-check requests
		if path != "/healthz" && path != "/metrics" {
			if ua := c.Get("User-Agent"); ua != "" {
				attrs = append(attrs, slog.String("user_agent", ua))
			}
		}

		// Add response size
		attrs = append(attrs, slog.Int("bytes", len(c.Response().Body())))

		// Log at appropriate level based on status
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}

		switch {
		case status >= 500:
			config.Logger.Error("request", args...)
		case status >= 400:
			config.Logger.Warn("request", args...)
		default:
			config.Logger.Info("request", args...)
		}

		return err
	}
}

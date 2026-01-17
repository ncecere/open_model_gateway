// Package middleware provides HTTP middleware for the Open Model Gateway.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/ncecere/open_model_gateway/backend/internal/logging"
)

// WideEventConfig holds configuration for the wide event middleware.
type WideEventConfig struct {
	// Logger is the slog.Logger to use. Defaults to slog.Default().
	Logger *slog.Logger

	// ServiceName is the service name to include in log events.
	ServiceName string

	// ServiceVersion is the service version to include in log events.
	ServiceVersion string

	// SkipPaths contains paths to skip logging.
	SkipPaths []string

	// IncludeUserAgent includes the User-Agent header in log events.
	IncludeUserAgent bool
}

// WideEvent returns a Fiber middleware that emits wide event logs.
// It initializes a WideEvent at request start, attaches it to context for
// enrichment by handlers, and emits a single comprehensive log at request end.
func WideEvent(cfg WideEventConfig) fiber.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "open-model-gateway"
	}

	skipMap := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip logging for configured paths
		if skipMap[path] {
			return c.Next()
		}

		start := time.Now()

		// Initialize wide event
		event := &logging.WideEvent{
			Timestamp: start,
			Method:    c.Method(),
			Path:      path,
			IP:        c.IP(),
			Service:   cfg.ServiceName,
			Version:   cfg.ServiceVersion,
		}

		// Get request ID from requestid middleware
		if rid := c.Locals("requestid"); rid != nil {
			if id, ok := rid.(string); ok {
				event.RequestID = id
			}
		}

		// Get trace ID from OTEL context if available
		if span := trace.SpanFromContext(c.UserContext()); span.SpanContext().IsValid() {
			event.TraceID = span.SpanContext().TraceID().String()
		}

		// Include user agent if configured (skip for health/metrics endpoints)
		if cfg.IncludeUserAgent && path != "/healthz" && path != "/metrics" {
			event.UserAgent = c.Get("User-Agent")
		}

		// Attach event to context for enrichment by handlers
		ctx := logging.WithWideEvent(c.UserContext(), event)
		c.SetUserContext(ctx)

		// Process request
		err := c.Next()

		// Check if this is a streaming request that handles its own logging
		if event.ShouldSkipMiddlewareLog() {
			return err
		}

		// Finalize event with response data
		statusCode := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				statusCode = fiberErr.Code
				event.SetError("fiber.Error", "", fiberErr.Message, false)
			} else {
				statusCode = fiber.StatusInternalServerError
				event.SetError("error", "", err.Error(), false)
			}
		}

		// Get route pattern if available
		if r := c.Route(); r != nil {
			event.Route = r.Path
		}

		// Finalize and emit
		event.Finalize(
			time.Since(start).Milliseconds(),
			statusCode,
			len(c.Response().Body()),
		)
		event.Emit(cfg.Logger)

		return err
	}
}

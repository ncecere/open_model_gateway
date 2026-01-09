// Package logging provides structured logging utilities for the Open Model Gateway.
// It wraps Go's slog package with common patterns and field helpers.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

// Config holds configuration for the logger.
type Config struct {
	// Level is the minimum log level (debug, info, warn, error).
	Level string

	// Format is the output format (json, text).
	Format string

	// Output is the output writer (defaults to os.Stdout).
	Output io.Writer

	// AddSource adds source file information to log entries.
	AddSource bool
}

// Logger wraps slog.Logger with additional context methods.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger with the given configuration.
func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default returns a default logger with text format at info level.
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "text",
	})
}

// parseLevel converts a string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Context key types for request-scoped values.
type contextKey string

const (
	loggerKey    contextKey = "logger"
	requestIDKey contextKey = "request_id"
	tenantIDKey  contextKey = "tenant_id"
	userIDKey    contextKey = "user_id"
	apiKeyIDKey  contextKey = "api_key_id"
)

// WithContext returns a new context with the logger attached.
func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context, or returns default logger.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return Default()
}

// WithRequestID returns a new context with the request ID attached.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithTenantID returns a new context with the tenant ID attached.
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithUserID returns a new context with the user ID attached.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithAPIKeyID returns a new context with the API key ID attached.
func WithAPIKeyID(ctx context.Context, apiKeyID uuid.UUID) context.Context {
	return context.WithValue(ctx, apiKeyIDKey, apiKeyID)
}

// ContextLogger returns a logger with context values as fields.
func ContextLogger(ctx context.Context) *Logger {
	logger := FromContext(ctx)

	var attrs []slog.Attr

	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	if tenantID, ok := ctx.Value(tenantIDKey).(uuid.UUID); ok && tenantID != uuid.Nil {
		attrs = append(attrs, slog.String("tenant_id", tenantID.String()))
	}

	if userID, ok := ctx.Value(userIDKey).(uuid.UUID); ok && userID != uuid.Nil {
		attrs = append(attrs, slog.String("user_id", userID.String()))
	}

	if apiKeyID, ok := ctx.Value(apiKeyIDKey).(uuid.UUID); ok && apiKeyID != uuid.Nil {
		attrs = append(attrs, slog.String("api_key_id", apiKeyID.String()))
	}

	if len(attrs) > 0 {
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}
		return &Logger{Logger: logger.Logger.With(args...)}
	}

	return logger
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithGroup returns a new Logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

// Common field helpers for structured logging.

// TenantID returns a slog attribute for tenant ID.
func TenantID(id uuid.UUID) slog.Attr {
	return slog.String("tenant_id", id.String())
}

// UserID returns a slog attribute for user ID.
func UserID(id uuid.UUID) slog.Attr {
	return slog.String("user_id", id.String())
}

// APIKeyID returns a slog attribute for API key ID.
func APIKeyID(id uuid.UUID) slog.Attr {
	return slog.String("api_key_id", id.String())
}

// RequestID returns a slog attribute for request ID.
func RequestID(id string) slog.Attr {
	return slog.String("request_id", id)
}

// Model returns a slog attribute for model name/alias.
func Model(name string) slog.Attr {
	return slog.String("model", name)
}

// Provider returns a slog attribute for provider name.
func Provider(name string) slog.Attr {
	return slog.String("provider", name)
}

// Op returns a slog attribute for operation name.
func Op(name string) slog.Attr {
	return slog.String("op", name)
}

// Error returns a slog attribute for an error.
func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String("error", err.Error())
}

// Duration returns a slog attribute for a duration in milliseconds.
func DurationMs(ms int64) slog.Attr {
	return slog.Int64("duration_ms", ms)
}

// StatusCode returns a slog attribute for HTTP status code.
func StatusCode(code int) slog.Attr {
	return slog.Int("status_code", code)
}

// TokensInput returns a slog attribute for input token count.
func TokensInput(count int64) slog.Attr {
	return slog.Int64("tokens_input", count)
}

// TokensOutput returns a slog attribute for output token count.
func TokensOutput(count int64) slog.Attr {
	return slog.Int64("tokens_output", count)
}

// Path returns a slog attribute for request path.
func Path(path string) slog.Attr {
	return slog.String("path", path)
}

// Method returns a slog attribute for HTTP method.
func Method(method string) slog.Attr {
	return slog.String("method", method)
}

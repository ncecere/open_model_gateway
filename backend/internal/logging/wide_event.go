// Package logging provides wide event (canonical log line) support for comprehensive request logging.
package logging

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// wideEventKey is the context key for storing wide events.
type wideEventKey struct{}

// WideEvent represents a comprehensive log event for a single request.
// It captures all relevant context for debugging and analytics in a single log line.
type WideEvent struct {
	mu sync.Mutex

	// Timing
	Timestamp  time.Time `json:"timestamp"`
	DurationMs int64     `json:"duration_ms"`

	// Request Identity
	RequestID      string `json:"request_id"`
	TraceID        string `json:"trace_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// HTTP
	Method     string `json:"method"`
	Path       string `json:"path"`
	Route      string `json:"route,omitempty"`
	StatusCode int    `json:"status_code"`
	BytesSent  int    `json:"bytes_sent"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent,omitempty"`

	// Service
	Service string `json:"service"`
	Version string `json:"version"`

	// Tenant/Auth Context (public API)
	TenantID     string `json:"tenant_id,omitempty"`
	APIKeyID     string `json:"api_key_id,omitempty"`
	APIKeyPrefix string `json:"api_key_prefix,omitempty"`

	// Admin Context (admin API)
	AdminUserID    string `json:"admin_user_id,omitempty"`
	AdminUserEmail string `json:"admin_user_email,omitempty"`
	AdminKeyID     string `json:"admin_key_id,omitempty"`
	TargetResource string `json:"target_resource,omitempty"`

	// User Context (user API)
	UserID    string `json:"user_id,omitempty"`
	UserEmail string `json:"user_email,omitempty"`

	// Model/Provider (for /v1/* routes)
	ModelAlias    string `json:"model_alias,omitempty"`
	Provider      string `json:"provider,omitempty"`
	ProviderModel string `json:"provider_model,omitempty"`
	Deployment    string `json:"deployment,omitempty"`
	Streaming     bool   `json:"streaming,omitempty"`

	// Provider Execution
	ProviderLatencyMs int64 `json:"provider_latency_ms,omitempty"`
	RetryCount        int   `json:"retry_count,omitempty"`
	RouteCount        int   `json:"route_count,omitempty"`

	// Usage Metrics
	TokensInput  int64 `json:"tokens_input,omitempty"`
	TokensOutput int64 `json:"tokens_output,omitempty"`
	TokensTotal  int64 `json:"tokens_total,omitempty"`
	CostMicroUSD int64 `json:"cost_micro_usd,omitempty"`
	ImageCount   int   `json:"image_count,omitempty"`

	// Budget/Rate Limit State
	BudgetLimitCents     int64 `json:"budget_limit_cents,omitempty"`
	BudgetUsedCents      int64 `json:"budget_used_cents,omitempty"`
	BudgetRemainingCents int64 `json:"budget_remaining_cents,omitempty"`
	BudgetExceeded       bool  `json:"budget_exceeded,omitempty"`
	RateLimited          bool  `json:"rate_limited,omitempty"`

	// Cache/Idempotency
	IdempotencyCacheHit bool `json:"idempotency_cache_hit,omitempty"`

	// Error Details
	ErrorType    string `json:"error_type,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Retriable    bool   `json:"retriable,omitempty"`

	// Outcome
	Outcome string `json:"outcome"`

	// Internal flags
	emitted           bool // prevent double-emit for streaming
	skipMiddlewareLog bool // streaming handlers emit their own log
}

// WithWideEvent attaches a wide event to the context for accumulation.
func WithWideEvent(ctx context.Context, event *WideEvent) context.Context {
	return context.WithValue(ctx, wideEventKey{}, event)
}

// WideEventFromContext retrieves the wide event from context.
func WideEventFromContext(ctx context.Context) (*WideEvent, bool) {
	if ctx == nil {
		return nil, false
	}
	event, ok := ctx.Value(wideEventKey{}).(*WideEvent)
	return event, ok
}

// EnrichFromRequestContext populates tenant/API key fields from the request context.
func (e *WideEvent) EnrichFromRequestContext(rc *requestctx.Context) {
	if rc == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	if rc.TenantID != uuid.Nil {
		e.TenantID = rc.TenantID.String()
	}
	if rc.APIKeyID != uuid.Nil {
		e.APIKeyID = rc.APIKeyID.String()
	}
	if rc.APIKeyPrefix != "" {
		e.APIKeyPrefix = rc.APIKeyPrefix
	}
	e.BudgetLimitCents = rc.BudgetLimitCents
}

// SetTargetResource sets the target resource for admin operations.
func (e *WideEvent) SetTargetResource(resourceType string, resourceID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.TargetResource = resourceType + ":" + resourceID
}

// SetAdminContext sets admin user context fields.
func (e *WideEvent) SetAdminContext(userID, email string, keyID *uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.AdminUserID = userID
	e.AdminUserEmail = email
	if keyID != nil && *keyID != uuid.Nil {
		e.AdminKeyID = keyID.String()
	}
}

// SetUserContext sets user context fields.
func (e *WideEvent) SetUserContext(userID, email string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.UserID = userID
	e.UserEmail = email
}

// SetModelContext sets model/provider fields.
func (e *WideEvent) SetModelContext(alias, provider, providerModel, deployment string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ModelAlias = alias
	e.Provider = provider
	e.ProviderModel = providerModel
	e.Deployment = deployment
}

// SetExecutionMetrics sets provider execution metrics.
func (e *WideEvent) SetExecutionMetrics(latencyMs int64, retryCount, routeCount int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ProviderLatencyMs = latencyMs
	e.RetryCount = retryCount
	e.RouteCount = routeCount
}

// SetUsageMetrics sets token and cost metrics.
func (e *WideEvent) SetUsageMetrics(tokensIn, tokensOut, tokensTotal, costMicroUSD int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.TokensInput = tokensIn
	e.TokensOutput = tokensOut
	e.TokensTotal = tokensTotal
	e.CostMicroUSD = costMicroUSD
}

// SetBudgetStatus sets budget-related fields.
func (e *WideEvent) SetBudgetStatus(usedCents, remainingCents int64, exceeded bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.BudgetUsedCents = usedCents
	e.BudgetRemainingCents = remainingCents
	e.BudgetExceeded = exceeded
}

// SetError sets error-related fields.
func (e *WideEvent) SetError(errType, errCode, errMsg string, retriable bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ErrorType = errType
	e.ErrorCode = errCode
	e.ErrorMessage = errMsg
	e.Retriable = retriable
}

// MarkStreaming marks this as a streaming request.
func (e *WideEvent) MarkStreaming() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Streaming = true
	e.skipMiddlewareLog = true
}

// ShouldSkipMiddlewareLog returns true if middleware should skip logging (streaming handles it).
func (e *WideEvent) ShouldSkipMiddlewareLog() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.skipMiddlewareLog
}

// Finalize sets the duration and determines the outcome.
func (e *WideEvent) Finalize(durationMs int64, statusCode, bytesSent int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.DurationMs = durationMs
	e.StatusCode = statusCode
	e.BytesSent = bytesSent
	e.Outcome = e.determineOutcome()
}

// determineOutcome determines the outcome string based on event state.
// Must be called with lock held.
func (e *WideEvent) determineOutcome() string {
	if e.RateLimited {
		return "rate_limited"
	}
	if e.BudgetExceeded {
		return "budget_exceeded"
	}
	if e.StatusCode >= 500 {
		return "server_error"
	}
	if e.StatusCode >= 400 {
		return "client_error"
	}
	return "success"
}

// ToSlogAttrs converts the wide event to slog attributes for emission.
func (e *WideEvent) ToSlogAttrs() []slog.Attr {
	e.mu.Lock()
	defer e.mu.Unlock()

	attrs := make([]slog.Attr, 0, 50)

	// Always include core fields
	attrs = append(attrs,
		slog.Time("timestamp", e.Timestamp),
		slog.Int64("duration_ms", e.DurationMs),
		slog.String("request_id", e.RequestID),
		slog.String("method", e.Method),
		slog.String("path", e.Path),
		slog.Int("status_code", e.StatusCode),
		slog.Int("bytes_sent", e.BytesSent),
		slog.String("ip", e.IP),
		slog.String("service", e.Service),
		slog.String("outcome", e.Outcome),
	)

	// Conditionally add optional fields (omit empty/zero values)
	if e.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", e.TraceID))
	}
	if e.Route != "" {
		attrs = append(attrs, slog.String("route", e.Route))
	}
	if e.Version != "" {
		attrs = append(attrs, slog.String("version", e.Version))
	}
	if e.UserAgent != "" {
		attrs = append(attrs, slog.String("user_agent", e.UserAgent))
	}
	if e.IdempotencyKey != "" {
		attrs = append(attrs, slog.String("idempotency_key", e.IdempotencyKey))
	}

	// Tenant/Auth
	if e.TenantID != "" {
		attrs = append(attrs, slog.String("tenant_id", e.TenantID))
	}
	if e.APIKeyID != "" {
		attrs = append(attrs, slog.String("api_key_id", e.APIKeyID))
	}
	if e.APIKeyPrefix != "" {
		attrs = append(attrs, slog.String("api_key_prefix", e.APIKeyPrefix))
	}

	// Admin context
	if e.AdminUserID != "" {
		attrs = append(attrs, slog.String("admin_user_id", e.AdminUserID))
	}
	if e.AdminUserEmail != "" {
		attrs = append(attrs, slog.String("admin_user_email", e.AdminUserEmail))
	}
	if e.AdminKeyID != "" {
		attrs = append(attrs, slog.String("admin_key_id", e.AdminKeyID))
	}
	if e.TargetResource != "" {
		attrs = append(attrs, slog.String("target_resource", e.TargetResource))
	}

	// User context
	if e.UserID != "" {
		attrs = append(attrs, slog.String("user_id", e.UserID))
	}
	if e.UserEmail != "" {
		attrs = append(attrs, slog.String("user_email", e.UserEmail))
	}

	// Model/Provider
	if e.ModelAlias != "" {
		attrs = append(attrs, slog.String("model_alias", e.ModelAlias))
	}
	if e.Provider != "" {
		attrs = append(attrs, slog.String("provider", e.Provider))
	}
	if e.ProviderModel != "" {
		attrs = append(attrs, slog.String("provider_model", e.ProviderModel))
	}
	if e.Deployment != "" {
		attrs = append(attrs, slog.String("deployment", e.Deployment))
	}
	if e.Streaming {
		attrs = append(attrs, slog.Bool("streaming", e.Streaming))
	}

	// Execution metrics
	if e.ProviderLatencyMs > 0 {
		attrs = append(attrs, slog.Int64("provider_latency_ms", e.ProviderLatencyMs))
	}
	if e.RetryCount > 0 {
		attrs = append(attrs, slog.Int("retry_count", e.RetryCount))
	}
	if e.RouteCount > 0 {
		attrs = append(attrs, slog.Int("route_count", e.RouteCount))
	}

	// Usage
	if e.TokensInput > 0 {
		attrs = append(attrs, slog.Int64("tokens_input", e.TokensInput))
	}
	if e.TokensOutput > 0 {
		attrs = append(attrs, slog.Int64("tokens_output", e.TokensOutput))
	}
	if e.TokensTotal > 0 {
		attrs = append(attrs, slog.Int64("tokens_total", e.TokensTotal))
	}
	if e.CostMicroUSD > 0 {
		attrs = append(attrs, slog.Int64("cost_micro_usd", e.CostMicroUSD))
	}
	if e.ImageCount > 0 {
		attrs = append(attrs, slog.Int("image_count", e.ImageCount))
	}

	// Budget/Rate limit
	if e.BudgetLimitCents > 0 {
		attrs = append(attrs, slog.Int64("budget_limit_cents", e.BudgetLimitCents))
	}
	if e.BudgetUsedCents > 0 {
		attrs = append(attrs, slog.Int64("budget_used_cents", e.BudgetUsedCents))
	}
	if e.BudgetRemainingCents > 0 {
		attrs = append(attrs, slog.Int64("budget_remaining_cents", e.BudgetRemainingCents))
	}
	if e.BudgetExceeded {
		attrs = append(attrs, slog.Bool("budget_exceeded", e.BudgetExceeded))
	}
	if e.RateLimited {
		attrs = append(attrs, slog.Bool("rate_limited", e.RateLimited))
	}

	// Cache
	if e.IdempotencyCacheHit {
		attrs = append(attrs, slog.Bool("idempotency_cache_hit", e.IdempotencyCacheHit))
	}

	// Error
	if e.ErrorType != "" {
		attrs = append(attrs, slog.String("error_type", e.ErrorType))
	}
	if e.ErrorCode != "" {
		attrs = append(attrs, slog.String("error_code", e.ErrorCode))
	}
	if e.ErrorMessage != "" {
		attrs = append(attrs, slog.String("error_message", e.ErrorMessage))
	}
	if e.Retriable {
		attrs = append(attrs, slog.Bool("retriable", e.Retriable))
	}

	return attrs
}

// Emit logs the wide event using the provided logger.
// Returns true if emitted, false if already emitted.
func (e *WideEvent) Emit(logger *slog.Logger) bool {
	e.mu.Lock()
	if e.emitted {
		e.mu.Unlock()
		return false
	}
	e.emitted = true
	e.mu.Unlock()

	attrs := e.ToSlogAttrs()
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}

	switch {
	case e.StatusCode >= 500:
		logger.Error("request", args...)
	case e.StatusCode >= 400:
		logger.Warn("request", args...)
	default:
		logger.Info("request", args...)
	}

	return true
}

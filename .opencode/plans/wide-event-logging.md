# Wide Event Logging Implementation Plan

## Overview

Implement "wide event" (canonical log line) logging pattern across the Open Model Gateway. Each HTTP request will emit a single, comprehensive JSON log line containing all relevant context for debugging and analytics.

## Current State

### Logging Infrastructure
- **Library**: Go's `slog` package (1.21+)
- **Config**: `logging.level`, `logging.format`, `logging.add_source`
- **Default format**: text (need to switch to JSON)

### Current HTTP Logger (`middleware/logger.go`)
Fields logged:
- `method`, `path`, `status`, `latency_ms`, `ip`, `request_id`, `user_agent`, `bytes`

### Available Context (not currently logged)
From `requestctx.Context`:
- `TenantID`, `APIKeyID`, `APIKeyPrefix`, `Scopes`
- `BudgetLimitCents`, `WarningThreshold`, `BudgetRefreshSchedule`
- `AlertsEnabled`, `HasBudgetOverride`

From executor results:
- `Provider`, `ProviderModel`, `Deployment`
- `Latency`, `RetryCount`
- `TokensInput`, `TokensOutput`, `CostMicroUSD`
- Error details

---

## Implementation Plan

### Phase 1: Core Infrastructure

#### 1.1 Extend Logging Config
**File**: `backend/internal/config/server.go`

```go
// LoggingConfig holds logging configuration.
type LoggingConfig struct {
    Level     string          `mapstructure:"level"`
    Format    string          `mapstructure:"format"`
    AddSource bool            `mapstructure:"add_source"`
    WideEvent WideEventConfig `mapstructure:"wide_event"`
}

// WideEventConfig holds wide event logging settings.
type WideEventConfig struct {
    Enabled          bool     `mapstructure:"enabled"`
    IncludeUserAgent bool     `mapstructure:"include_user_agent"`
    IncludeHeaders   []string `mapstructure:"include_headers"`
}
```

**File**: `backend/internal/config/config.go`
Add defaults:
```go
v.SetDefault("logging.wide_event.enabled", true)
v.SetDefault("logging.wide_event.include_user_agent", true)
v.SetDefault("logging.wide_event.include_headers", []string{})
```

#### 1.2 Create Wide Event Types
**File**: `backend/internal/logging/wide_event.go` (NEW)

```go
package logging

import (
    "log/slog"
    "time"
)

// WideEvent represents a comprehensive log event for a single request.
type WideEvent struct {
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
    TargetResource string `json:"target_resource,omitempty"` // e.g., "tenant:uuid"

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
    Outcome string `json:"outcome"` // "success", "error", "rate_limited", "budget_exceeded"
}

// ToSlogAttrs converts the wide event to slog attributes for emission.
func (e *WideEvent) ToSlogAttrs() []slog.Attr {
    attrs := make([]slog.Attr, 0, 40)
    
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
func (e *WideEvent) Emit(logger *slog.Logger) {
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
}
```

#### 1.3 Create Wide Event Context Key
**File**: `backend/internal/logging/wide_event.go` (add to same file)

```go
type wideEventKey struct{}

// WithWideEvent attaches a wide event to the context for accumulation.
func WithWideEvent(ctx context.Context, event *WideEvent) context.Context {
    return context.WithValue(ctx, wideEventKey{}, event)
}

// WideEventFromContext retrieves the wide event from context.
func WideEventFromContext(ctx context.Context) (*WideEvent, bool) {
    event, ok := ctx.Value(wideEventKey{}).(*WideEvent)
    return event, ok
}
```

---

### Phase 2: New Middleware

#### 2.1 Create Wide Event Middleware
**File**: `backend/internal/httpserver/middleware/wide_event.go` (NEW)

```go
package middleware

import (
    "log/slog"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel/trace"

    "github.com/ncecere/open_model_gateway/backend/internal/config"
    "github.com/ncecere/open_model_gateway/backend/internal/logging"
)

// WideEventConfig holds configuration for the wide event middleware.
type WideEventConfig struct {
    Logger           *slog.Logger
    ServiceName      string
    ServiceVersion   string
    SkipPaths        []string
    IncludeUserAgent bool
}

// WideEvent returns a Fiber middleware that emits wide event logs.
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

        // Get trace ID from OTEL context
        if span := trace.SpanFromContext(c.UserContext()); span.SpanContext().IsValid() {
            event.TraceID = span.SpanContext().TraceID().String()
        }

        // Include user agent if configured
        if cfg.IncludeUserAgent && path != "/healthz" && path != "/metrics" {
            event.UserAgent = c.Get("User-Agent")
        }

        // Attach event to context for enrichment by handlers
        ctx := logging.WithWideEvent(c.UserContext(), event)
        c.SetUserContext(ctx)

        // Process request
        err := c.Next()

        // Finalize event
        event.DurationMs = time.Since(start).Milliseconds()
        event.StatusCode = c.Response().StatusCode()
        event.BytesSent = len(c.Response().Body())

        if r := c.Route(); r != nil {
            event.Route = r.Path
        }

        // Handle fiber errors
        if err != nil {
            if fiberErr, ok := err.(*fiber.Error); ok {
                event.StatusCode = fiberErr.Code
                event.ErrorMessage = fiberErr.Message
            } else {
                event.StatusCode = fiber.StatusInternalServerError
                event.ErrorMessage = err.Error()
            }
        }

        // Determine outcome
        event.Outcome = determineOutcome(event)

        // Emit the wide event
        event.Emit(cfg.Logger)

        return err
    }
}

func determineOutcome(event *logging.WideEvent) string {
    if event.RateLimited {
        return "rate_limited"
    }
    if event.BudgetExceeded {
        return "budget_exceeded"
    }
    if event.StatusCode >= 400 {
        return "error"
    }
    return "success"
}
```

#### 2.2 Update Server to Use New Middleware
**File**: `backend/internal/httpserver/server.go`

Replace:
```go
app.Use(middleware.Logger(middleware.LoggerConfig{
    Logger:    container.Log(),
    SkipPaths: []string{"/healthz", "/metrics"},
}))
```

With:
```go
app.Use(middleware.WideEvent(middleware.WideEventConfig{
    Logger:           container.Log(),
    ServiceName:      "open-model-gateway",
    ServiceVersion:   container.Config.Version, // Add version to config
    SkipPaths:        []string{"/healthz", "/metrics"},
    IncludeUserAgent: container.Config.Logging.WideEvent.IncludeUserAgent,
}))
```

#### 2.3 Delete Old Logger Middleware
**File**: `backend/internal/httpserver/middleware/logger.go`
- Delete this file entirely

---

### Phase 3: Enrich from Request Context

#### 3.1 Add Helper to Enrich Wide Event from Request Context
**File**: `backend/internal/logging/wide_event.go` (add function)

```go
// EnrichFromRequestContext populates tenant/API key fields from the request context.
func (e *WideEvent) EnrichFromRequestContext(rc *requestctx.Context) {
    if rc == nil {
        return
    }
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
```

#### 3.2 Enrich in Public API Auth Middleware
**File**: `backend/internal/httpserver/public/auth.go`

After setting the request context, add:
```go
// Enrich wide event with tenant/key info
if event, ok := logging.WideEventFromContext(ctx); ok {
    event.EnrichFromRequestContext(rc)
}
```

#### 3.3 Enrich in Admin Auth Middleware
**File**: `backend/internal/httpserver/admin/admin_middleware.go`

After authenticating user, add:
```go
// Enrich wide event with admin user info
if event, ok := logging.WideEventFromContext(ctx); ok {
    event.AdminUserID = userID.String()
    event.AdminUserEmail = user.Email
    if keyID, ok := ctx.Value(adminContextKeyIDKey).(uuid.UUID); ok {
        event.AdminKeyID = keyID.String()
    }
}
```

---

### Phase 4: Enrich from Executor/Pipeline

#### 4.1 Add Execution Metrics to Results
**File**: `backend/internal/executor/executor.go`

Extend result types:
```go
// ExecutionMetrics captures provider execution details for logging.
type ExecutionMetrics struct {
    Provider       string
    ProviderModel  string
    Deployment     string
    LatencyMs      int64
    RetryCount     int
    RouteCount     int
    CostMicroUSD   int64
}

type ChatResult struct {
    Response     models.ChatResponse
    BudgetStatus usagepipeline.BudgetStatus
    Metrics      ExecutionMetrics // Add this
}
```

Update the Chat function to populate metrics:
```go
// In the success path, before returning:
return ChatResult{
    Response:     resp,
    BudgetStatus: budgetStatus,
    Metrics: ExecutionMetrics{
        Provider:      route.Provider,
        ProviderModel: route.Model,
        Deployment:    route.ResolveDeployment(),
        LatencyMs:     elapsed.Milliseconds(),
        RetryCount:    attempt - 1,
        RouteCount:    len(routes),
        CostMicroUSD:  budgetStatus.LastRecordCostMicroUSD, // Need to expose this
    },
}, nil
```

#### 4.2 Enrich Wide Event in Pipeline
**File**: `backend/internal/httpserver/public/chat_pipeline.go`

After executor returns:
```go
result, err := p.Executor.Chat(ctx, rc, alias, req, traceID, idempotencyKey)
if err != nil {
    // Enrich wide event with error info
    if event, ok := logging.WideEventFromContext(ctx); ok {
        event.ErrorType = reflect.TypeOf(err).String()
        event.ErrorMessage = err.Error()
        if status, _, ok := executor.AsAPIError(err); ok {
            event.StatusCode = status
        }
    }
    return p.HandleExecutorError(c, err)
}

// Enrich wide event with execution metrics
if event, ok := logging.WideEventFromContext(ctx); ok {
    event.ModelAlias = alias
    event.Provider = result.Metrics.Provider
    event.ProviderModel = result.Metrics.ProviderModel
    event.Deployment = result.Metrics.Deployment
    event.ProviderLatencyMs = result.Metrics.LatencyMs
    event.RetryCount = result.Metrics.RetryCount
    event.RouteCount = result.Metrics.RouteCount
    event.TokensInput = int64(result.Response.Usage.PromptTokens)
    event.TokensOutput = int64(result.Response.Usage.CompletionTokens)
    event.TokensTotal = int64(result.Response.Usage.TotalTokens)
    event.CostMicroUSD = result.Metrics.CostMicroUSD
    event.BudgetUsedCents = result.BudgetStatus.UsedCents
    event.BudgetRemainingCents = result.BudgetStatus.RemainingCents
}
```

#### 4.3 Handle Streaming Requests
**File**: `backend/internal/httpserver/public/chat_stream_pipeline.go`

The streaming pipeline uses `SetBodyStreamWriter` which runs in a goroutine after the middleware chain completes. We need to:

1. Capture the wide event reference before entering the stream writer
2. Enrich it inside the stream writer's defer
3. Emit it at stream close

```go
func (p *chatStreamPipeline) streamChat(...) error {
    ctx := c.UserContext()
    
    // Capture wide event before entering stream writer
    wideEvent, hasWideEvent := logging.WideEventFromContext(ctx)
    if hasWideEvent {
        wideEvent.Streaming = true
        wideEvent.ModelAlias = alias
    }
    
    // ... existing code ...
    
    c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
        // ... existing stream handling ...
        
        defer func() {
            // Enrich and emit wide event at stream close
            if hasWideEvent {
                wideEvent.Provider = route.Provider
                wideEvent.Deployment = route.ResolveDeployment()
                wideEvent.ProviderLatencyMs = firstTokenLatency.Milliseconds()
                wideEvent.TokensInput = int64(streamUsage.PromptTokens)
                wideEvent.TokensOutput = int64(streamUsage.CompletionTokens)
                wideEvent.TokensTotal = int64(streamUsage.TotalTokens)
                wideEvent.StatusCode = recordStatus
                wideEvent.Outcome = determineOutcome(wideEvent)
                wideEvent.DurationMs = time.Since(streamStart).Milliseconds()
                
                // Emit here since middleware won't see stream completion
                wideEvent.Emit(slog.Default())
            }
        }()
        
        // ... rest of stream handling ...
    })
    
    // For streaming, return nil to middleware but skip its emit
    // We'll emit in the stream writer's defer
    if hasWideEvent {
        // Mark event as already emitted so middleware skips it
        wideEvent.StatusCode = -1 // Sentinel value
    }
    return nil
}
```

Update middleware to check for sentinel:
```go
// In WideEvent middleware, before Emit:
if event.StatusCode == -1 {
    return err // Skip emit, streaming handler will do it
}
```

---

### Phase 5: Admin API Enrichment

#### 5.1 Add Target Resource Helper
**File**: `backend/internal/logging/wide_event.go`

```go
// SetTargetResource sets the target resource for admin operations.
func (e *WideEvent) SetTargetResource(resourceType string, resourceID string) {
    e.TargetResource = resourceType + ":" + resourceID
}
```

#### 5.2 Enrich Admin Handlers
For key admin operations, add enrichment. Example for tenant update:

**File**: `backend/internal/httpserver/admin/admin_tenants_core.go`

```go
func (h *tenantHandler) updateTenant(c *fiber.Ctx) error {
    ctx := c.UserContext()
    tenantID := c.Params("id")
    
    // Enrich wide event with target resource
    if event, ok := logging.WideEventFromContext(ctx); ok {
        event.SetTargetResource("tenant", tenantID)
    }
    
    // ... rest of handler ...
}
```

---

### Phase 6: Configuration Updates

#### 6.1 Update Sample Config
**File**: `deploy/router.example.yaml`

```yaml
logging:
  level: info
  format: json  # Required for wide events
  add_source: false
  wide_event:
    enabled: true
    include_user_agent: true
    include_headers: []
```

#### 6.2 Update Local Config
**File**: `deploy/router.local.yaml`

```yaml
logging:
  level: info
  format: json
  add_source: false
  wide_event:
    enabled: true
    include_user_agent: true
```

---

## Files to Modify/Create Summary

| File | Action | Lines Est. |
|------|--------|------------|
| `backend/internal/config/server.go` | Modify | +15 |
| `backend/internal/config/config.go` | Modify | +5 |
| `backend/internal/logging/wide_event.go` | **Create** | ~250 |
| `backend/internal/httpserver/middleware/wide_event.go` | **Create** | ~100 |
| `backend/internal/httpserver/middleware/logger.go` | **Delete** | -103 |
| `backend/internal/httpserver/server.go` | Modify | ~10 |
| `backend/internal/httpserver/public/auth.go` | Modify | +5 |
| `backend/internal/httpserver/admin/admin_middleware.go` | Modify | +10 |
| `backend/internal/executor/executor.go` | Modify | +30 |
| `backend/internal/httpserver/public/chat_pipeline.go` | Modify | +20 |
| `backend/internal/httpserver/public/chat_stream_pipeline.go` | Modify | +25 |
| `backend/internal/httpserver/public/embedding_pipeline.go` | Modify | +15 |
| `backend/internal/httpserver/public/image_pipeline.go` | Modify | +15 |
| `backend/internal/httpserver/public/audio_pipeline.go` | Modify | +15 |
| `backend/internal/httpserver/public/moderation_pipeline.go` | Modify | +15 |
| `backend/internal/httpserver/admin/admin_tenants_core.go` | Modify | +5 |
| `deploy/router.example.yaml` | Modify | +8 |
| `deploy/router.local.yaml` | Modify | +5 |

**Total**: ~2 new files, 16 modified files, 1 deleted file

---

## Expected Output Examples

### Successful Chat Completion
```json
{
  "time": "2026-01-16T21:30:45.612Z",
  "level": "INFO",
  "msg": "request",
  "timestamp": "2026-01-16T21:30:44.365Z",
  "duration_ms": 1247,
  "request_id": "req_8bf7ec2d",
  "trace_id": "abc123def456",
  "method": "POST",
  "path": "/v1/chat/completions",
  "route": "/v1/chat/completions",
  "status_code": 200,
  "bytes_sent": 1842,
  "ip": "192.168.1.42",
  "service": "open-model-gateway",
  "version": "0.1.29",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "api_key_id": "660e8400-e29b-41d4-a716-446655440001",
  "api_key_prefix": "demo",
  "model_alias": "gpt-5-mini",
  "provider": "azure",
  "provider_model": "gpt-5-mini",
  "deployment": "gpt-5-mini",
  "provider_latency_ms": 1089,
  "tokens_input": 150,
  "tokens_output": 89,
  "tokens_total": 239,
  "cost_micro_usd": 1425,
  "budget_limit_cents": 15000,
  "budget_remaining_cents": 12450,
  "outcome": "success"
}
```

### Rate Limited Request
```json
{
  "time": "2026-01-16T21:31:02.123Z",
  "level": "WARN",
  "msg": "request",
  "timestamp": "2026-01-16T21:31:02.100Z",
  "duration_ms": 23,
  "request_id": "req_9c8d3e4f",
  "method": "POST",
  "path": "/v1/chat/completions",
  "status_code": 429,
  "ip": "192.168.1.42",
  "service": "open-model-gateway",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "api_key_prefix": "demo",
  "model_alias": "gpt-5-mini",
  "rate_limited": true,
  "error_message": "rate limit exceeded",
  "outcome": "rate_limited"
}
```

### Admin Tenant Update
```json
{
  "time": "2026-01-16T21:32:15.789Z",
  "level": "INFO",
  "msg": "request",
  "timestamp": "2026-01-16T21:32:15.750Z",
  "duration_ms": 39,
  "request_id": "req_abc123",
  "method": "PATCH",
  "path": "/admin/tenants/550e8400-e29b-41d4-a716-446655440000",
  "status_code": 200,
  "ip": "10.0.0.5",
  "service": "open-model-gateway",
  "admin_user_id": "770e8400-e29b-41d4-a716-446655440002",
  "admin_user_email": "admin@example.com",
  "target_resource": "tenant:550e8400-e29b-41d4-a716-446655440000",
  "outcome": "success"
}
```

---

## Testing Plan

1. **Unit Tests**
   - `wide_event_test.go`: Test `ToSlogAttrs()` produces correct attributes
   - `wide_event_test.go`: Test `determineOutcome()` logic

2. **Integration Tests**
   - Verify JSON log output format
   - Verify all fields populated for chat/embedding/image requests
   - Verify streaming requests emit at close
   - Verify admin requests include admin context

3. **Manual Validation**
   - Run `make run-backend` with JSON logging
   - Send test requests and verify log output
   - Verify log lines are parseable by `jq`

---

## Rollout Plan

1. **Phase 1**: Merge with `wide_event.enabled: false` default
2. **Phase 2**: Enable in dev/staging, validate logs
3. **Phase 3**: Enable in production, monitor log volume
4. **Phase 4**: (Future) Add tail sampling if needed

---

## Questions Resolved

1. **Config location**: Under `logging.wide_event`
2. **Backward compatibility**: Delete old logger middleware entirely
3. **Admin enrichment**: Include `target_resource` field
4. **Streaming**: Emit single event at stream close

---

## Ready for Implementation

This plan is comprehensive and ready for execution. Shall I proceed with implementation?

package requestctx

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const fiberLocalsKey = "requestctx"

// Key is the typed context key used for storing the RequestContext.
var Key contextKey = "open-model-gateway/requestctx"

// Context captures caller identity, quota, and routing hints resolved from the API key.
type Context struct {
	TenantID              uuid.UUID
	APIKeyID              uuid.UUID
	APIKeyPrefix          string
	Scopes                []string
	BudgetLimitCents      int64
	WarningThreshold      float64
	BudgetRefreshSchedule string
	AlertsEnabled         bool
	AlertEmails           []string
	AlertWebhooks         []string
	AlertCooldown         time.Duration
	AlertLastLevel        string
	AlertLastSent         time.Time
	HasBudgetOverride     bool
}

// WithContext embeds the request context into the parent context.
func WithContext(parent context.Context, rc *Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, Key, rc)
}

// FromContext retrieves the request context if present.
func FromContext(ctx context.Context) (*Context, bool) {
	if ctx == nil {
		return nil, false
	}
	rc, ok := ctx.Value(Key).(*Context)
	return rc, ok
}

// FiberLocalsKey returns the key used in fiber.Locals for request context storage.
func FiberLocalsKey() string {
	return fiberLocalsKey
}

package containers

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	runtimeratelimits "github.com/ncecere/open_model_gateway/backend/internal/runtime/ratelimits"
)

// RateLimitContainer holds rate limiting state and services.
type RateLimitContainer struct {
	RateLimiter        *limits.RateLimiter
	KeyRateLimits      map[string]limits.LimitConfig
	TenantRateLimits   map[uuid.UUID]limits.LimitConfig
	DefaultKeyLimit    limits.LimitConfig
	DefaultTenantLimit limits.LimitConfig

	rateStore *runtimeratelimits.Store
	mu        sync.RWMutex
}

// NewRateLimitContainer creates a new RateLimitContainer.
func NewRateLimitContainer(
	limiter *limits.RateLimiter,
	keyLimits map[string]limits.LimitConfig,
	tenantLimits map[uuid.UUID]limits.LimitConfig,
	defaultKeyLimit limits.LimitConfig,
	defaultTenantLimit limits.LimitConfig,
) *RateLimitContainer {
	if keyLimits == nil {
		keyLimits = make(map[string]limits.LimitConfig)
	}
	if tenantLimits == nil {
		tenantLimits = make(map[uuid.UUID]limits.LimitConfig)
	}

	store := &runtimeratelimits.Store{
		DefaultKey:      defaultKeyLimit,
		DefaultTenant:   defaultTenantLimit,
		KeyOverrides:    keyLimits,
		TenantOverrides: tenantLimits,
	}

	return &RateLimitContainer{
		RateLimiter:        limiter,
		KeyRateLimits:      keyLimits,
		TenantRateLimits:   tenantLimits,
		DefaultKeyLimit:    defaultKeyLimit,
		DefaultTenantLimit: defaultTenantLimit,
		rateStore:          store,
	}
}

// UpdateTenantRateLimit overrides (or clears) the tenant-level rate limit.
func (r *RateLimitContainer) UpdateTenantRateLimit(tenantID uuid.UUID, cfg *limits.LimitConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rateStore != nil {
		r.rateStore.UpdateTenant(tenantID, cfg)
	}
}

// UpdateAPIKeyRateLimit overrides (or clears) the key-level rate limit.
func (r *RateLimitContainer) UpdateAPIKeyRateLimit(prefix string, cfg *limits.LimitConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rateStore != nil {
		r.rateStore.UpdateKey(prefix, cfg)
	}
}

// EffectiveRateLimits returns the applied key-level and tenant-level configs.
func (r *RateLimitContainer) EffectiveRateLimits(prefix string, tenantID uuid.UUID) (limits.LimitConfig, limits.LimitConfig) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.rateStore == nil {
		return r.DefaultKeyLimit, r.DefaultTenantLimit
	}
	return r.rateStore.Effective(prefix, tenantID)
}

// ResolveRateLimits resolves rate limits from the request context.
func (r *RateLimitContainer) ResolveRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.rateStore == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, fmt.Errorf("rate store unavailable")
	}
	return r.rateStore.Resolve(ctx, alias)
}

// AcquireRateLimits acquires rate limits for a request.
func (r *RateLimitContainer) AcquireRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, func(), error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.rateStore == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, fmt.Errorf("rate store unavailable")
	}
	return r.rateStore.Acquire(ctx, alias, r.RateLimiter)
}

// UpdateDefaults updates the default rate limit configurations.
func (r *RateLimitContainer) UpdateDefaults(defaultKeyLimit, defaultTenantLimit limits.LimitConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.DefaultKeyLimit = defaultKeyLimit
	r.DefaultTenantLimit = defaultTenantLimit
	if r.rateStore != nil {
		r.rateStore.DefaultKey = defaultKeyLimit
		r.rateStore.DefaultTenant = defaultTenantLimit
	}
}

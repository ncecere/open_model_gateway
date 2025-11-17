package ratelimits

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
)

// Store tracks runtime rate-limit overrides for keys and tenants.
type Store struct {
	DefaultKey      limits.LimitConfig
	DefaultTenant   limits.LimitConfig
	KeyOverrides    map[string]limits.LimitConfig
	TenantOverrides map[uuid.UUID]limits.LimitConfig
	keyMu           sync.RWMutex
	tenantMu        sync.RWMutex
}

// UpdateTenant mutates the tenant override map.
func (s *Store) UpdateTenant(tenantID uuid.UUID, cfg *limits.LimitConfig) {
	s.tenantMu.Lock()
	defer s.tenantMu.Unlock()
	if cfg == nil {
		delete(s.TenantOverrides, tenantID)
		return
	}
	if s.TenantOverrides == nil {
		s.TenantOverrides = make(map[uuid.UUID]limits.LimitConfig)
	}
	s.TenantOverrides[tenantID] = *cfg
}

// UpdateKey mutates the API key override map.
func (s *Store) UpdateKey(prefix string, cfg *limits.LimitConfig) {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	if cfg == nil {
		delete(s.KeyOverrides, prefix)
		return
	}
	if s.KeyOverrides == nil {
		s.KeyOverrides = make(map[string]limits.LimitConfig)
	}
	s.KeyOverrides[prefix] = *cfg
}

// Effective returns the applied configs for the given key/tenant pair.
func (s *Store) Effective(prefix string, tenantID uuid.UUID) (limits.LimitConfig, limits.LimitConfig) {
	keyCfg := s.DefaultKey
	s.keyMu.RLock()
	if override, ok := s.KeyOverrides[prefix]; ok {
		keyCfg = merge(keyCfg, override)
	}
	s.keyMu.RUnlock()
	tenantCfg := s.DefaultTenant
	s.tenantMu.RLock()
	if override, ok := s.TenantOverrides[tenantID]; ok {
		tenantCfg = merge(tenantCfg, override)
	}
	s.tenantMu.RUnlock()
	return keyCfg, tenantCfg
}

// Resolve builds storage keys + configs using the request context.
func (s *Store) Resolve(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, error) {
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, fmt.Errorf("request context missing")
	}
	keyCfg, tenantCfg := s.Effective(rc.APIKeyPrefix, rc.TenantID)
	return fmt.Sprintf("%s:%s", rc.APIKeyPrefix, alias), keyCfg, rc.TenantID.String(), tenantCfg, nil
}

// Acquire enforces rate limits and returns a release func.
func (s *Store) Acquire(ctx context.Context, alias string, limiter *limits.RateLimiter) (string, limits.LimitConfig, string, limits.LimitConfig, func(), error) {
	keyKey, keyCfg, tenantKey, tenantCfg, err := s.Resolve(ctx, alias)
	if err != nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
	}
	if limiter == nil {
		return keyKey, keyCfg, tenantKey, tenantCfg, func() {}, nil
	}
	keyAcquired := false
	tenantAcquired := false
	keyStorage := "key:" + keyKey
	tenantStorage := "tenant:" + tenantKey
	if keyCfg.RequestsPerMinute > 0 || keyCfg.ParallelRequests > 0 {
		if err := limiter.Allow(ctx, keyStorage, keyCfg); err != nil {
			return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
		}
		keyAcquired = true
	}
	if tenantCfg.RequestsPerMinute > 0 || tenantCfg.ParallelRequests > 0 {
		if err := limiter.Allow(ctx, tenantStorage, tenantCfg); err != nil {
			if keyAcquired {
				limiter.Release(ctx, keyStorage, keyCfg)
			}
			return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
		}
		tenantAcquired = true
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			if tenantAcquired {
				limiter.Release(ctx, tenantStorage, tenantCfg)
			}
			if keyAcquired {
				limiter.Release(ctx, keyStorage, keyCfg)
			}
		})
	}
	return keyKey, keyCfg, tenantKey, tenantCfg, release, nil
}

func merge(base, override limits.LimitConfig) limits.LimitConfig {
	if override.RequestsPerMinute > 0 {
		base.RequestsPerMinute = override.RequestsPerMinute
	}
	if override.TokensPerMinute > 0 {
		base.TokensPerMinute = override.TokensPerMinute
	}
	if override.ParallelRequests > 0 {
		base.ParallelRequests = override.ParallelRequests
	}
	return base
}

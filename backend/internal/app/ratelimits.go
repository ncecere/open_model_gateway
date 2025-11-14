package app

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
)

// LoadRateLimitDefaults pulls persisted rate limit defaults (if present) into cfg.
func LoadRateLimitDefaults(ctx context.Context, queries *db.Queries, cfg *config.Config) error {
	if queries == nil || cfg == nil {
		return nil
	}
	record, err := queries.GetRateLimitDefaults(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	cfg.RateLimits = RateLimitConfigFromRecord(cfg.RateLimits, record)
	return nil
}

// RateLimitConfigFromRecord overwrites config defaults using a stored record.
func RateLimitConfigFromRecord(base config.RateLimitConfig, record db.RateLimitDefault) config.RateLimitConfig {
	if record.RequestsPerMinute >= 0 {
		base.DefaultRequestsPerMinute = int(record.RequestsPerMinute)
	}
	if record.TokensPerMinute >= 0 {
		base.DefaultTokensPerMinute = int(record.TokensPerMinute)
	}
	if record.ParallelRequestsKey >= 0 {
		base.DefaultParallelRequestsKey = int(record.ParallelRequestsKey)
	}
	if record.ParallelRequestsTenant >= 0 {
		base.DefaultParallelRequestsTenant = int(record.ParallelRequestsTenant)
	}
	return base
}

// UpdateRateLimitConfig refreshes in-memory defaults + limiter settings.
func (c *Container) UpdateRateLimitConfig(cfg config.RateLimitConfig) {
	c.Config.RateLimits = cfg
	c.DefaultKeyLimit = limits.LimitConfig{
		RequestsPerMinute: cfg.DefaultRequestsPerMinute,
		TokensPerMinute:   cfg.DefaultTokensPerMinute,
		ParallelRequests:  cfg.DefaultParallelRequestsKey,
	}
	c.DefaultTenantLimit = limits.LimitConfig{
		RequestsPerMinute: cfg.DefaultRequestsPerMinute,
		TokensPerMinute:   cfg.DefaultTokensPerMinute,
		ParallelRequests:  cfg.DefaultParallelRequestsTenant,
	}
}

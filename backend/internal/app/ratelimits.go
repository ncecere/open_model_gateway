package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
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

// LoadTenantRateLimitOverrides returns tenant-level overrides stored in the database.
func LoadTenantRateLimitOverrides(ctx context.Context, queries *db.Queries) (map[uuid.UUID]limits.LimitConfig, error) {
	result := make(map[uuid.UUID]limits.LimitConfig)
	if queries == nil {
		return result, nil
	}
	rows, err := queries.ListTenantRateLimits(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		tenantID, err := uuidFromPg(row.TenantID)
		if err != nil {
			continue
		}
		result[tenantID] = limits.LimitConfig{
			RequestsPerMinute: int(row.RequestsPerMinute),
			TokensPerMinute:   int(row.TokensPerMinute),
			ParallelRequests:  int(row.ParallelRequests),
		}
	}
	return result, nil
}

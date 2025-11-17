package app

import (
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
)

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
	if c.rateStore != nil {
		c.rateStore.DefaultKey = c.DefaultKeyLimit
		c.rateStore.DefaultTenant = c.DefaultTenantLimit
	}
}

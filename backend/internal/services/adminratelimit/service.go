package adminratelimit

import (
	"context"
	"errors"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

var (
	ErrServiceUnavailable = errors.New("admin rate limit service not initialized")
	ErrInvalidRateLimit   = errors.New("rate limit values must be positive")
)

// Service manages platform-level rate limit defaults.
type Service struct {
	repo Repository
	cfg  *config.Config
}

// NewService builds an admin rate limit service.
// Deprecated: Use NewServiceWithRepository for new code.
func NewService(queries *db.Queries, cfg *config.Config) *Service {
	return NewServiceWithRepository(NewQueriesRepository(queries), cfg)
}

// NewServiceWithRepository builds an admin rate limit service with a Repository interface.
func NewServiceWithRepository(repo Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// DefaultUpdate captures the editable request payload for defaults.
type DefaultUpdate struct {
	RequestsPerMinute      int
	TokensPerMinute        int
	ParallelRequestsKey    int
	ParallelRequestsTenant int
}

// UpdateDefaults validates + persists the new defaults, returning the refreshed config snapshot.
func (s *Service) UpdateDefaults(ctx context.Context, req DefaultUpdate) (config.RateLimitConfig, error) {
	if s == nil || s.repo == nil || s.cfg == nil {
		return config.RateLimitConfig{}, ErrServiceUnavailable
	}
	if req.RequestsPerMinute <= 0 || req.TokensPerMinute <= 0 || req.ParallelRequestsKey <= 0 || req.ParallelRequestsTenant <= 0 {
		return config.RateLimitConfig{}, ErrInvalidRateLimit
	}

	if _, err := s.repo.UpsertRateLimitDefaults(ctx, db.UpsertRateLimitDefaultsParams{
		RequestsPerMinute:      int32(req.RequestsPerMinute),
		TokensPerMinute:        int32(req.TokensPerMinute),
		ParallelRequestsKey:    int32(req.ParallelRequestsKey),
		ParallelRequestsTenant: int32(req.ParallelRequestsTenant),
	}); err != nil {
		return config.RateLimitConfig{}, err
	}

	updated := config.RateLimitConfig{
		DefaultRequestsPerMinute:      req.RequestsPerMinute,
		DefaultTokensPerMinute:        req.TokensPerMinute,
		DefaultParallelRequestsKey:    req.ParallelRequestsKey,
		DefaultParallelRequestsTenant: req.ParallelRequestsTenant,
	}
	s.cfg.RateLimits = updated
	return updated, nil
}

package ratelimits

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
)

// LoadDefaults pulls persisted rate limit defaults (if present) into cfg.
func LoadDefaults(ctx context.Context, queries *db.Queries, cfg *config.Config) error {
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
	cfg.RateLimits = FromRecord(cfg.RateLimits, record)
	return nil
}

// FromRecord overwrites config defaults using a stored record.
func FromRecord(base config.RateLimitConfig, record db.RateLimitDefault) config.RateLimitConfig {
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

// LoadTenantOverrides returns tenant-level overrides stored in the database.
func LoadTenantOverrides(ctx context.Context, queries *db.Queries) (map[uuid.UUID]limits.LimitConfig, error) {
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

// LoadKeyOverrides returns API key overrides stored in the database.
func LoadKeyOverrides(ctx context.Context, queries *db.Queries) (map[string]limits.LimitConfig, error) {
	result := make(map[string]limits.LimitConfig)
	if queries == nil {
		return result, nil
	}
	rows, err := queries.ListAPIKeyRateLimits(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		prefix := strings.TrimSpace(row.Prefix)
		if prefix == "" {
			continue
		}
		result[prefix] = limits.LimitConfig{
			RequestsPerMinute: int(row.RequestsPerMinute),
			TokensPerMinute:   int(row.TokensPerMinute),
			ParallelRequests:  int(row.ParallelRequests),
		}
	}
	return result, nil
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

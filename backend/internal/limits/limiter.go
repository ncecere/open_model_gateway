package limits

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLimitExceeded = errors.New("rate limit exceeded")

type LimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
	ParallelRequests  int
}

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (l *RateLimiter) Allow(ctx context.Context, key string, overrides LimitConfig) error {
	if l == nil || l.client == nil {
		return nil
	}

	cfg := overrides
	if cfg.RequestsPerMinute > 0 {
		if err := l.countCheck(ctx, fmt.Sprintf("rpm:%s", key), time.Minute, cfg.RequestsPerMinute); err != nil {
			return err
		}
	}
	if cfg.ParallelRequests > 0 {
		if err := l.semaphoreAcquire(ctx, fmt.Sprintf("sem:%s", key), cfg.ParallelRequests); err != nil {
			return err
		}
	}

	return nil
}

func (l *RateLimiter) Release(ctx context.Context, key string, cfg LimitConfig) {
	if l == nil || l.client == nil {
		return
	}
	if cfg.ParallelRequests > 0 {
		l.semaphoreRelease(ctx, fmt.Sprintf("sem:%s", key))
	}
}

func (l *RateLimiter) countCheck(ctx context.Context, key string, ttl time.Duration, limit int) error {
	now := time.Now().UTC().Unix() / int64(ttl.Seconds())
	redisKey := fmt.Sprintf("%s:%d", key, now)

	cnt, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return err
	}
	if cnt == 1 {
		l.client.Expire(ctx, redisKey, ttl)
	}
	if int(cnt) > limit {
		return ErrLimitExceeded
	}
	return nil
}

func (l *RateLimiter) semaphoreAcquire(ctx context.Context, key string, max int) error {
	ttl := 5 * time.Minute
	redisKey := key
	cnt, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return err
	}
	if cnt == 1 {
		l.client.Expire(ctx, redisKey, ttl)
	}
	if int(cnt) > max {
		l.client.Decr(ctx, redisKey)
		return ErrLimitExceeded
	}
	return nil
}

func (l *RateLimiter) semaphoreRelease(ctx context.Context, key string) {
	l.client.Decr(ctx, key)
}

func (l *RateLimiter) TokenAllowance(ctx context.Context, key string, tokens int, cfg LimitConfig) error {
	if cfg.TokensPerMinute <= 0 {
		return nil
	}
	now := time.Now().UTC().Unix() / 60
	redisKey := fmt.Sprintf("tpm:%s:%d", key, now)

	used, err := l.client.IncrBy(ctx, redisKey, int64(tokens)).Result()
	if err != nil {
		return err
	}
	if used == int64(tokens) {
		l.client.Expire(ctx, redisKey, time.Minute)
	}
	if int(used) > cfg.TokensPerMinute {
		l.client.IncrBy(ctx, redisKey, -int64(tokens))
		return ErrLimitExceeded
	}
	return nil
}

func ParseLimits(metadata map[string]string, defaults LimitConfig) LimitConfig {
	cfg := defaults
	if metadata == nil {
		return cfg
	}
	if v, ok := metadata["requests_per_minute"]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.RequestsPerMinute = i
		}
	}
	if v, ok := metadata["tokens_per_minute"]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.TokensPerMinute = i
		}
	}
	if v, ok := metadata["parallel_requests"]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.ParallelRequests = i
		}
	}
	return cfg
}

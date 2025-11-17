package limits

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	promreg "github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var ErrLimitExceeded = errors.New("rate limit exceeded")

type LimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
	ParallelRequests  int
}

type RateLimiter struct {
	client *redis.Client
	tracer trace.Tracer
	gauge  *promreg.GaugeVec
}

func NewRateLimiter(client *redis.Client, gauge *promreg.GaugeVec) *RateLimiter {
	return &RateLimiter{client: client, tracer: otel.Tracer("open-model-gateway/ratelimiter"), gauge: gauge}
}

func (l *RateLimiter) Allow(ctx context.Context, key string, overrides LimitConfig) error {
	if l == nil || l.client == nil {
		return nil
	}

	cfg := overrides
	ctx, span := l.startSpan(ctx, "RateLimiter.Allow", key, cfg)
	defer span.End()
	if cfg.RequestsPerMinute > 0 {
		if err := l.countCheck(ctx, fmt.Sprintf("rpm:%s", key), time.Minute, cfg.RequestsPerMinute); err != nil {
			l.recordError(span, err)
			return err
		}
	}
	if cfg.ParallelRequests > 0 {
		if err := l.semaphoreAcquire(ctx, fmt.Sprintf("sem:%s", key), cfg.ParallelRequests); err != nil {
			l.recordError(span, err)
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
	ctx, span := l.startSpan(ctx, "RateLimiter.SemaphoreAcquire", key, LimitConfig{ParallelRequests: max})
	defer span.End()
	ttl := 5 * time.Minute
	redisKey := key
	cnt, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		l.recordError(span, err)
		return err
	}
	if cnt == 1 {
		l.client.Expire(ctx, redisKey, ttl)
	}
	if int(cnt) > max {
		l.client.Decr(ctx, redisKey)
		err := ErrLimitExceeded
		l.recordError(span, err)
		return err
	}
	if l != nil && l.gauge != nil {
		l.gauge.WithLabelValues(redisKey, "parallel").Set(float64(cnt))
	}
	return nil
}

func (l *RateLimiter) semaphoreRelease(ctx context.Context, key string) {
	if l == nil || l.client == nil {
		return
	}
	cnt, err := l.client.Decr(ctx, key).Result()
	if err != nil {
		return
	}
	if l.gauge != nil {
		l.gauge.WithLabelValues(key, "parallel").Set(float64(cnt))
	}
}

func (l *RateLimiter) TokenAllowance(ctx context.Context, key string, tokens int, cfg LimitConfig) error {
	if cfg.TokensPerMinute <= 0 {
		return nil
	}
	ctx, span := l.startSpan(ctx, "RateLimiter.TokenAllowance", key, cfg, attribute.Int("tokens", tokens))
	defer span.End()
	now := time.Now().UTC().Unix() / 60
	redisKey := fmt.Sprintf("tpm:%s:%d", key, now)

	used, err := l.client.IncrBy(ctx, redisKey, int64(tokens)).Result()
	if err != nil {
		l.recordError(span, err)
		return err
	}
	if used == int64(tokens) {
		l.client.Expire(ctx, redisKey, time.Minute)
	}
	if int(used) > cfg.TokensPerMinute {
		l.client.IncrBy(ctx, redisKey, -int64(tokens))
		err := ErrLimitExceeded
		l.recordError(span, err)
		return err
	}
	return nil
}

func (l *RateLimiter) startSpan(ctx context.Context, name, key string, cfg LimitConfig, extra ...attribute.KeyValue) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("key", key),
		attribute.Int("rpm", cfg.RequestsPerMinute),
		attribute.Int("tpm", cfg.TokensPerMinute),
		attribute.Int("parallel", cfg.ParallelRequests),
	}
	attrs = append(attrs, extra...)
	if l == nil || l.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return l.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

func (l *RateLimiter) recordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	if errors.Is(err, ErrLimitExceeded) {
		span.SetAttributes(attribute.String("status", "limit_exceeded"))
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
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

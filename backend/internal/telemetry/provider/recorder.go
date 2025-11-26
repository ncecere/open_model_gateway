package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Recorder writes provider call samples into Redis rolling windows for SLI evaluation.
type Recorder struct {
	client         *redis.Client
	windowSize     time.Duration
	latencyBuckets []int64
}

// NewRecorder constructs a recorder; windowSize should align to the telemetry provider window.
func NewRecorder(client *redis.Client, windowSize time.Duration) *Recorder {
	return &Recorder{
		client:         client,
		windowSize:     windowSize,
		latencyBuckets: defaultLatencyBucketsMs(),
	}
}

// RecordSample aggregates a provider call into the current minute bucket.
func (r *Recorder) RecordSample(ctx context.Context, sample Sample) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("telemetry recorder not initialized")
	}
	if sample.Provider == "" || sample.ModelAlias == "" {
		return fmt.Errorf("provider and model alias are required")
	}
	if sample.Timestamp.IsZero() {
		sample.Timestamp = time.Now().UTC()
	}
	bucketStart := sample.Timestamp.UTC().Truncate(time.Minute)
	key := r.bucketKey(sample.Provider, sample.ModelAlias, sample.Route, bucketStart)

	pipe := r.client.TxPipeline()
	pipe.HIncrBy(ctx, key, "req_total", 1)

	switch sample.Result {
	case ResultError:
		pipe.HIncrBy(ctx, key, "err_total", 1)
	case ResultTimeout:
		pipe.HIncrBy(ctx, key, "timeout_total", 1)
	case ResultCanceled:
		pipe.HIncrBy(ctx, key, "canceled_total", 1)
	}
	if sample.ErrorClass != "" {
		field := fmt.Sprintf("err_class_%s", normalizeMetricLabel(sample.ErrorClass))
		pipe.HIncrBy(ctx, key, field, 1)
	}

	if sample.Latency > 0 {
		ms := sample.Latency.Milliseconds()
		pipe.HIncrBy(ctx, key, "latency_ms_sum", ms)
		pipe.HIncrBy(ctx, key, "latency_count", 1)
		bucketField := fmt.Sprintf("latency_le_%d", r.latencyBucket(ms))
		pipe.HIncrBy(ctx, key, bucketField, 1)
	}
	if sample.InputTokens > 0 {
		pipe.HIncrBy(ctx, key, "tokens_in_total", sample.InputTokens)
	}
	if sample.OutputTokens > 0 {
		pipe.HIncrBy(ctx, key, "tokens_out_total", sample.OutputTokens)
	}
	pipe.Expire(ctx, key, r.windowSize+2*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("record telemetry sample: %w", err)
	}
	return nil
}

func (r *Recorder) bucketKey(provider, alias, route string, bucket time.Time) string {
	parts := []string{
		"telemetry", "provider", normalizeMetricLabel(provider), normalizeMetricLabel(alias),
	}
	if route = strings.TrimSpace(route); route != "" {
		parts = append(parts, normalizeMetricLabel(route))
	}
	parts = append(parts, fmt.Sprintf("%d", bucket.Unix()))
	return strings.Join(parts, ":")
}

func (r *Recorder) latencyBucket(ms int64) int64 {
	for _, upper := range r.latencyBuckets {
		if ms <= upper {
			return upper
		}
	}
	return r.latencyBuckets[len(r.latencyBuckets)-1]
}

func normalizeMetricLabel(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

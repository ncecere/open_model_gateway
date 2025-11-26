package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Identifier uniquely marks a provider/model/route triple.
type Identifier struct {
	Provider   string
	ModelAlias string
	Route      string
}

// WindowSummary aggregates counts across a time window.
type WindowSummary struct {
	Identifier
	Requests        int64
	Errors          int64
	Timeouts        int64
	Canceled        int64
	TokensIn        int64
	TokensOut       int64
	LatencyCount    int64
	LatencyMsSum    int64
	LatencyBuckets  map[int64]int64
	ErrorClassCount map[string]int64
}

// AggregateWindow scans Redis buckets for the given window and returns per-route summaries.
func (r *Recorder) AggregateWindow(ctx context.Context, window time.Duration, now time.Time) (map[Identifier]WindowSummary, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("telemetry recorder not initialized")
	}
	if window <= 0 {
		window = r.windowSize
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	start := now.UTC().Add(-window).Truncate(time.Minute)
	end := now.UTC().Truncate(time.Minute)

	out := make(map[Identifier]WindowSummary)
	iter := r.client.Scan(ctx, 0, "telemetry:provider:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		id, ts, err := parseKey(key)
		if err != nil {
			continue
		}
		bucketTime := time.Unix(ts, 0).UTC()
		if bucketTime.Before(start) || bucketTime.After(end) {
			continue
		}
		fields, err := r.client.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}
		sum := out[id]
		sum.Identifier = id
		for k, v := range fields {
			val, _ := strconv.ParseInt(v, 10, 64)
			switch {
			case k == "req_total":
				sum.Requests += val
			case k == "err_total":
				sum.Errors += val
			case k == "timeout_total":
				sum.Timeouts += val
			case k == "canceled_total":
				sum.Canceled += val
			case k == "tokens_in_total":
				sum.TokensIn += val
			case k == "tokens_out_total":
				sum.TokensOut += val
			case k == "latency_ms_sum":
				sum.LatencyMsSum += val
			case k == "latency_count":
				sum.LatencyCount += val
			case strings.HasPrefix(k, "latency_le_"):
				if sum.LatencyBuckets == nil {
					sum.LatencyBuckets = make(map[int64]int64)
				}
				if bucket, err := strconv.ParseInt(strings.TrimPrefix(k, "latency_le_"), 10, 64); err == nil {
					sum.LatencyBuckets[bucket] += val
				}
			case strings.HasPrefix(k, "err_class_"):
				if sum.ErrorClassCount == nil {
					sum.ErrorClassCount = make(map[string]int64)
				}
				class := strings.TrimPrefix(k, "err_class_")
				sum.ErrorClassCount[class] += val
			}
		}
		out[id] = sum
	}
	if err := iter.Err(); err != nil && err != redis.Nil {
		return out, err
	}
	return out, nil
}

func parseKey(key string) (Identifier, int64, error) {
	parts := strings.Split(key, ":")
	// Expect telemetry:provider:{provider}:{alias}:{route}:{unix}
	if len(parts) < 4 {
		return Identifier{}, 0, fmt.Errorf("invalid key")
	}
	tsPart := parts[len(parts)-1]
	ts, err := strconv.ParseInt(tsPart, 10, 64)
	if err != nil {
		return Identifier{}, 0, fmt.Errorf("invalid timestamp")
	}
	id := Identifier{
		Provider:   parts[2],
		ModelAlias: parts[3],
	}
	switch len(parts) {
	case 5:
		// no route present
	case 6:
		id.Route = parts[4]
	default:
		return Identifier{}, 0, fmt.Errorf("invalid key")
	}
	return id, ts, nil
}

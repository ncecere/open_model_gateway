package provider

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

func TestServiceCurrentSLIs(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := NewRecorder(client, 5*time.Minute)

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	samples := []Sample{
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultSuccess, Latency: 1200 * time.Millisecond, Timestamp: now.Add(-3 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultTimeout, ErrorClass: "timeout", Latency: 2500 * time.Millisecond, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultSuccess, Latency: 1500 * time.Millisecond, Timestamp: now.Add(-90 * time.Second)},
	}
	for _, s := range samples {
		if err := rec.RecordSample(ctx, s); err != nil {
			t.Fatalf("record sample: %v", err)
		}
	}

	cfg := config.TelemetryConfig{
		Provider: config.ProviderTelemetryConfig{
			WindowSize: 5 * time.Minute,
			Defaults: config.ProviderSLIDefaults{
				LatencyP95Ms:         2000,
				ErrorRateThreshold:   0.2,
				TimeoutRateThreshold: 0.1,
				MinSamples:           1,
			},
		},
	}
	svc := NewService(rec, nil, cfg, config.BudgetConfig{}, nil, nil)
	svc.now = func() time.Time { return now }

	results, err := svc.CurrentSLIs(ctx)
	if err != nil {
		t.Fatalf("current slis: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 sli result, got %d", len(results))
	}
	res := results[0]
	if res.RequestCount != 3 {
		t.Fatalf("expected 3 requests, got %d", res.RequestCount)
	}
	if res.TimeoutCount != 1 {
		t.Fatalf("expected 1 timeout, got %d", res.TimeoutCount)
	}
	if res.ErrorRate != 0 {
		t.Fatalf("expected zero error rate, got %f", res.ErrorRate)
	}
	if res.LatencyP95Ms == 0 {
		t.Fatalf("expected latency p95 to be set")
	}
	if res.WindowStart != now.Add(-cfg.Provider.WindowSize) {
		t.Fatalf("unexpected window start %v", res.WindowStart)
	}
	if res.WindowEnd != now {
		t.Fatalf("unexpected window end %v", res.WindowEnd)
	}
}

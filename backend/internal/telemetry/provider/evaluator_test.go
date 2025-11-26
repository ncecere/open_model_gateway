package provider

import (
	"testing"

	"context"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"time"
)

func TestThresholdsForOverrides(t *testing.T) {
	defaults := config.ProviderSLIDefaults{
		LatencyP95Ms:         5000,
		ErrorRateThreshold:   0.1,
		TimeoutRateThreshold: 0.05,
		MinSamples:           50,
	}
	overrides := map[string]config.ProviderSLIDefaults{
		"openai": {
			LatencyP95Ms:         3000,
			ErrorRateThreshold:   0.05,
			TimeoutRateThreshold: 0.02,
			MinSamples:           20,
		},
	}
	ov := thresholdsFor(Identifier{Provider: "OpenAI"}, defaults, overrides)
	if ov.LatencyP95Ms != 3000 || ov.ErrorRateThreshold != 0.05 || ov.TimeoutRateThreshold != 0.02 || ov.MinSamples != 20 {
		t.Fatalf("unexpected override %+v", ov)
	}
	dv := thresholdsFor(Identifier{Provider: "other"}, defaults, overrides)
	if dv.LatencyP95Ms != defaults.LatencyP95Ms || dv.ErrorRateThreshold != defaults.ErrorRateThreshold || dv.MinSamples != defaults.MinSamples {
		t.Fatalf("defaults not applied %+v", dv)
	}
}

func TestEvaluatorDetectsErrorIncident(t *testing.T) {
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
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultError, ErrorClass: "5xx", Latency: 3 * time.Second, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultError, ErrorClass: "5xx", Latency: 3 * time.Second, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultSuccess, Latency: 1 * time.Second, Timestamp: now.Add(-90 * time.Second)},
	}
	for _, s := range samples {
		if err := rec.RecordSample(ctx, s); err != nil {
			t.Fatalf("record sample: %v", err)
		}
	}

	ev := NewEvaluator(rec, func(id Identifier) SLIThresholds {
		return SLIThresholds{
			LatencyP95Ms:         2500,
			ErrorRateThreshold:   0.25,
			TimeoutRateThreshold: 0.5,
			MinSamples:           1,
		}
	})
	ev.now = func() time.Time { return now }

	incidents, err := ev.Evaluate(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(incidents) != 2 {
		t.Fatalf("expected 2 incidents (latency + error), got %d", len(incidents))
	}
	var types []string
	for _, inc := range incidents {
		types = append(types, inc.Type)
		if inc.Provider != "openai" || inc.ModelAlias != "gpt-5" {
			t.Fatalf("unexpected identifier %+v", inc.Identifier)
		}
		if inc.RequestCount != 3 {
			t.Fatalf("expected request count 3, got %d", inc.RequestCount)
		}
	}
	want := map[string]bool{"error_rate": true, "latency_p95": true}
	for _, typ := range types {
		if !want[typ] {
			t.Fatalf("unexpected incident type %s", typ)
		}
	}
}

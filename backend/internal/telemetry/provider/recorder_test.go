package provider

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRecorderAggregateWindow(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	rec := NewRecorder(client, 5*time.Minute)
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	samples := []Sample{
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultSuccess, Latency: 1200 * time.Millisecond, InputTokens: 10, OutputTokens: 20, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "east", Result: ResultError, ErrorClass: "5xx", Latency: 1500 * time.Millisecond, Timestamp: now.Add(-2 * time.Minute)},
		{Provider: "openai", ModelAlias: "gpt-5", Route: "west", Result: ResultTimeout, ErrorClass: "timeout", Latency: 5 * time.Second, Timestamp: now.Add(-1 * time.Minute)},
		{Provider: "anthropic", ModelAlias: "claude-3", Route: "", Result: ResultSuccess, Latency: 800 * time.Millisecond, InputTokens: 5, OutputTokens: 5, Timestamp: now.Add(-30 * time.Second)},
	}
	for _, s := range samples {
		if err := rec.RecordSample(ctx, s); err != nil {
			t.Fatalf("record sample: %v", err)
		}
	}

	out, err := rec.AggregateWindow(ctx, 5*time.Minute, now)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}

	id := Identifier{Provider: "openai", ModelAlias: "gpt-5", Route: "east"}
	sum, ok := out[id]
	if !ok {
		t.Fatalf("missing summary for %v", id)
	}
	if sum.Requests != 2 {
		t.Fatalf("expected 2 requests, got %d", sum.Requests)
	}
	if sum.Errors != 1 {
		t.Fatalf("expected 1 error, got %d", sum.Errors)
	}
	if sum.LatencyCount != 2 {
		t.Fatalf("expected latency count 2, got %d", sum.LatencyCount)
	}
	if sum.LatencyMsSum <= 0 {
		t.Fatalf("expected latency sum >0, got %d", sum.LatencyMsSum)
	}
	if sum.ErrorClassCount["5xx"] != 1 {
		t.Fatalf("expected 5xx count 1, got %d", sum.ErrorClassCount["5xx"])
	}

	timeoutID := Identifier{Provider: "openai", ModelAlias: "gpt-5", Route: "west"}
	timeoutSum := out[timeoutID]
	if timeoutSum.Timeouts != 1 {
		t.Fatalf("expected timeout count 1, got %d", timeoutSum.Timeouts)
	}

	anthID := Identifier{Provider: "anthropic", ModelAlias: "claude-3", Route: ""}
	anthSum := out[anthID]
	if anthSum.TokensIn != 5 || anthSum.TokensOut != 5 {
		t.Fatalf("expected tokens 5/5, got %d/%d", anthSum.TokensIn, anthSum.TokensOut)
	}
}

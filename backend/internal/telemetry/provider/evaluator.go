package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// SLIThresholds captures thresholds for a provider/model.
type SLIThresholds struct {
	LatencyP95Ms         int
	ErrorRateThreshold   float64
	TimeoutRateThreshold float64
	MinSamples           int
}

// thresholdsFor merges defaults with optional per-provider overrides.
func thresholdsFor(id Identifier, defaults config.ProviderSLIDefaults, overrides map[string]config.ProviderSLIDefaults) SLIThresholds {
	o := defaults
	if overrides != nil {
		if ov, ok := overrides[strings.ToLower(id.Provider)]; ok {
			o = ov
		}
	}
	return SLIThresholds{
		LatencyP95Ms:         orDefaultInt(o.LatencyP95Ms, defaults.LatencyP95Ms),
		ErrorRateThreshold:   orDefaultFloat(o.ErrorRateThreshold, defaults.ErrorRateThreshold),
		TimeoutRateThreshold: orDefaultFloat(o.TimeoutRateThreshold, defaults.TimeoutRateThreshold),
		MinSamples:           orDefaultInt(o.MinSamples, defaults.MinSamples),
	}
}

func orDefaultInt(v int, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func orDefaultFloat(v float64, def float64) float64 {
	if v > 0 {
		return v
	}
	return def
}

// IncidentStatus represents current state.
type IncidentStatus string

const (
	IncidentOpen     IncidentStatus = "open"
	IncidentResolved IncidentStatus = "resolved"
)

// Incident summarizes a detected breach.
type Incident struct {
	Identifier
	ID            string
	Type          string
	Status        IncidentStatus
	WindowSeconds int
	OpenedAt      time.Time
	ResolvedAt    *time.Time
	WindowStart   time.Time
	WindowEnd     time.Time
	SampleError   string
	RequestCount  int64
	ErrorCount    int64
	TimeoutCount  int64
	LatencyP95Ms  int64
	Metadata      map[string]any
}

// Evaluator processes Redis window summaries into SLI incidents.
type Evaluator struct {
	recorder   *Recorder
	thresholds func(id Identifier) SLIThresholds
	now        func() time.Time
}

// NewEvaluator creates an SLI evaluator.
func NewEvaluator(rec *Recorder, thresholds func(id Identifier) SLIThresholds) *Evaluator {
	return &Evaluator{
		recorder:   rec,
		thresholds: thresholds,
		now:        time.Now,
	}
}

// Evaluate computes SLIs for the configured window and returns incidents to open/resolve.
func (e *Evaluator) Evaluate(ctx context.Context, window time.Duration) ([]Incident, error) {
	if e == nil || e.recorder == nil {
		return nil, fmt.Errorf("evaluator not initialized")
	}
	now := e.now().UTC()
	summaries, err := e.recorder.AggregateWindow(ctx, window, now)
	if err != nil {
		return nil, err
	}
	var incidents []Incident
	for id, sum := range summaries {
		th := e.thresholds(id)
		if th.MinSamples <= 0 {
			th.MinSamples = 1
		}
		if sum.Requests < int64(th.MinSamples) {
			continue
		}
		p95 := percentileFromBuckets(sum.LatencyBuckets, sum.LatencyCount)
		errRate := float64(sum.Errors) / float64(sum.Requests)
		timeoutRate := float64(sum.Timeouts) / float64(sum.Requests)
		windowSeconds := int(window.Seconds())
		windowStart := now.Add(-window).UTC()

		if p95 > int64(th.LatencyP95Ms) {
			incidents = append(incidents, Incident{
				Identifier:    id,
				Type:          "latency_p95",
				Status:        IncidentOpen,
				WindowSeconds: windowSeconds,
				WindowStart:   windowStart,
				WindowEnd:     now,
				RequestCount:  sum.Requests,
				LatencyP95Ms:  p95,
				Metadata: map[string]any{
					"threshold_ms": th.LatencyP95Ms,
				},
			})
		}
		if errRate > th.ErrorRateThreshold {
			incidents = append(incidents, Incident{
				Identifier:    id,
				Type:          "error_rate",
				Status:        IncidentOpen,
				WindowSeconds: windowSeconds,
				WindowStart:   windowStart,
				WindowEnd:     now,
				RequestCount:  sum.Requests,
				ErrorCount:    sum.Errors,
				Metadata: map[string]any{
					"error_rate": errRate,
					"threshold":  th.ErrorRateThreshold,
				},
			})
		}
		if timeoutRate > th.TimeoutRateThreshold {
			incidents = append(incidents, Incident{
				Identifier:    id,
				Type:          "timeout_rate",
				Status:        IncidentOpen,
				WindowSeconds: windowSeconds,
				WindowStart:   windowStart,
				WindowEnd:     now,
				RequestCount:  sum.Requests,
				TimeoutCount:  sum.Timeouts,
				Metadata: map[string]any{
					"timeout_rate": timeoutRate,
					"threshold":    th.TimeoutRateThreshold,
				},
			})
		}
	}
	return incidents, nil
}

// percentileFromBuckets estimates p95 from cumulative buckets.
func percentileFromBuckets(buckets map[int64]int64, count int64) int64 {
	if count == 0 || len(buckets) == 0 {
		return 0
	}
	target := float64(count) * 0.95
	type bucket struct {
		upper int64
		count int64
	}
	bs := make([]bucket, 0, len(buckets))
	for upper, c := range buckets {
		bs = append(bs, bucket{upper: upper, count: c})
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].upper < bs[j].upper })

	var cumulative int64
	for _, b := range bs {
		cumulative += b.count
		if float64(cumulative) >= target {
			return b.upper
		}
	}
	return bs[len(bs)-1].upper
}

package runtime

import (
	"context"
	"sync"
	"time"
)

// HealthResult captures the outcome of a single check.
type HealthResult struct {
	Name    string
	Latency time.Duration
	Err     error
}

// HealthReporter exposes the ability to run health checks on demand.
type HealthReporter interface {
	Run(ctx context.Context) []HealthResult
}

type healthCheck struct {
	name string
	fn   func(ctx context.Context) error
}

// HealthChecks aggregates registered health probes.
type HealthChecks struct {
	mu     sync.RWMutex
	checks []healthCheck
}

// NewHealthChecks returns an empty registry ready for registration.
func NewHealthChecks() *HealthChecks {
	return &HealthChecks{}
}

// Register adds a named check to the registry.
func (h *HealthChecks) Register(name string, fn func(ctx context.Context) error) {
	if h == nil || fn == nil || name == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks = append(h.checks, healthCheck{name: name, fn: fn})
}

// Run executes all registered checks and returns their results. Each check runs
// serially; callers can control concurrency by registering composite functions.
func (h *HealthChecks) Run(ctx context.Context) []HealthResult {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	checks := make([]healthCheck, len(h.checks))
	copy(checks, h.checks)
	h.mu.RUnlock()
	results := make([]HealthResult, 0, len(checks))
	for _, check := range checks {
		start := time.Now()
		err := check.fn(ctx)
		results = append(results, HealthResult{
			Name:    check.name,
			Latency: time.Since(start),
			Err:     err,
		})
	}
	return results
}

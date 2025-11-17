package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

// Options wraps configuration for runtime initialization.
type Options struct {
	Config config.Options
	// SkipMigrations bypasses Goose migrations (useful for read-only tooling).
	SkipMigrations bool
}

// Runtime owns the core dependencies needed to run the router process.
type Runtime struct {
	Config    *config.Config
	Container *app.Container

	datastores *Datastores
	redis      *redis.Client
	jobs       *jobsStage
	health     *HealthChecks
}

// New constructs the runtime by loading configuration, running migrations, and
// initializing the shared container.
func New(ctx context.Context, opts Options) (*Runtime, error) {
	return NewBuilder(opts).Build(ctx)
}

// Shutdown releases runtime resources. It is safe to call multiple times.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var errs []error
	if r.Container != nil {
		if closer, ok := r.Container.UsageLogger.(interface{ Shutdown(context.Context) error }); ok {
			if err := closer.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown usage logger: %w", err))
			}
		}
		if r.Container.Observability != nil {
			if err := r.Container.Observability.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown observability: %w", err))
			}
		}
	}
	if r.redis != nil {
		if err := r.redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close redis: %w", err))
		}
		r.redis = nil
	}
	if r.datastores != nil {
		r.datastores.Close()
		r.datastores = nil
	}
	return errors.Join(errs...)
}

// StartJobs launches background workers (batch processor, file sweeper, etc.).
// It is safe to call multiple times; subsequent calls are no-ops.
func (r *Runtime) StartJobs(ctx context.Context) {
	if r == nil || r.jobs == nil {
		return
	}
	r.jobs.Start(ctx)
}

// HealthReporter returns the registry of runtime health checks.
func (r *Runtime) HealthReporter() HealthReporter {
	if r == nil {
		return nil
	}
	return r.health
}

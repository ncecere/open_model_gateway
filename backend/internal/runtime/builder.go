package runtime

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/batchworker"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/redisclient"
)

// Builder orchestrates staged runtime construction. As additional runtime
// modules (services, routing, telemetry, jobs) come online we can extend this
// type with explicit stages instead of letting callers poke at
// `app.Container` directly.
type Builder struct {
	opts       Options
	datastores *Datastores
	config     *config.Config
	redis      *redis.Client
	services   *servicesStage
	routing    *routingStage
	telemetry  *telemetryStage
	jobs       *jobsStage
	health     *HealthChecks
}

// servicesStage describes the dependencies produced after wiring core services.
// Future iterations will populate this with concrete service pointers exported by
// the builder so other stages (routing, telemetry, jobs) can consume them
// without touching `app.Container` directly.
type servicesStage struct {
	services *app.Services
}

// routingStage captures router/provider dependencies derived from config,
// services, and database state.
type routingStage struct {
	routing *app.Routing
}

// telemetryStage bundles observability, usage logging, and background job
// metadata soon to be wired during the telemetry/jobs phase.
type telemetryStage struct {
	telemetry *app.Telemetry
}

// NewBuilder returns a build coordinator using the provided options.
func NewBuilder(opts Options) *Builder {
	return &Builder{opts: opts}
}

// Build executes the staged construction flow and returns a ready Runtime.
func (b *Builder) Build(ctx context.Context) (*Runtime, error) {
	if err := b.loadDatastores(ctx); err != nil {
		return nil, err
	}
	if err := b.connectRedis(ctx); err != nil {
		b.closeDatastores()
		return nil, err
	}
	if err := b.buildServices(ctx); err != nil {
		b.redis.Close()
		b.closeDatastores()
		return nil, err
	}
	if err := b.buildRouting(ctx); err != nil {
		b.redis.Close()
		b.closeDatastores()
		return nil, err
	}
	if err := b.buildTelemetry(ctx); err != nil {
		b.redis.Close()
		b.closeDatastores()
		return nil, err
	}
	stages := &app.ContainerStages{}
	if b.services != nil {
		stages.Services = b.services.services
	}
	if b.routing != nil {
		stages.Routing = b.routing.routing
	}
	if b.telemetry != nil {
		stages.Telemetry = b.telemetry.telemetry
	}
	container, err := app.NewContainer(ctx, b.config, b.datastores.DB, b.redis, stages)
	if err != nil {
		b.redis.Close()
		b.closeDatastores()
		return nil, fmt.Errorf("build container: %w", err)
	}
	b.registerRouterHealth(container)
	b.buildJobs(container)
	return &Runtime{
		Config:     b.config,
		Container:  container,
		datastores: b.datastores,
		redis:      b.redis,
		jobs:       b.jobs,
		health:     b.health,
	}, nil
}

func (b *Builder) loadDatastores(ctx context.Context) error {
	if b.datastores != nil {
		return nil
	}
	datastores, err := OpenDatastores(ctx, b.opts)
	if err != nil {
		return err
	}
	b.datastores = datastores
	b.config = datastores.Config
	if db := datastores.DB; db != nil {
		h := b.ensureHealth()
		h.Register("postgres", db.Ping)
	}
	return nil
}

func (b *Builder) connectRedis(ctx context.Context) error {
	if b.redis != nil {
		return nil
	}
	client := redisclient.New(b.config.Redis)
	if err := redisclient.Ping(ctx, client); err != nil {
		client.Close()
		return fmt.Errorf("connect redis: %w", err)
	}
	b.redis = client
	b.ensureHealth().Register("redis", func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	})
	return nil
}

func (b *Builder) closeDatastores() {
	if b.datastores != nil {
		b.datastores.Close()
		b.datastores = nil
	}
}

// buildServices will eventually construct the core service layer (accounts,
// tenants, admin services, usage, etc.) using the already-initialized config,
// database, and redis clients.
func (b *Builder) buildServices(ctx context.Context) error {
	if b.services != nil {
		return nil
	}
	svcs, err := app.BuildServices(ctx, b.config, b.datastores.DB)
	if err != nil {
		return err
	}
	b.services = &servicesStage{services: svcs}
	return nil
}

// buildRouting will handle provider catalog hydration, router engine creation,
// and rate-limit/tenant stores by consuming the services stage output.
func (b *Builder) buildRouting(ctx context.Context) error {
	if b.routing != nil {
		return nil
	}
	if b.services == nil || b.services.services == nil {
		return fmt.Errorf("services stage required before routing")
	}
	routing, err := app.BuildRouting(ctx, b.config, b.services.services.Queries)
	if err != nil {
		return err
	}
	b.routing = &routingStage{routing: routing}
	return nil
}

// buildTelemetry will take the accumulated runtime dependencies and wire
// observability, usage logging, and background jobs (batch worker, file
// sweeper, etc.).
func (b *Builder) buildTelemetry(ctx context.Context) error {
	if b.telemetry != nil {
		return nil
	}
	if b.services == nil || b.services.services == nil {
		return fmt.Errorf("services stage required before telemetry")
	}
	if b.routing == nil || b.routing.routing == nil {
		return fmt.Errorf("routing stage required before telemetry")
	}
	telem, err := app.BuildTelemetry(ctx, b.config, b.datastores.DB, b.services.services.Queries, b.routing.routing.Entries)
	if err != nil {
		return err
	}
	b.telemetry = &telemetryStage{telemetry: telem}
	return nil
}

func (b *Builder) buildJobs(container *app.Container) {
	if b.jobs != nil || container == nil {
		return
	}
	stage := &jobsStage{}
	stage.start = func(ctx context.Context) {
		if container.Batches != nil {
			go batchworker.New(container, executor.New(container)).Run(ctx)
		}
		if container.Files != nil && b.config != nil {
			startFileSweeper(ctx, container.Files, b.config.Files)
		}
	}
	b.jobs = stage
}

func (b *Builder) registerRouterHealth(container *app.Container) {
	if container == nil || container.Engine == nil {
		return
	}
	engine := container.Engine
	b.ensureHealth().Register("router", func(ctx context.Context) error {
		aliases := engine.ListAliases()
		if len(aliases) == 0 {
			return fmt.Errorf("no provider routes configured")
		}
		return nil
	})
}

func (b *Builder) ensureHealth() *HealthChecks {
	if b.health == nil {
		b.health = NewHealthChecks()
	}
	return b.health
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	promreg "github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/app/containers"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/cache"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/health"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/logging"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
	runtimebudgets "github.com/ncecere/open_model_gateway/backend/internal/runtime/budgets"
	runtimeratelimits "github.com/ncecere/open_model_gateway/backend/internal/runtime/ratelimits"
	runtimetype "github.com/ncecere/open_model_gateway/backend/internal/runtime/tenants"
	adminapikeyssvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminapikeys"
	adminauditsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminaudit"
	adminbudgetsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminbudget"
	admincatalogsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admincatalog"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
	adminprovidersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminprovider"
	adminratelimitsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminratelimit"
	adminrbacsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminrbac"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
	adminusersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminuser"
	auditservice "github.com/ncecere/open_model_gateway/backend/internal/services/audit"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	billinghooks "github.com/ncecere/open_model_gateway/backend/internal/services/billinghooks"
	exportsvc "github.com/ncecere/open_model_gateway/backend/internal/services/exports"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	"github.com/ncecere/open_model_gateway/backend/internal/services/responsestore"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
	usageService "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

// Container aggregates runtime dependencies for handlers and services.
// It composes themed sub-containers for better organization:
//   - Data: database pool, Redis, queries
//   - Services: business logic services
//   - Routing: provider factory, router engine, health monitor
//   - Telemetry: observability, usage logging, metrics
//   - RateLimits: rate limiting state and services
type Container struct {
	// Sub-containers (new architecture)
	Data       *containers.DataContainer
	Services   *containers.ServiceContainer
	Routing    *containers.RoutingContainer
	Telemetry  *containers.TelemetryContainer
	RateLimits *containers.RateLimitContainer

	// Configuration
	Config *config.Config

	// Logger is the configured structured logger.
	Logger *logging.Logger

	// Legacy fields (maintained for backwards compatibility)
	// These delegate to sub-containers where applicable
	DBPool               *pgxpool.Pool
	Redis                *redis.Client
	Queries              *db.Queries
	Accounts             *accounts.PersonalService
	AdminUsers           *adminusersvc.Service
	AdminCatalog         *admincatalogsvc.Service
	AdminBudgets         *adminbudgetsvc.Service
	AdminRateLimits      *adminratelimitsvc.Service
	AdminProviders       *adminprovidersvc.Service
	AdminTenants         *admintenantsvc.Service
	AdminRBAC            *adminrbacsvc.Service
	AdminConfig          *adminconfigsvc.Service
	AdminAudit           *adminauditsvc.Service
	AdminAPIKeys         *adminapikeyssvc.Service
	Batches              *batchsvc.Service
	Exports              *exportsvc.Service
	BillingWebhooks      *billinghooks.Service
	DefaultModels        *catalog.DefaultModelService
	UsageService         *usageService.Service
	TenantService        *tenantservice.Service
	AdminAuth            *auth.AdminAuthService
	Factory              *providers.Factory
	Engine               *router.Engine
	RateLimiter          *limits.RateLimiter
	KeyRateLimits        map[string]limits.LimitConfig
	TenantRateLimits     map[uuid.UUID]limits.LimitConfig
	DefaultKeyLimit      limits.LimitConfig
	DefaultTenantLimit   limits.LimitConfig
	rateStore            *runtimeratelimits.Store
	UsageLogger          UsageLogger
	ProviderTelemetry    *providermetrics.Recorder
	ProviderTelemetrySvc *providermetrics.Service
	Idempotency          *cache.IdempotencyCache
	HealthMon            *health.Monitor
	Observability        *observability.Provider
	Files                *filesvc.Service
	ResponseStore        *responsestore.Service
	TelemetryCancel      context.CancelFunc
	tenantModels         *runtimetype.AccessStore
	tenantRateLimitMu    sync.RWMutex
	keyRateLimitMu       sync.RWMutex
	ReportingLocation    *time.Location
}

// ContainerStages allows callers (e.g., runtime.Builder) to pre-build core stages
// before constructing the final container. When nil, NewContainer will build the
// stages itself.
type ContainerStages struct {
	Services  *Services
	Routing   *Routing
	Telemetry *Telemetry
}

// NewContainer builds a dependency container from the provided primitives.
func NewContainer(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, redisClient *redis.Client, stages *ContainerStages) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if pool == nil {
		return nil, fmt.Errorf("db pool is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	// Build or use provided services stage
	var services *Services
	if stages != nil {
		services = stages.Services
	}
	if services == nil {
		var err error
		services, err = BuildServices(ctx, cfg, pool)
		if err != nil {
			return nil, err
		}
	}
	queries := services.Queries
	reportingLoc := services.ReportingLocation
	personalSvc := services.Personal
	defaultModels := services.DefaultModels
	usageSvc := services.Usage
	tenantSvc := services.Tenant
	providerSvc := services.AdminProvider
	adminAuth := services.AdminAuth
	adminUserSvc := services.AdminUsers
	keyLimitOverrides := services.KeyRateLimits
	tenantLimitOverrides := services.TenantRateLimits

	// Build or use provided routing stage
	var routing *Routing
	if stages != nil {
		routing = stages.Routing
	}
	if routing == nil {
		var err error
		routing, err = BuildRouting(ctx, cfg, queries)
		if err != nil {
			return nil, err
		}
	}
	entries := routing.Entries
	factory := routing.Factory
	engine := routing.Engine

	// Build or use provided telemetry stage
	var telemetry *Telemetry
	var telemetryCancel context.CancelFunc
	if stages != nil {
		telemetry = stages.Telemetry
	}
	if telemetry == nil {
		var err error
		telemetry, telemetryCancel, err = BuildTelemetry(ctx, cfg, pool, queries, entries, redisClient)
		if err != nil {
			return nil, err
		}
	}

	// NOTE: bootstrap.Ensure is called inside BuildServices; no need to repeat here.

	// Set up telemetry guard for degraded providers
	var telemetryGuard func(alias, routeKey string) bool
	if telemetry != nil && telemetry.ProviderSvc != nil && cfg.Telemetry.Provider.DownweightWhenDegraded {
		telemetryGuard = telemetry.ProviderSvc.IsDegraded
	}

	// Initialize rate limiter
	var rateGauge *promreg.GaugeVec
	if telemetry != nil && telemetry.Observability != nil {
		rateGauge = telemetry.Observability.RateLimiterGauge()
	}
	rateLimiter := limits.NewRateLimiter(redisClient, rateGauge)
	idem := cache.NewIdempotencyCache(redisClient, 30*time.Minute)

	// Initialize health monitor
	monitor := health.NewMonitor(engine, cfg.Health)
	if telemetry != nil {
		rec := telemetry.ProviderRec
		obs := telemetry.Observability
		monitor.WithRecorder(func(sample providermetrics.Sample) {
			if rec != nil {
				_ = rec.RecordSample(context.Background(), sample)
			}
			if obs != nil {
				obs.RecordProviderMetrics(sample)
			}
		})
	}
	monitor.Start(ctx, func() map[string][]providers.Route {
		return engine.ListAliases()
	})
	if telemetryGuard != nil {
		engine.SetTelemetryGuard(func(alias, routeKey string) bool {
			return telemetryGuard(alias, routeKey)
		})
	}

	// Extract telemetry components
	usageLogger := telemetry.UsageLogger
	obsProvider := telemetry.Observability
	filesService := telemetry.Files
	batchesService := telemetry.Batches
	adminConfigService := telemetry.AdminConfig
	providerRec := telemetry.ProviderRec
	providerTelemetrySvc := telemetry.ProviderSvc

	// Build rate limit defaults
	defaultKeyLimit := limits.LimitConfig{
		RequestsPerMinute: cfg.RateLimits.DefaultRequestsPerMinute,
		TokensPerMinute:   cfg.RateLimits.DefaultTokensPerMinute,
		ParallelRequests:  cfg.RateLimits.DefaultParallelRequestsKey,
	}
	defaultTenantLimit := limits.LimitConfig{
		RequestsPerMinute: cfg.RateLimits.DefaultRequestsPerMinute,
		TokensPerMinute:   cfg.RateLimits.DefaultTokensPerMinute,
		ParallelRequests:  cfg.RateLimits.DefaultParallelRequestsTenant,
	}

	rateStore := &runtimeratelimits.Store{
		DefaultKey:      defaultKeyLimit,
		DefaultTenant:   defaultTenantLimit,
		KeyOverrides:    keyLimitOverrides,
		TenantOverrides: tenantLimitOverrides,
	}

	// Build sub-containers
	dataContainer := containers.NewDataContainer(pool, redisClient)
	// Override queries to use the one from services (already initialized)
	dataContainer.Queries = queries

	routingContainer := containers.NewRoutingContainer(factory, engine, monitor)

	telemetryContainer := containers.NewTelemetryContainer(
		obsProvider,
		usageLogger,
		providerRec,
		providerTelemetrySvc,
		idem,
		telemetryCancel,
	)

	rateLimitContainer := containers.NewRateLimitContainer(
		rateLimiter,
		keyLimitOverrides,
		tenantLimitOverrides,
		defaultKeyLimit,
		defaultTenantLimit,
	)

	// Build container with both sub-containers and legacy fields
	container := &Container{
		// Sub-containers
		Data:       dataContainer,
		Routing:    routingContainer,
		Telemetry:  telemetryContainer,
		RateLimits: rateLimitContainer,

		// Configuration
		Config: cfg,

		// Legacy fields (for backwards compatibility)
		DBPool:               pool,
		Redis:                redisClient,
		Queries:              queries,
		Accounts:             personalSvc,
		AdminUsers:           adminUserSvc,
		DefaultModels:        defaultModels,
		AdminProviders:       providerSvc,
		UsageService:         usageSvc,
		TenantService:        tenantSvc,
		AdminAuth:            adminAuth,
		Factory:              factory,
		Engine:               engine,
		RateLimiter:          rateLimiter,
		KeyRateLimits:        keyLimitOverrides,
		TenantRateLimits:     tenantLimitOverrides,
		DefaultKeyLimit:      defaultKeyLimit,
		DefaultTenantLimit:   defaultTenantLimit,
		rateStore:            rateStore,
		UsageLogger:          usageLogger,
		ProviderTelemetry:    providerRec,
		ProviderTelemetrySvc: providerTelemetrySvc,
		Idempotency:          idem,
		HealthMon:            monitor,
		Observability:        obsProvider,
		Files:                filesService,
		AdminConfig:          adminConfigService,
		Batches:              batchesService,
		Exports: func() *exportsvc.Service {
			if filesService == nil {
				return nil
			}
			return exportsvc.NewService(queries, filesService, reportingLoc)
		}(),
		BillingWebhooks:   billinghooks.NewService(queries, cfg.Budgets.Alert.Webhook, nil),
		TelemetryCancel:   telemetryCancel,
		ReportingLocation: reportingLoc,
	}

	// Build ServiceContainer
	container.Services = &containers.ServiceContainer{
		Accounts:          personalSvc,
		AdminUsers:        adminUserSvc,
		AdminAuth:         adminAuth,
		TenantService:     tenantSvc,
		DefaultModels:     defaultModels,
		UsageService:      usageSvc,
		Files:             filesService,
		Batches:           batchesService,
		AdminProviders:    providerSvc,
		AdminConfig:       adminConfigService,
		BillingWebhooks:   container.BillingWebhooks,
		Exports:           container.Exports,
		ReportingLocation: reportingLoc,
	}

	// Set up tenant model updater callback
	personalSvc.SetTenantModelUpdater(func(id uuid.UUID, aliases []string) {
		container.SetTenantModels(id, aliases)
	})

	// Initialize admin services that need container references
	container.AdminCatalog = admincatalogsvc.NewService(queries, container.ReloadRouter)
	container.AdminBudgets = adminbudgetsvc.NewService(queries, cfg)
	container.AdminRateLimits = adminratelimitsvc.NewService(queries, cfg)
	container.AdminTenants = admintenantsvc.NewService(cfg, queries, reportingLoc, pool, personalSvc, adminAuth, container.SetTenantModels, container.UpdateTenantRateLimit, container.UpdateAPIKeyRateLimit)
	container.AdminRBAC = adminrbacsvc.NewService(queries)
	container.AdminAudit = adminauditsvc.NewService(auditservice.NewService(queries))
	container.AdminAPIKeys = adminapikeyssvc.NewService(queries)
	container.ResponseStore = responsestore.New(queries, container.Logger, 7*24*time.Hour)

	// Update ServiceContainer with admin services
	container.Services.AdminTenants = container.AdminTenants
	container.Services.AdminCatalog = container.AdminCatalog
	container.Services.AdminBudgets = container.AdminBudgets
	container.Services.AdminRateLimits = container.AdminRateLimits
	container.Services.AdminRBAC = container.AdminRBAC
	container.Services.AdminAudit = container.AdminAudit
	container.Services.AdminAPIKeys = container.AdminAPIKeys

	// Load tenant model access
	if err := container.loadTenantModelAccess(ctx); err != nil {
		return nil, err
	}

	return container, nil
}

// ReloadRouter rebuilds provider routes after catalog changes.
func (c *Container) ReloadRouter(ctx context.Context) error {
	dbEntries, err := c.Queries.ListModelCatalog(ctx)
	if err != nil {
		return err
	}

	entries, err := router.MergeEntries(c.Config.ModelCatalog, dbEntries)
	if err != nil {
		return err
	}

	override := *c.Config
	override.ModelCatalog = entries

	factory := providers.NewFactory(&override)
	if err := c.Engine.Reload(ctx, factory); err != nil {
		return err
	}

	c.Factory = factory
	if c.Routing != nil {
		c.Routing.Factory = factory
	}
	if c.UsageLogger != nil {
		c.UsageLogger.LoadCatalog(entries)
	}
	return nil
}

func (c *Container) loadTenantModelAccess(ctx context.Context) error {
	if c == nil || c.Queries == nil {
		return nil
	}
	store, err := runtimetype.LoadModelAccess(ctx, c.Queries)
	if err != nil {
		return err
	}
	c.tenantModels = store
	if c.Routing != nil {
		c.Routing.LoadTenantModelAccess(store)
	}
	return nil
}

// RecordProviderTelemetry fans out provider samples to metrics and Redis windows.
func (c *Container) RecordProviderTelemetry(ctx context.Context, sample providermetrics.Sample) {
	if c == nil {
		return
	}
	if c.Telemetry != nil {
		c.Telemetry.RecordProviderTelemetry(ctx, sample)
		return
	}
	// Legacy fallback
	if c.ProviderTelemetry != nil {
		_ = c.ProviderTelemetry.RecordSample(ctx, sample)
	}
	if c.Observability != nil {
		c.Observability.RecordProviderMetrics(sample)
	}
}

// ReportingLoc returns the configured reporting timezone location (defaults to UTC).
func (c *Container) ReportingLoc() *time.Location {
	if c != nil && c.ReportingLocation != nil {
		return c.ReportingLocation
	}
	return time.UTC
}

// Log returns the configured logger, or slog.Default() if not set.
func (c *Container) Log() *slog.Logger {
	if c != nil && c.Logger != nil {
		return c.Logger.Logger
	}
	return slog.Default()
}

func (c *Container) IsModelAllowed(tenantID uuid.UUID, alias string) bool {
	if c == nil {
		return true
	}
	if c.Routing != nil {
		return c.Routing.IsModelAllowed(tenantID, alias)
	}
	// Legacy fallback
	if c.tenantModels == nil {
		return true
	}
	return c.tenantModels.IsAllowed(tenantID, alias)
}

func (c *Container) SetTenantModels(tenantID uuid.UUID, aliases []string) {
	if c.Routing != nil {
		c.Routing.SetTenantModels(tenantID, aliases)
		return
	}
	// Legacy fallback
	store := c.ensureTenantModels()
	if store == nil {
		return
	}
	store.Set(tenantID, aliases)
}

func (c *Container) ClearTenantModels(tenantID uuid.UUID) {
	if c.Routing != nil {
		c.Routing.ClearTenantModels(tenantID)
		return
	}
	// Legacy fallback
	if store := c.ensureTenantModels(); store != nil {
		store.Clear(tenantID)
	}
}

// UpdateTenantRateLimit overrides (or clears) the tenant-level rate limit.
func (c *Container) UpdateTenantRateLimit(tenantID uuid.UUID, cfg *limits.LimitConfig) {
	if c.RateLimits != nil {
		c.RateLimits.UpdateTenantRateLimit(tenantID, cfg)
		return
	}
	// Legacy fallback
	store := c.ensureRateStore()
	if store == nil {
		return
	}
	store.UpdateTenant(tenantID, cfg)
}

// UpdateAPIKeyRateLimit overrides (or clears) the key-level rate limit.
func (c *Container) UpdateAPIKeyRateLimit(prefix string, cfg *limits.LimitConfig) {
	if c.RateLimits != nil {
		c.RateLimits.UpdateAPIKeyRateLimit(prefix, cfg)
		return
	}
	// Legacy fallback
	store := c.ensureRateStore()
	if store == nil {
		return
	}
	store.UpdateKey(prefix, cfg)
}

// EffectiveTenantBudget returns the tenant's budget limit (USD) and warning threshold.
func (c *Container) EffectiveTenantBudget(ctx context.Context, tenantID uuid.UUID) (float64, float64, error) {
	if c == nil || c.Config == nil {
		return 0, 0, errors.New("configuration unavailable")
	}
	return runtimebudgets.Effective(ctx, c.Queries, c.Config.Budgets, tenantID)
}

func normalizeModelAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

// EffectiveRateLimits returns the applied key-level and tenant-level configs
// without requiring a request context.
func (c *Container) EffectiveRateLimits(prefix string, tenantID uuid.UUID) (limits.LimitConfig, limits.LimitConfig) {
	if c.RateLimits != nil {
		return c.RateLimits.EffectiveRateLimits(prefix, tenantID)
	}
	// Legacy fallback
	store := c.ensureRateStore()
	if store == nil {
		return c.DefaultKeyLimit, c.DefaultTenantLimit
	}
	return store.Effective(prefix, tenantID)
}

func (c *Container) ResolveRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, error) {
	if c.RateLimits != nil {
		return c.RateLimits.ResolveRateLimits(ctx, alias)
	}
	// Legacy fallback
	store := c.ensureRateStore()
	if store == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, fmt.Errorf("request context missing")
	}
	return store.Resolve(ctx, alias)
}

func (c *Container) AcquireRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, func(), error) {
	if c.RateLimits != nil {
		return c.RateLimits.AcquireRateLimits(ctx, alias)
	}
	// Legacy fallback
	store := c.ensureRateStore()
	if store == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, fmt.Errorf("rate store unavailable")
	}
	return store.Acquire(ctx, alias, c.RateLimiter)
}

func (c *Container) ensureRateStore() *runtimeratelimits.Store {
	if c == nil {
		return nil
	}
	if c.rateStore == nil {
		if c.KeyRateLimits == nil {
			c.KeyRateLimits = make(map[string]limits.LimitConfig)
		}
		if c.TenantRateLimits == nil {
			c.TenantRateLimits = make(map[uuid.UUID]limits.LimitConfig)
		}
		c.rateStore = &runtimeratelimits.Store{
			DefaultKey:      c.DefaultKeyLimit,
			DefaultTenant:   c.DefaultTenantLimit,
			KeyOverrides:    c.KeyRateLimits,
			TenantOverrides: c.TenantRateLimits,
		}
	}
	return c.rateStore
}

func (c *Container) ensureTenantModels() *runtimetype.AccessStore {
	if c == nil {
		return nil
	}
	if c.tenantModels == nil {
		c.tenantModels = runtimetype.NewAccessStore(nil)
	}
	return c.tenantModels
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

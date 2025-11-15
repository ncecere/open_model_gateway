package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	decimal "github.com/shopspring/decimal"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/cache"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/health"
	"github.com/ncecere/open_model_gateway/backend/internal/limits"
	"github.com/ncecere/open_model_gateway/backend/internal/observability"
	"github.com/ncecere/open_model_gateway/backend/internal/providers"
	"github.com/ncecere/open_model_gateway/backend/internal/rbac"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	"github.com/ncecere/open_model_gateway/backend/internal/router"
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
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
	usageService "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
	"github.com/ncecere/open_model_gateway/backend/internal/storage/blob"
)

// Container aggregates runtime dependencies for handlers and services.
type Container struct {
	Config             *config.Config
	DBPool             *pgxpool.Pool
	Redis              *redis.Client
	Queries            *db.Queries
	Accounts           *accounts.PersonalService
	AdminUsers         *adminusersvc.Service
	AdminCatalog       *admincatalogsvc.Service
	AdminBudgets       *adminbudgetsvc.Service
	AdminRateLimits    *adminratelimitsvc.Service
	AdminProviders     *adminprovidersvc.Service
	AdminTenants       *admintenantsvc.Service
	AdminRBAC          *adminrbacsvc.Service
	AdminConfig        *adminconfigsvc.Service
	AdminAudit         *adminauditsvc.Service
	Batches            *batchsvc.Service
	DefaultModels      *catalog.DefaultModelService
	UsageService       *usageService.Service
	TenantService      *tenantservice.Service
	AdminAuth          *auth.AdminAuthService
	Factory            *providers.Factory
	Engine             *router.Engine
	RateLimiter        *limits.RateLimiter
	KeyRateLimits      map[string]limits.LimitConfig
	TenantRateLimits   map[uuid.UUID]limits.LimitConfig
	DefaultKeyLimit    limits.LimitConfig
	DefaultTenantLimit limits.LimitConfig
	UsageLogger        *usagepipeline.Logger
	Idempotency        *cache.IdempotencyCache
	HealthMon          *health.Monitor
	Observability      *observability.Provider
	Files              *filesvc.Service
	tenantModelMu      sync.RWMutex
	tenantModelAccess  map[uuid.UUID]map[string]struct{}
	tenantRateLimitMu  sync.RWMutex
	ReportingLocation  *time.Location
}

// NewContainer builds a dependency container from the provided primitives.
func NewContainer(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, redisClient *redis.Client) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if pool == nil {
		return nil, fmt.Errorf("db pool is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	locName := strings.TrimSpace(cfg.Reporting.Timezone)
	if locName == "" {
		locName = "UTC"
	}
	reportingLoc, err := time.LoadLocation(locName)
	if err != nil {
		return nil, fmt.Errorf("load reporting timezone: %w", err)
	}

	queries := db.New(pool)
	adminconfigsvc.ApplyOverrides(ctx, queries, cfg)
	personalSvc := accounts.NewPersonalService(pool, queries)
	defaultModels := catalog.NewDefaultModelService(queries)
	usageSvc := usageService.NewService(queries, reportingLoc)
	tenantSvc := tenantservice.NewService(cfg, queries, reportingLoc)
	providerSvc := adminprovidersvc.NewService()

	if err := LoadRateLimitDefaults(ctx, queries, cfg); err != nil {
		return nil, fmt.Errorf("load rate limit defaults: %w", err)
	}
	if err := LoadBudgetDefaults(ctx, queries, cfg); err != nil {
		return nil, fmt.Errorf("load budget defaults: %w", err)
	}

	adminAuth, err := auth.NewAdminAuthService(ctx, cfg.Admin, queries, personalSvc)
	if err != nil {
		return nil, fmt.Errorf("init admin auth: %w", err)
	}
	adminUserSvc := adminusersvc.NewService(queries, personalSvc, adminAuth)

	dbEntries, err := queries.ListModelCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("load model catalog: %w", err)
	}

	entries, err := router.MergeEntries(cfg.ModelCatalog, dbEntries)
	if err != nil {
		return nil, fmt.Errorf("merge model catalog: %w", err)
	}

	override := *cfg
	override.ModelCatalog = entries

	factory := providers.NewFactory(&override)
	engine := router.NewEngine()
	if err := engine.Reload(ctx, factory); err != nil {
		return nil, fmt.Errorf("init router engine: %w", err)
	}

	if err := ensureCatalogPersisted(ctx, queries, entries); err != nil {
		return nil, err
	}

	keyLimitOverrides := make(map[string]limits.LimitConfig)
	tenantLimitOverrides, err := LoadTenantRateLimitOverrides(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("load tenant rate limits: %w", err)
	}
	if tenantLimitOverrides == nil {
		tenantLimitOverrides = make(map[uuid.UUID]limits.LimitConfig)
	}

	if err := ensureBootstrap(ctx, queries, adminAuth, personalSvc, cfg.Bootstrap, cfg.Budgets, keyLimitOverrides, tenantLimitOverrides); err != nil {
		return nil, err
	}

	rateLimiter := limits.NewRateLimiter(redisClient)
	idem := cache.NewIdempotencyCache(redisClient, 30*time.Minute)

	monitor := health.NewMonitor(engine, cfg.Health)
	monitor.Start(ctx, func() map[string][]providers.Route {
		return engine.ListAliases()
	})

	obsProvider, err := observability.Setup(ctx, cfg.Observability)
	if err != nil {
		return nil, fmt.Errorf("setup observability: %w", err)
	}

	alertSink := usagepipeline.NewCompositeSink(
		usagepipeline.NewSMTPSink(cfg.Budgets.Alert.SMTP, slog.Default()),
		usagepipeline.NewWebhookSink(cfg.Budgets.Alert.Webhook, slog.Default()),
		usagepipeline.NewLogAlertSink(slog.Default()),
	)
	usageLogger := usagepipeline.NewLogger(pool, queries, cfg.Budgets, alertSink, obsProvider)
	usageLogger.LoadCatalog(entries)

	blobStore, err := blob.New(ctx, cfg.Files)
	if err != nil {
		return nil, fmt.Errorf("init blob store: %w", err)
	}
	filesService := filesvc.NewService(queries, blobStore, &cfg.Files)
	batchesService := batchsvc.NewService(pool, queries, filesService, &cfg.Batches)
	adminConfigService := adminconfigsvc.NewService(queries, cfg, filesService, batchesService)

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

	container := &Container{
		Config:             cfg,
		DBPool:             pool,
		Redis:              redisClient,
		Queries:            queries,
		Accounts:           personalSvc,
		AdminUsers:         adminUserSvc,
		DefaultModels:      defaultModels,
		AdminProviders:     providerSvc,
		UsageService:       usageSvc,
		TenantService:      tenantSvc,
		AdminAuth:          adminAuth,
		Factory:            factory,
		Engine:             engine,
		RateLimiter:        rateLimiter,
		KeyRateLimits:      keyLimitOverrides,
		TenantRateLimits:   tenantLimitOverrides,
		DefaultKeyLimit:    defaultKeyLimit,
		DefaultTenantLimit: defaultTenantLimit,
		UsageLogger:        usageLogger,
		Idempotency:        idem,
		HealthMon:          monitor,
		Observability:      obsProvider,
		Files:              filesService,
		AdminConfig:        adminConfigService,
		Batches:            batchesService,
		ReportingLocation:  reportingLoc,
		tenantModelAccess:  make(map[uuid.UUID]map[string]struct{}),
	}

	personalSvc.SetTenantModelUpdater(func(id uuid.UUID, aliases []string) {
		container.SetTenantModels(id, aliases)
	})

	container.AdminCatalog = admincatalogsvc.NewService(queries, container.ReloadRouter)
	container.AdminBudgets = adminbudgetsvc.NewService(queries, cfg)
	container.AdminRateLimits = adminratelimitsvc.NewService(queries, cfg)
container.AdminTenants = admintenantsvc.NewService(cfg, queries, reportingLoc, pool, personalSvc, adminAuth, container.SetTenantModels, container.UpdateTenantRateLimit)
	container.AdminRBAC = adminrbacsvc.NewService(queries)
	container.AdminAudit = adminauditsvc.NewService(auditservice.NewService(queries))

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
	if c.UsageLogger != nil {
		c.UsageLogger.LoadCatalog(entries)
	}
	return nil
}

func (c *Container) loadTenantModelAccess(ctx context.Context) error {
	if c == nil || c.Queries == nil {
		return nil
	}

	// Seed every tenant with an explicit entry (possibly empty) so we can
	// distinguish between "no models allowed" vs "not yet configured".
	tenantRows, err := c.Queries.ListTenants(ctx, db.ListTenantsParams{
		Limit:  math.MaxInt32,
		Offset: 0,
	})
	if err != nil {
		return err
	}

	modelMap := make(map[uuid.UUID]map[string]struct{}, len(tenantRows))
	for _, row := range tenantRows {
		id, err := uuidFromPg(row.ID)
		if err != nil {
			continue
		}
		modelMap[id] = make(map[string]struct{})
	}

	rows, err := c.Queries.ListAllTenantModels(ctx)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if !row.TenantID.Valid {
			continue
		}
		tenantID, err := uuid.FromBytes(row.TenantID.Bytes[:])
		if err != nil {
			continue
		}
		alias := normalizeModelAlias(row.Alias)
		if alias == "" {
			continue
		}
		set := modelMap[tenantID]
		if set == nil {
			set = make(map[string]struct{})
			modelMap[tenantID] = set
		}
		set[alias] = struct{}{}
	}

	c.tenantModelMu.Lock()
	c.tenantModelAccess = modelMap
	c.tenantModelMu.Unlock()
	return nil
}

// ReportingLoc returns the configured reporting timezone location (defaults to UTC).
func (c *Container) ReportingLoc() *time.Location {
	if c != nil && c.ReportingLocation != nil {
		return c.ReportingLocation
	}
	return time.UTC
}

func (c *Container) IsModelAllowed(tenantID uuid.UUID, alias string) bool {
	if c == nil {
		return true
	}

	normalized := normalizeModelAlias(alias)
	if normalized == "" {
		return false
	}

	c.tenantModelMu.RLock()
	defer c.tenantModelMu.RUnlock()

	if len(c.tenantModelAccess) == 0 {
		return true
	}

	allowed, ok := c.tenantModelAccess[tenantID]
	if !ok {
		return true
	}
	if len(allowed) == 0 {
		return false
	}

	_, exists := allowed[normalized]
	return exists
}

func (c *Container) SetTenantModels(tenantID uuid.UUID, aliases []string) {
	if c == nil {
		return
	}

	set := make(map[string]struct{})
	for _, alias := range aliases {
		norm := normalizeModelAlias(alias)
		if norm != "" {
			set[norm] = struct{}{}
		}
	}

	c.tenantModelMu.Lock()
	defer c.tenantModelMu.Unlock()

	if c.tenantModelAccess == nil {
		c.tenantModelAccess = make(map[uuid.UUID]map[string]struct{})
	}
	c.tenantModelAccess[tenantID] = set
}

func (c *Container) ClearTenantModels(tenantID uuid.UUID) {
	if c == nil {
		return
	}
	c.tenantModelMu.Lock()
	delete(c.tenantModelAccess, tenantID)
	c.tenantModelMu.Unlock()
}

// UpdateTenantRateLimit overrides (or clears) the tenant-level rate limit.
func (c *Container) UpdateTenantRateLimit(tenantID uuid.UUID, cfg *limits.LimitConfig) {
	if c == nil {
		return
	}
	c.tenantRateLimitMu.Lock()
	defer c.tenantRateLimitMu.Unlock()
	if cfg == nil {
		delete(c.TenantRateLimits, tenantID)
		return
	}
	if c.TenantRateLimits == nil {
		c.TenantRateLimits = make(map[uuid.UUID]limits.LimitConfig)
	}
	c.TenantRateLimits[tenantID] = *cfg
}

func ensureCatalogPersisted(ctx context.Context, queries *db.Queries, entries []config.ModelCatalogEntry) error {
	for _, entry := range entries {
		modalitiesJSON, err := json.Marshal(entry.Modalities)
		if err != nil {
			return err
		}
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err != nil {
			return err
		}
		providerCfgJSON, err := json.Marshal(entry.ProviderOverrides)
		if err != nil {
			return err
		}

		priceInput := decimal.NewFromFloat(entry.PriceInput)
		priceOutput := decimal.NewFromFloat(entry.PriceOutput)
		if priceInput.IsNegative() {
			priceInput = decimal.Zero
		}
		if priceOutput.IsNegative() {
			priceOutput = decimal.Zero
		}
		currency := entry.Currency
		if currency == "" {
			currency = "USD"
		}

		provider := catalog.NormalizeProviderSlug(entry.Provider)

		modelType := strings.TrimSpace(entry.ModelType)
		if modelType == "" {
			modelType = "llm"
		}

		_, err = queries.UpsertModelCatalogEntry(ctx, db.UpsertModelCatalogEntryParams{
			Alias:              entry.Alias,
			Provider:           provider,
			ProviderModel:      entry.ProviderModel,
			ModelType:          modelType,
			ContextWindow:      entry.ContextWindow,
			MaxOutputTokens:    entry.MaxOutputTokens,
			ModalitiesJson:     modalitiesJSON,
			SupportsTools:      entry.SupportsTools,
			PriceInput:         priceInput,
			PriceOutput:        priceOutput,
			Currency:           currency,
			Enabled:            entry.IsEnabled(),
			Deployment:         entry.Deployment,
			Endpoint:           entry.Endpoint,
			ApiKey:             entry.APIKey,
			ApiVersion:         entry.APIVersion,
			Region:             entry.Region,
			MetadataJson:       metadataJSON,
			Weight:             int32(entry.Weight),
			ProviderConfigJson: providerCfgJSON,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeModelAlias(alias string) string {
	return strings.ToLower(strings.TrimSpace(alias))
}

func ensureBootstrap(ctx context.Context, queries *db.Queries, adminAuth *auth.AdminAuthService, personalSvc *accounts.PersonalService, bootstrap config.BootstrapConfig, budgetCfg config.BudgetConfig, keyLimits map[string]limits.LimitConfig, tenantLimits map[uuid.UUID]limits.LimitConfig) error {
	for _, tenant := range bootstrap.Tenants {
		name := strings.TrimSpace(tenant.Name)
		if name == "" {
			continue
		}
		if _, err := queries.GetTenantByName(ctx, name); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("bootstrap tenant %q lookup: %w", name, err)
			}
			status := db.TenantStatusActive
			if strings.EqualFold(tenant.Status, "suspended") {
				status = db.TenantStatusSuspended
			}
			if _, err := queries.CreateTenant(ctx, db.CreateTenantParams{
				Name:   name,
				Status: status,
				Kind:   db.TenantKindOrganization,
			}); err != nil {
				return fmt.Errorf("bootstrap tenant %q create: %w", name, err)
			}
		}
	}

	for _, user := range bootstrap.AdminUsers {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		var dbUser db.User
		var err error
		dbUser, err = queries.GetUserByEmail(ctx, email)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("bootstrap admin %q lookup: %w", email, err)
			}
			dbUser, err = queries.CreateUser(ctx, db.CreateUserParams{
				Email: email,
				Name:  strings.TrimSpace(user.Name),
			})
			if err != nil {
				return fmt.Errorf("bootstrap admin %q create: %w", email, err)
			}
		}

		if personalSvc != nil {
			if updated, _, err := personalSvc.EnsurePersonalTenant(ctx, dbUser); err == nil {
				dbUser = updated
			} else {
				return fmt.Errorf("bootstrap admin %q personal tenant: %w", email, err)
			}
		}

		if strings.TrimSpace(user.Password) != "" {
			userID, err := uuidFromPg(dbUser.ID)
			if err != nil {
				return fmt.Errorf("bootstrap admin %q user id: %w", email, err)
			}
			if err := adminAuth.UpsertLocalPassword(ctx, userID, email, user.Password); err != nil {
				return fmt.Errorf("bootstrap admin %q password: %w", email, err)
			}
		}

		if err := queries.SetUserSuperAdmin(ctx, db.SetUserSuperAdminParams{
			ID:           dbUser.ID,
			IsSuperAdmin: user.IsSuperAdmin(),
		}); err != nil {
			return fmt.Errorf("bootstrap admin %q super admin: %w", email, err)
		}
	}

	for _, key := range bootstrap.APIKeys {
		prefix := strings.TrimSpace(key.Prefix)
		if prefix == "" {
			continue
		}
		_, err := queries.GetAPIKeyByPrefix(ctx, prefix)
		notFound := false
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				notFound = true
			} else {
				return fmt.Errorf("bootstrap api key %q lookup: %w", prefix, err)
			}
		}

		tenantName := strings.TrimSpace(key.Tenant)
		if tenantName == "" {
			return fmt.Errorf("bootstrap api key %q missing tenant", prefix)
		}
		tenant, err := queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap api key %q tenant %q: %w", prefix, tenantName, err)
		}

		if notFound {
			hash, err := auth.HashPassword(strings.TrimSpace(key.Secret))
			if err != nil {
				return fmt.Errorf("bootstrap api key %q hash: %w", prefix, err)
			}

			if _, err := queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
				TenantID:    tenant.ID,
				Prefix:      prefix,
				SecretHash:  hash,
				Name:        strings.TrimSpace(key.Name),
				ScopesJson:  []byte("[]"),
				QuotaJson:   []byte("{}"),
				Kind:        db.ApiKeyKindService,
				OwnerUserID: pgtype.UUID{Valid: false},
			}); err != nil {
				return fmt.Errorf("bootstrap api key %q create: %w", prefix, err)
			}
		}
		keyLimits[prefix] = limitFromBootstrapRate(key.RateLimit)
	}

	for _, member := range bootstrap.Memberships {
		tenantName := strings.TrimSpace(member.Tenant)
		email := strings.TrimSpace(member.Email)
		roleValue := strings.TrimSpace(member.Role)
		if tenantName == "" || email == "" {
			continue
		}

		tenant, err := queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap membership tenant %q: %w", tenantName, err)
		}

		user, err := queries.GetUserByEmail(ctx, email)
		if err != nil {
			return fmt.Errorf("bootstrap membership user %q: %w", email, err)
		}

		role, ok := rbac.ParseRole(roleValue)
		if !ok {
			return fmt.Errorf("bootstrap membership role %q invalid", roleValue)
		}

		existing, err := queries.GetTenantMembership(ctx, db.GetTenantMembershipParams{
			TenantID: tenant.ID,
			UserID:   user.ID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if _, err := queries.AddTenantMembership(ctx, db.AddTenantMembershipParams{
					TenantID: tenant.ID,
					UserID:   user.ID,
					Role:     role,
				}); err != nil {
					return fmt.Errorf("bootstrap membership add %q/%q: %w", tenantName, email, err)
				}
			} else {
				return fmt.Errorf("bootstrap membership lookup %q/%q: %w", tenantName, email, err)
			}
		} else if existing.Role != role {
			if _, err := queries.UpdateTenantMembershipRole(ctx, db.UpdateTenantMembershipRoleParams{
				TenantID: tenant.ID,
				UserID:   user.ID,
				Role:     role,
			}); err != nil {
				return fmt.Errorf("bootstrap membership update %q/%q: %w", tenantName, email, err)
			}
		}
	}

	for _, limit := range bootstrap.TenantLimits {
		tenantName := strings.TrimSpace(limit.Tenant)
		if tenantName == "" {
			continue
		}
		tenant, err := queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap tenant limit %q: %w", tenantName, err)
		}
		tenantUUID, err := uuidFromPg(tenant.ID)
		if err != nil {
			return fmt.Errorf("bootstrap tenant limit %q id: %w", tenantName, err)
		}
		override := limitFromBootstrapRate(limit.Limits)
		tenantLimits[tenantUUID] = override
		if _, err := queries.UpsertTenantRateLimit(ctx, db.UpsertTenantRateLimitParams{
			TenantID:          tenant.ID,
			RequestsPerMinute: int32(limit.Limits.RequestsPerMinute),
			TokensPerMinute:   int32(limit.Limits.TokensPerMinute),
			ParallelRequests:  int32(limit.Limits.ParallelRequests),
		}); err != nil {
			return fmt.Errorf("bootstrap tenant limit %q upsert: %w", tenantName, err)
		}
	}

	for _, entry := range bootstrap.TenantBudgets {
		tenantName := strings.TrimSpace(entry.Tenant)
		if tenantName == "" {
			continue
		}
		tenant, err := queries.GetTenantByName(ctx, tenantName)
		if err != nil {
			return fmt.Errorf("bootstrap tenant budget %q: %w", tenantName, err)
		}

		budgetValue := budgetCfg.DefaultUSD
		if entry.BudgetUSD != nil {
			budgetValue = *entry.BudgetUSD
		}
		if budgetValue <= 0 {
			return fmt.Errorf("bootstrap tenant budget %q budget_usd must be > 0", tenantName)
		}

		warning := budgetCfg.WarningThresholdPerc
		if entry.WarningThreshold != nil {
			warning = *entry.WarningThreshold
		}

		refreshSchedule := budgetCfg.RefreshSchedule
		if strings.TrimSpace(entry.RefreshSchedule) != "" {
			refreshSchedule = config.NormalizeBudgetRefreshSchedule(entry.RefreshSchedule)
		}

		alertEmails := entry.AlertEmails
		alertWebhooks := entry.AlertWebhooks
		if len(alertEmails) == 0 && len(alertWebhooks) == 0 && budgetCfg.Alert.Enabled {
			alertEmails = budgetCfg.Alert.Emails
			alertWebhooks = budgetCfg.Alert.Webhooks
		}

		alertCooldown := budgetCfg.Alert.Cooldown
		if entry.AlertCooldown > 0 {
			alertCooldown = entry.AlertCooldown
		}
		if alertCooldown <= 0 {
			alertCooldown = time.Hour
		}

		cooldownSeconds := alertCooldown / time.Second
		if cooldownSeconds > math.MaxInt32 {
			cooldownSeconds = math.MaxInt32
		}

		params := db.UpsertTenantBudgetOverrideParams{
			TenantID:             tenant.ID,
			BudgetUsd:            decimal.NewFromFloat(budgetValue).Round(2),
			WarningThreshold:     decimal.NewFromFloat(warning),
			RefreshSchedule:      refreshSchedule,
			AlertEmails:          alertEmails,
			AlertWebhooks:        alertWebhooks,
			AlertCooldownSeconds: int32(cooldownSeconds),
		}

		if _, err := queries.UpsertTenantBudgetOverride(ctx, params); err != nil {
			return fmt.Errorf("bootstrap tenant budget %q upsert: %w", tenantName, err)
		}
	}

	return nil
}

func limitFromBootstrapRate(rate config.BootstrapRateLimit) limits.LimitConfig {
	return limits.LimitConfig{
		RequestsPerMinute: rate.RequestsPerMinute,
		TokensPerMinute:   rate.TokensPerMinute,
		ParallelRequests:  rate.ParallelRequests,
	}
}

func mergeLimitConfigs(base limits.LimitConfig, override limits.LimitConfig) limits.LimitConfig {
	if override.RequestsPerMinute > 0 {
		base.RequestsPerMinute = override.RequestsPerMinute
	}
	if override.TokensPerMinute > 0 {
		base.TokensPerMinute = override.TokensPerMinute
	}
	if override.ParallelRequests > 0 {
		base.ParallelRequests = override.ParallelRequests
	}
	return base
}

// EffectiveRateLimits returns the applied key-level and tenant-level configs
// without requiring a request context.
func (c *Container) EffectiveRateLimits(prefix string, tenantID uuid.UUID) (limits.LimitConfig, limits.LimitConfig) {
	keyCfg := c.DefaultKeyLimit
	if override, ok := c.KeyRateLimits[prefix]; ok {
		keyCfg = mergeLimitConfigs(keyCfg, override)
	}
	tenantCfg := c.DefaultTenantLimit
	c.tenantRateLimitMu.RLock()
	override, ok := c.TenantRateLimits[tenantID]
	c.tenantRateLimitMu.RUnlock()
	if ok {
		tenantCfg = mergeLimitConfigs(tenantCfg, override)
	}
	return keyCfg, tenantCfg
}

func (c *Container) ResolveRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, error) {
	rc, ok := requestctx.FromContext(ctx)
	if !ok || rc == nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, fmt.Errorf("request context missing")
	}

	keyCfg := c.DefaultKeyLimit
	if override, ok := c.KeyRateLimits[rc.APIKeyPrefix]; ok {
		keyCfg = mergeLimitConfigs(keyCfg, override)
	}
	tenantCfg := c.DefaultTenantLimit
	c.tenantRateLimitMu.RLock()
	override, ok := c.TenantRateLimits[rc.TenantID]
	c.tenantRateLimitMu.RUnlock()
	if ok {
		tenantCfg = mergeLimitConfigs(tenantCfg, override)
	}

	keyKey := fmt.Sprintf("%s:%s", rc.APIKeyPrefix, alias)
	tenantKey := rc.TenantID.String()
	return keyKey, keyCfg, tenantKey, tenantCfg, nil
}

func (c *Container) AcquireRateLimits(ctx context.Context, alias string) (string, limits.LimitConfig, string, limits.LimitConfig, func(), error) {
	keyKey, keyCfg, tenantKey, tenantCfg, err := c.ResolveRateLimits(ctx, alias)
	if err != nil {
		return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
	}

	keyAcquired := false
	tenantAcquired := false

	keyStorage := "key:" + keyKey
	tenantStorage := "tenant:" + tenantKey

	if keyCfg.RequestsPerMinute > 0 || keyCfg.ParallelRequests > 0 {
		if err := c.RateLimiter.Allow(ctx, keyStorage, keyCfg); err != nil {
			return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
		}
		keyAcquired = true
	}

	if tenantCfg.RequestsPerMinute > 0 || tenantCfg.ParallelRequests > 0 {
		if err := c.RateLimiter.Allow(ctx, tenantStorage, tenantCfg); err != nil {
			if keyAcquired {
				c.RateLimiter.Release(ctx, keyStorage, keyCfg)
			}
			return "", limits.LimitConfig{}, "", limits.LimitConfig{}, nil, err
		}
		tenantAcquired = true
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			if tenantAcquired {
				c.RateLimiter.Release(ctx, tenantStorage, tenantCfg)
			}
			if keyAcquired {
				c.RateLimiter.Release(ctx, keyStorage, keyCfg)
			}
		})
	}

	return keyKey, keyCfg, tenantKey, tenantCfg, release, nil
}

func uuidFromPg(id pgtype.UUID) (uuid.UUID, error) {
	if !id.Valid {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return uuid.FromBytes(id.Bytes[:])
}

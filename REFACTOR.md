# Open Model Gateway Refactoring Plan

This document outlines a comprehensive refactoring strategy to improve maintainability, testability, and extensibility of the Open Model Gateway codebase.

## Executive Summary

The codebase demonstrates solid architectural fundamentals (dependency injection, interfaces, middleware chains) but shows signs of growth-driven complexity. Key issues include:

- **Backend**: Container "god object", config bloat, duplicated HTTP pipelines, inconsistent service patterns
- **Frontend**: Large page components (1000+ lines), duplicated API modules, repeated table/dialog patterns, minimal test coverage

Estimated effort reduction after refactoring: ~2000 lines of duplicated code eliminated.

---

## Current Architecture Overview

### Backend Statistics
| Metric | Value |
|--------|-------|
| Total Go Files | 281 |
| Lines of Code | ~55K |
| Service Packages | 17 |
| Provider Adapters | 8 |
| Internal Packages | 27 |
| Test Coverage | ~13.5% of files |

### Frontend Statistics
| Metric | Value |
|--------|-------|
| Total TypeScript Files | 176 |
| Lines of Code | ~25K |
| Feature Modules | 7 |
| Largest Components | 1124 lines |
| Test Coverage | <1% |

---

## Phase 1: Foundation (Backend Core)

### 1.1 Split Container God Object

**Problem**: `app/container.go` (476 lines, 40+ fields) violates Single Responsibility Principle. It holds all services, providers, caches, and runtime state in one struct.

**Solution**: Decompose into themed sub-containers:

```go
type Container struct {
    Services   *ServiceContainer   // Business logic services
    Routing    *RoutingContainer   // Provider factory, router engine
    Data       *DataContainer      // DB pool, Redis, queries
    Telemetry  *TelemetryContainer // OTEL, metrics, health
    Config     *config.Config      // Read-only configuration
}

type ServiceContainer struct {
    Tenant   services.TenantService
    Usage    services.UsageService
    Billing  services.BillingService
    // ...
}

type RoutingContainer struct {
    Factory  *providers.Factory
    Engine   *router.Engine
    Health   *health.Monitor
}
```

**Benefits**:
- Clear dependency boundaries
- Easier to mock in tests
- Reduced cognitive load when navigating code

### 1.2 Decompose Config Struct

**Problem**: `config.go` (987 lines) contains 200+ nested fields across 10+ section types.

**Solution**: Split into domain-specific config packages:

```
internal/config/
├── config.go       # Root Config struct, loader
├── server.go       # Server, timeouts, body limits
├── database.go     # Database connection config
├── redis.go        # Redis config
├── providers.go    # All provider configs
├── features.go     # Feature flags, budgets, limits
├── bootstrap.go    # Bootstrap seeding config
└── validation.go   # Cross-field validation
```

**Benefits**:
- Easier to document individual sections
- Reduced merge conflicts
- Provider configs can be extended independently

### 1.3 Extract Base HTTP Pipeline

**Problem**: 8 pipeline files in `httpserver/public/` share 50-100 lines of boilerplate each (auth, validation, idempotency, error handling).

**Solution**: Create a generic pipeline abstraction:

```go
type Pipeline[Req any, Resp any] struct {
    container  *app.Container
    executor   *executor.Executor
    validator  func(Req) error
    execute    func(ctx context.Context, req Req) (Resp, error)
    converter  func(Resp) (any, error)
}

func (p *Pipeline[Req, Resp]) Handle(c *fiber.Ctx) error {
    // 1. Parse request
    // 2. Validate
    // 3. Check idempotency cache
    // 4. Execute
    // 5. Convert response
    // 6. Cache result
    // 7. Return response
}
```

**Benefits**:
- DRY principle enforced
- Consistent error handling
- Easy to add new operations

### 1.4 Define Service Interfaces

**Problem**: Services directly depend on `*db.Queries` and `*config.Config`, making unit testing difficult.

**Solution**: Define interfaces for each service:

```go
// internal/services/tenant/interface.go
type TenantService interface {
    GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
    Create(ctx context.Context, req CreateTenantRequest) (*Tenant, error)
    Update(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) error
    Delete(ctx context.Context, id uuid.UUID) error
    ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]APIKey, error)
}

// internal/services/tenant/service.go
type service struct {
    repo   TenantRepository  // Interface, not *db.Queries
    config TenantConfig      // Subset of config
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
    return s.repo.GetTenant(ctx, id)
}
```

**Benefits**:
- Services testable with mock repositories
- Clear contracts between layers
- Easier to add caching layers

---

## Phase 2: Backend Improvements

### 2.1 Standardize Error Handling

**Problem**: Custom `apiError` type used inconsistently. Stack traces and context lost.

**Solution**: Use wrapped errors with sentinel types:

```go
// internal/apperror/errors.go
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrRateLimited   = errors.New("rate limited")
    ErrBudgetExceeded = errors.New("budget exceeded")
)

type Error struct {
    Op      string  // Operation that failed
    Code    string  // Machine-readable code
    Message string  // Human-readable message
    Err     error   // Wrapped error
}

func (e *Error) Unwrap() error { return e.Err }

// Usage
return &apperror.Error{
    Op:      "tenant.Create",
    Code:    "DUPLICATE_NAME",
    Message: "tenant with this name already exists",
    Err:     ErrConflict,
}
```

**Benefits**:
- Consistent error handling across codebase
- Errors can be logged with full context
- HTTP status mapping centralized

### 2.2 Extract Common Adapter Patterns

**Problem**: 8 adapters with significant implementation variation (401-1259 lines each).

**Solution**: Create base adapter with common functionality:

```go
// internal/adapters/base/adapter.go
type BaseAdapter struct {
    client     *http.Client
    baseURL    string
    apiKey     string
    logger     *slog.Logger
    retryCount int
}

func (b *BaseAdapter) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Common retry logic, logging, error handling
}

func (b *BaseAdapter) ParseError(resp *http.Response) error {
    // Common error parsing
}

// Adapters embed base
type OpenAIAdapter struct {
    base.BaseAdapter
    orgID string
}
```

**Benefits**:
- Consistent retry and error handling
- Reduced code in individual adapters
- Easier to add new providers

### 2.3 Improve Repository Pattern

**Problem**: SQLC-generated `*db.Queries` used directly in services. No abstraction for testing.

**Solution**: Create repository interfaces wrapping SQLC:

```go
// internal/repository/tenant.go
type TenantRepository interface {
    Get(ctx context.Context, id uuid.UUID) (*db.Tenant, error)
    List(ctx context.Context, opts ListOptions) ([]db.Tenant, error)
    Create(ctx context.Context, params db.CreateTenantParams) (*db.Tenant, error)
    Update(ctx context.Context, id uuid.UUID, params db.UpdateTenantParams) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type tenantRepo struct {
    queries *db.Queries
}

func (r *tenantRepo) Get(ctx context.Context, id uuid.UUID) (*db.Tenant, error) {
    return r.queries.GetTenant(ctx, id)
}
```

**Benefits**:
- Services depend on interfaces, not implementations
- Easy to mock for unit tests
- Can add caching layer without changing services

### 2.4 Standardize Logging

**Problem**: Mixed use of Fiber logger and `log/slog`. Inconsistent structured logging.

**Solution**: Standardize on `slog` with structured fields:

```go
// internal/logging/logger.go
func NewLogger(cfg LogConfig) *slog.Logger {
    var handler slog.Handler
    if cfg.Format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Level})
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.Level})
    }
    return slog.New(handler)
}

// Usage in services
s.logger.Info("tenant created",
    slog.String("tenant_id", tenant.ID.String()),
    slog.String("name", tenant.Name),
)
```

**Benefits**:
- Consistent log format
- Structured fields for log aggregation
- Level-based filtering

---

## Phase 3: Frontend Foundation

### 3.1 Break Down Large Page Components

**Problem**: Page components exceed 1000 lines (TenantsPage: 1124, ApiKeysPage: 1088).

**Solution**: Extract into feature-specific sub-components:

```
pages/TenantsPage/
├── index.tsx              # Main page (orchestration only)
├── TenantsList.tsx        # List/table component
├── TenantFilters.tsx      # Filter controls
├── TenantDialogs.tsx      # All dialog components
├── useTenantData.ts       # Data fetching hooks
└── useTenantActions.ts    # Mutation hooks
```

**Target**: No component should exceed 300 lines.

### 3.2 Unify API Layer

**Problem**: Parallel admin (`/api/`) and user (`/api/user/`) implementations with duplicated logic.

**Solution**: Single API module with role-based paths:

```typescript
// api/tenants.ts
export function createTenantsApi(role: 'admin' | 'user') {
    const basePath = role === 'admin' ? '/admin' : '/user';
    const client = role === 'admin' ? adminClient : userClient;

    return {
        list: (params: ListParams) => client.get(`${basePath}/tenants`, { params }),
        get: (id: string) => client.get(`${basePath}/tenants/${id}`),
        create: (data: CreateTenantRequest) => client.post(`${basePath}/tenants`, data),
        update: (id: string, data: UpdateTenantRequest) => client.put(`${basePath}/tenants/${id}`, data),
        delete: (id: string) => client.delete(`${basePath}/tenants/${id}`),
    };
}

// Usage
const adminTenantsApi = createTenantsApi('admin');
const userTenantsApi = createTenantsApi('user');
```

**Benefits**:
- Single source of truth for API contracts
- Reduced duplication (~400 lines)
- Easier to maintain type safety

### 3.3 Extract Reusable Table Component

**Problem**: Table patterns duplicated across 5+ components (AdminKeyTable, AdminBatchTable, AdminFilesTable, UserBatchTable, UserFilesTable).

**Solution**: Create generic `AdminTable` component:

```typescript
// components/tables/AdminTable.tsx
interface AdminTableProps<T> {
    data: T[];
    columns: ColumnDef<T>[];
    isLoading: boolean;
    filters?: FilterConfig[];
    onFilterChange?: (filters: FilterState) => void;
    actions?: (item: T) => ReactNode;
    emptyState?: EmptyStateConfig;
    pagination?: PaginationConfig;
}

function AdminTable<T>({ data, columns, isLoading, ... }: AdminTableProps<T>) {
    if (isLoading) return <TableSkeleton columns={columns.length} />;
    if (data.length === 0) return <EmptyState {...emptyState} />;

    return (
        <div>
            {filters && <FilterBar filters={filters} onChange={onFilterChange} />}
            <DataTable columns={columns} data={data} />
            {pagination && <Pagination {...pagination} />}
        </div>
    );
}
```

**Benefits**:
- Consistent table UX across app
- Single place to update table styling
- Reduced code by ~300 lines

### 3.4 Create FormDialog Abstraction

**Problem**: Dialog patterns repeated across 12+ dialogs (header, footer, form layout, button groups).

**Solution**: Extract `FormDialog` wrapper:

```typescript
// components/dialogs/FormDialog.tsx
interface FormDialogProps<T> {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    description?: string;
    form: UseFormReturn<T>;
    onSubmit: (data: T) => Promise<void>;
    submitLabel?: string;
    children: ReactNode;
}

function FormDialog<T>({ open, onOpenChange, title, form, onSubmit, children }: FormDialogProps<T>) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    {description && <DialogDescription>{description}</DialogDescription>}
                </DialogHeader>
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)}>
                        {children}
                        <DialogFooter>
                            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                                Cancel
                            </Button>
                            <Button type="submit" disabled={form.formState.isSubmitting}>
                                {submitLabel ?? 'Save'}
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
```

**Benefits**:
- Consistent dialog UX
- Automatic form handling
- Reduced boilerplate by ~200 lines

### 3.5 Consolidate Utility Functions

**Problem**: `dateFormatter` defined in 5 files, `formatBytes` duplicated, date/number utilities scattered.

**Solution**: Centralize in `/lib/`:

```typescript
// lib/formatters.ts
export const formatDate = (date: string | Date, format: 'short' | 'long' | 'relative' = 'short') => {
    // Implementation
};

export const formatBytes = (bytes: number, decimals = 2) => {
    // Implementation
};

export const formatCurrency = (amount: number, currency = 'USD') => {
    // Implementation
};

export const formatNumber = (num: number, options?: Intl.NumberFormatOptions) => {
    // Implementation
};
```

**Benefits**:
- Single source of truth
- Consistent formatting across app
- Easy to update/localize

---

## Phase 4: Frontend Improvements

### 4.1 Add Error Boundaries

**Problem**: No error UI fallbacks. Unhandled errors crash entire app.

**Solution**: Add error boundaries at page and feature level:

```typescript
// components/ErrorBoundary.tsx
function ErrorBoundary({ children, fallback }: ErrorBoundaryProps) {
    return (
        <ReactErrorBoundary
            fallbackRender={({ error, resetErrorBoundary }) => (
                fallback ?? <ErrorFallback error={error} onReset={resetErrorBoundary} />
            )}
        >
            {children}
        </ReactErrorBoundary>
    );
}

// Usage in App.tsx
<ErrorBoundary>
    <Suspense fallback={<PageSkeleton />}>
        <RouterProvider router={router} />
    </Suspense>
</ErrorBoundary>
```

### 4.2 Optimize Query Loading

**Problem**: Some pages load 10+ queries sequentially (waterfall pattern).

**Solution**: Use parallel queries and Suspense:

```typescript
// Before (sequential)
const { data: profile } = useQuery({ queryKey: ['profile'] });
const { data: keys } = useQuery({ queryKey: ['keys'] });
const { data: usage } = useQuery({ queryKey: ['usage'] });

// After (parallel with suspense)
const queries = useSuspenseQueries({
    queries: [
        { queryKey: ['profile'], queryFn: fetchProfile },
        { queryKey: ['keys'], queryFn: fetchKeys },
        { queryKey: ['usage'], queryFn: fetchUsage },
    ],
});
```

### 4.3 Consider OpenAPI Codegen

**Problem**: Manual type definitions for API contracts risk drift from backend.

**Solution**: Generate types from OpenAPI spec:

```bash
# Add to package.json scripts
"generate:api": "openapi-typescript ../openapi/openapi.yaml -o src/api/generated.ts"
```

**Benefits**:
- Types always match backend
- Reduced manual work
- Catch breaking changes early

### 4.4 Add Test Coverage

**Problem**: <1% test coverage (only 2 test files).

**Solution**: Prioritize testing:

1. **Unit tests for utilities** (formatters, validators)
2. **Component tests for reusable UI** (AdminTable, FormDialog)
3. **Integration tests for pages** (render + mock API)

Target: 50% coverage on shared components, 30% on pages.

---

## Phase 5: Cross-Cutting Improvements

### 5.1 Add Integration Tests

- Backend: HTTP handler tests with test DB
- Frontend: Playwright/Cypress E2E tests for critical flows
- API contract tests (consumer-driven contracts)

### 5.2 Documentation

- Architecture Decision Records (ADRs) for major patterns
- API documentation (already have OpenAPI spec)
- Component Storybook for UI library
- Developer onboarding guide

### 5.3 Performance Monitoring

- Add Lighthouse CI for frontend performance
- Add query performance logging for slow DB queries
- Monitor bundle size with bundlewatch

---

## Migration Strategy

### Principles

1. **Incremental changes**: Each refactor should be a standalone PR
2. **Backwards compatible**: No breaking changes to API contracts
3. **Feature flagged**: Major changes behind feature flags
4. **Test before refactor**: Add tests before changing code
5. **Measure impact**: Track metrics before/after

### Suggested Order

1. **Backend Phase 1** (Foundation) - Split container, config
2. **Frontend Phase 3** (Foundation) - Break down pages, unify API
3. **Backend Phase 2** (Improvements) - Error handling, repositories
4. **Frontend Phase 4** (Improvements) - Error boundaries, tests
5. **Cross-cutting** (Phase 5) - Integration tests, docs

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Backend test coverage | 13.5% | 40% |
| Frontend test coverage | <1% | 30% |
| Largest component (lines) | 1124 | 300 |
| Container fields | 40+ | 10 (per sub-container) |
| Config struct fields | 200+ | 50 (per section) |
| Duplicated code | ~2000 lines | <500 lines |
| Build time | - | Measure baseline |
| Time to add new provider | - | <2 hours |
| Time to add new API endpoint | - | <1 hour |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Regression during refactor | Add tests before refactoring |
| Team velocity impact | Spread work across sprints |
| Scope creep | Strict PR scoping, no feature work mixed in |
| Breaking changes | Feature flags, semantic versioning |
| Knowledge silos | Document decisions, pair programming |

---

## Appendix: Key Files to Understand

### Backend
- `internal/app/container.go` - Current DI container
- `internal/runtime/builder.go` - Startup orchestration
- `internal/providers/factory.go` - Provider initialization
- `internal/httpserver/public/*.go` - HTTP pipelines
- `internal/config/config.go` - Configuration schema

### Frontend
- `src/apps/admin/App.tsx` - Admin app entry
- `src/apps/user/App.tsx` - User app entry
- `src/pages/TenantsPage.tsx` - Largest page component
- `src/api/` - API client modules
- `src/features/` - Feature-scoped components

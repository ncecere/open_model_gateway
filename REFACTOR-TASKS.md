# Refactoring Tasks

Detailed task list for the Open Model Gateway refactoring effort. Tasks are organized by phase and priority.

---

## Phase 1: Backend Foundation

### 1.1 Split Container God Object

- [x] Create `internal/app/containers/` directory structure
- [x] Define `ServiceContainer` interface and struct with tenant, usage, billing services
- [x] Define `RoutingContainer` interface and struct with factory, engine, health
- [x] Define `DataContainer` interface and struct with DB pool, Redis, queries
- [x] Define `TelemetryContainer` interface and struct with OTEL, metrics
- [x] Define `RateLimitContainer` interface and struct with rate limiting state
- [x] Refactor `Container` to compose the new sub-containers
- [x] Update `runtime/builder.go` to construct sub-containers in stages
- [x] Update all imports of `app.Container` to use appropriate sub-container
- [x] Update `httpserver/public/*.go` to use new container structure
- [x] Update `httpserver/admin/*.go` to use new container structure
- [x] Update `httpserver/user/*.go` to use new container structure
- [x] Add tests for container construction and dependency resolution
- [x] Document new container architecture in code comments

### 1.2 Decompose Config Struct

**Note:** Upon review, the config is already well-organized with nested structs (ServerConfig, DatabaseConfig, RedisConfig, etc.) in config.go. File splitting is optional for navigation convenience.

- [x] Create `ServerConfig` struct (already exists)
- [x] Create `DatabaseConfig` struct (already exists)
- [x] Create `RedisConfig` struct (already exists)
- [x] Create provider config structs (already exist)
- [x] Create feature/budget/limit config structs (already exist)
- [x] Create bootstrap config structs (already exist)
- [x] Config validation is already implemented
- [x] Main `Config` struct already composes sub-sections
- [x] Viper bindings already work with nested structure
- [x] Environment variable mappings already work
- [x] (Optional) Split config types into separate files for navigation
- [x] (Optional) Add more config loading tests

### 1.3 Extract Base HTTP Pipeline

**Note:** Existing pipelines vary in complexity (executor-based vs. direct route handling). Created base utilities package to reduce boilerplate without over-abstracting.

- [x] Create `internal/httpserver/pipeline/` package
- [x] Define `Base` struct with common fields (Container, Executor, Idempotency)
- [x] Implement common validation methods (ValidateAlias, CheckRoutes)
- [x] Extract idempotency cache logic into pipeline base (CheckIdempotency, CacheResponse)
- [x] Extract response serialization logic (SendJSONResponse, SendJSONWithIdempotency)
- [x] Extract error handling logic (HandleExecutorError)
- [x] Create generic result handler with `ExecuteAndHandle`
- [x] Add tests for base pipeline behavior
- [x] (Optional) Refactor `chat_pipeline.go` to use base pipeline utilities
- [x] (Optional) Refactor `embedding_pipeline.go` to use base pipeline utilities
- [x] (Optional) Refactor `image_pipeline.go` to use base pipeline utilities
- [x] (Optional) Refactor `moderation_pipeline.go` to use base pipeline utilities
- [x] (Optional) Document pipeline pattern and extension points

### 1.4 Define Service Interfaces

- [x] Create `internal/services/interfaces.go` with all service interfaces
- [x] Create `internal/repository/` package for repository interfaces
- [x] Define `TenantRepository` interface wrapping SQLC tenant queries
- [x] Define `UsageRepository` interface wrapping SQLC usage queries
- [x] Define `APIKeyRepository` interface wrapping SQLC key queries
- [x] Define `BudgetRepository` interface wrapping SQLC budget queries
- [x] Define `RateLimitRepository` interface wrapping SQLC rate limit queries
- [x] Define `AuditRepository` interface wrapping SQLC audit queries
- [x] Define `MembershipRepository` interface wrapping SQLC membership queries
- [x] Define `UserRepository` interface wrapping SQLC user queries
- [x] Define `ModelCatalogRepository` interface wrapping SQLC model catalog queries
- [x] Implement repository structs wrapping `*db.Queries` (QueriesAdapter)
- [x] Update `admintenant.Service` to depend on `TenantRepository` interface
- [x] Update `usage.Service` to depend on `UsageRepository` interface
- [x] Update `adminapikeys.Service` to depend on `APIKeyRepository` interface
- [x] Update `adminbudget.Service` to depend on `BudgetRepository` interface
- [x] Update `adminratelimit.Service` to depend on repository interface
- [x] Update `admincatalog.Service` to depend on repository interface
- [x] Update `adminrbac.Service` to depend on `rbac.Repository` interface
- [x] Update `audit.Service` to depend on repository interface
- [x] Update `tenant.Service` to depend on repository interface
- [x] Update `adminuser.Service` to depend on repository interface
- [x] Update `adminconfig.Service` to depend on repository interface
- [x] Update `batches.Service` to depend on repository interface (with transaction support)
- [x] Update `exports.Service` to depend on repository interface
- [x] Update `billinghooks.Service` to depend on repository interface
- [x] Update `usagepipeline` services (Logger, UsageRecorder, BudgetEvaluator, AlertDispatcher)
- [x] Create mock implementations of repositories for testing
- [ ] Add unit tests for services using mock repositories

---

## Phase 2: Backend Improvements

### 2.1 Standardize Error Handling

- [x] Create `internal/apperror/` package
- [x] Define sentinel errors (ErrNotFound, ErrUnauthorized, ErrRateLimited, etc.)
- [x] Define `Error` struct with Op, Code, Message, Err fields
- [x] Implement `Unwrap()` method for error unwrapping
- [x] Implement `Error()` method with formatted output
- [x] Create helper functions `Is()`, `As()`, `Wrap()`
- [x] Create HTTP status mapping function `StatusCode(err error) int`
- [x] Update `httputil.WriteError()` to use new error types (added WriteAppError functions)
- [ ] Refactor `executor/executor.go` to return wrapped errors
- [ ] Refactor adapter error handling to use wrapped errors
- [ ] Refactor service error handling to use wrapped errors
- [ ] Update HTTP handlers to use centralized error mapping
- [ ] Add error logging with full context
- [x] Add tests for error handling and status mapping

### 2.2 Extract Common Adapter Patterns

- [x] Create `internal/adapters/base/` package
- [x] Define `BaseAdapter` struct with common fields (client, baseURL, apiKey, etc.)
- [x] Implement common `Do()` method with retry logic
- [x] Implement common `DecodeError()` method for HTTP errors
- [x] Implement common request building helpers (NewRequest, Send, DoJSON)
- [x] Implement common response parsing helpers
- [ ] Refactor `openai/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `anthropic/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `azureopenai/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `bedrock/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `vertex/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `groq/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `openrouter/adapter.go` to embed `BaseAdapter`
- [ ] Refactor `vllm/adapter.go` to embed `BaseAdapter`
- [x] Add tests for base adapter behavior
- [ ] Document adapter extension pattern

### 2.3 Standardize Logging

- [x] Create `internal/logging/` package
- [x] Implement `NewLogger()` factory with configurable format (JSON/text)
- [x] Implement `WithContext()` for request-scoped logging
- [x] Add structured field helpers for common fields (tenant_id, request_id, etc.)
- [ ] Update `runtime/builder.go` to initialize logger early
- [ ] Update container to hold logger instance
- [ ] Replace Fiber logger usage with slog in HTTP handlers
- [ ] Update all services to use structured logging
- [ ] Update adapters to use structured logging
- [ ] Add request/response logging middleware
- [ ] Add config options for log level and format
- [x] Test log output format in different modes

### 2.4 Improve Type Safety

- [ ] Define typed metadata structs for provider routes
- [ ] Define typed options structs for model capabilities
- [ ] Replace `map[string]string` metadata with typed structs
- [ ] Add validation for metadata fields
- [ ] Update adapters to use typed metadata
- [ ] Update router to use typed metadata
- [ ] Add compile-time checks for metadata keys
- [ ] Document metadata schema

---

## Phase 3: Frontend Foundation

### 3.1 Break Down Large Page Components

#### TenantsPage (User Portal)
- [ ] Extract `TenantsList` component from TenantsPage
- [ ] Extract `TenantFilters` component from TenantsPage
- [ ] Extract `TenantDialogs` component (create/edit/delete dialogs)
- [ ] Create `useTenantData.ts` hook for data fetching
- [ ] Create `useTenantActions.ts` hook for mutations
- [ ] Refactor TenantsPage to orchestrate sub-components
- [ ] Ensure TenantsPage is under 300 lines

#### ApiKeysPage (User Portal)
- [ ] Extract `ApiKeysList` component from ApiKeysPage
- [ ] Extract `ApiKeyFilters` component from ApiKeysPage
- [ ] Extract `ApiKeyDialogs` component (create/edit/delete dialogs)
- [ ] Create `useApiKeyData.ts` hook for data fetching
- [ ] Create `useApiKeyActions.ts` hook for mutations
- [ ] Refactor ApiKeysPage to orchestrate sub-components
- [ ] Ensure ApiKeysPage is under 300 lines

#### TenantsPage (Admin Portal)
- [ ] Extract `AdminTenantsList` component
- [ ] Extract `AdminTenantFilters` component
- [ ] Extract `AdminTenantDialogs` component
- [ ] Create `useAdminTenantData.ts` hook
- [ ] Create `useAdminTenantActions.ts` hook
- [ ] Refactor admin TenantsPage to under 300 lines

#### KeysPage (Admin Portal)
- [ ] Extract `AdminKeysList` component from KeysPage
- [ ] Extract `AdminKeyFilters` component from KeysPage
- [ ] Extract `AdminKeyDialogs` component
- [ ] Create `useAdminKeyData.ts` hook
- [ ] Create `useAdminKeyActions.ts` hook
- [ ] Refactor KeysPage to under 300 lines

### 3.2 Unify API Layer

- [ ] Create `api/createApi.ts` factory function for role-based API creation
- [ ] Create unified `api/tenants.ts` using factory pattern
- [ ] Create unified `api/apiKeys.ts` using factory pattern
- [ ] Create unified `api/batches.ts` using factory pattern
- [ ] Create unified `api/files.ts` using factory pattern
- [ ] Create unified `api/usage.ts` using factory pattern
- [ ] Create unified `api/budgets.ts` using factory pattern
- [ ] Create unified `api/models.ts` using factory pattern
- [ ] Remove duplicate `api/user/*.ts` files
- [ ] Update admin app imports to use unified API
- [ ] Update user app imports to use unified API
- [ ] Add TypeScript types for all API requests/responses
- [ ] Add API error handling utilities
- [ ] Test API layer with mock server

### 3.3 Extract Reusable Table Component

- [ ] Create `components/tables/` directory
- [ ] Create `TableSkeleton` component for loading states
- [ ] Create `EmptyState` component for empty data
- [ ] Create `FilterBar` component for table filters
- [ ] Create `ActionsMenu` component for row actions
- [ ] Create `AdminTable<T>` generic component
- [ ] Add pagination support to AdminTable
- [ ] Add sorting support to AdminTable
- [ ] Add column visibility toggle to AdminTable
- [ ] Refactor `AdminKeyTable` to use AdminTable
- [ ] Refactor `AdminBatchTable` to use AdminTable
- [ ] Refactor `AdminFilesTable` to use AdminTable
- [ ] Refactor `UserBatchTable` to use AdminTable
- [ ] Refactor `UserFilesTable` to use AdminTable
- [ ] Add Storybook stories for table components
- [ ] Add tests for table components

### 3.4 Create FormDialog Abstraction

- [ ] Create `components/dialogs/` directory
- [ ] Create `BaseDialog` component with standard header/footer
- [ ] Create `FormDialog` component integrating react-hook-form
- [ ] Create `ConfirmDialog` component for delete confirmations
- [ ] Create `AlertDialog` component for warnings
- [ ] Add form validation display to FormDialog
- [ ] Add loading state handling to FormDialog
- [ ] Refactor `TenantCreateDialog` to use FormDialog
- [ ] Refactor `TenantEditDialog` to use FormDialog
- [ ] Refactor `BatchDetailsDialog` to use BaseDialog
- [ ] Refactor `FileDetailsDialog` to use BaseDialog
- [ ] Refactor `ApiKeyCreateDialog` to use FormDialog
- [ ] Refactor `ApiKeyEditDialog` to use FormDialog
- [ ] Add Storybook stories for dialog components
- [ ] Add tests for dialog components

### 3.5 Consolidate Utility Functions

- [ ] Create `lib/formatters.ts` with all formatting utilities
- [ ] Implement `formatDate()` with multiple format options
- [ ] Implement `formatBytes()` for file sizes
- [ ] Implement `formatCurrency()` for monetary values
- [ ] Implement `formatNumber()` with locale support
- [ ] Implement `formatDuration()` for time durations
- [ ] Remove duplicate `dateFormatter` from `batches/utils.ts`
- [ ] Remove duplicate `dateFormatter` from `files/utils.ts`
- [ ] Remove duplicate `dateFormatter` from `AdminKeyTable.tsx`
- [ ] Update all components to use centralized formatters
- [ ] Add tests for formatting functions
- [ ] Add i18n support to formatters (future)

---

## Phase 4: Frontend Improvements

### 4.1 Add Error Boundaries

- [ ] Install `react-error-boundary` package
- [ ] Create `ErrorFallback` component with reset button
- [ ] Create `ErrorBoundary` wrapper component
- [ ] Create `PageErrorBoundary` for page-level errors
- [ ] Create `FeatureErrorBoundary` for feature-level errors
- [ ] Add error boundary to admin app root
- [ ] Add error boundary to user app root
- [ ] Add error boundaries around major features
- [ ] Add error logging to error boundaries
- [ ] Test error boundary behavior

### 4.2 Optimize Query Loading

- [ ] Audit all pages for query waterfall patterns
- [ ] Convert sequential queries to `useQueries` parallel loading
- [ ] Add Suspense boundaries for data loading
- [ ] Create `PageSkeleton` component for Suspense fallback
- [ ] Create `FeatureSkeleton` components for feature loading
- [ ] Update TenantsPage to use parallel queries
- [ ] Update ApiKeysPage to use parallel queries
- [ ] Update DashboardPage to use parallel queries
- [ ] Add React Query devtools for development
- [ ] Measure and document query performance improvements

### 4.3 Consider OpenAPI Codegen

- [ ] Evaluate OpenAPI codegen tools (openapi-typescript, orval, etc.)
- [ ] Create proof-of-concept with one API endpoint
- [ ] Set up codegen script in package.json
- [ ] Generate types from OpenAPI spec
- [ ] Update one API module to use generated types
- [ ] Compare manual vs generated types for drift
- [ ] Document codegen workflow
- [ ] Roll out to remaining API modules (if POC successful)

### 4.4 Add Test Coverage

#### Unit Tests
- [ ] Set up Vitest configuration
- [ ] Add tests for `lib/formatters.ts`
- [ ] Add tests for `lib/utils.ts`
- [ ] Add tests for `lib/validators.ts` (if exists)
- [ ] Add tests for form validation logic

#### Component Tests
- [ ] Set up Testing Library
- [ ] Add tests for `AdminTable` component
- [ ] Add tests for `FormDialog` component
- [ ] Add tests for `ErrorBoundary` component
- [ ] Add tests for `FilterBar` component
- [ ] Add tests for `TableSkeleton` component

#### Integration Tests
- [ ] Set up MSW for API mocking
- [ ] Add integration test for TenantsPage
- [ ] Add integration test for ApiKeysPage
- [ ] Add integration test for login flow
- [ ] Add integration test for key creation flow

#### E2E Tests
- [ ] Evaluate Playwright vs Cypress
- [ ] Set up E2E test infrastructure
- [ ] Add E2E test for admin login
- [ ] Add E2E test for tenant creation
- [ ] Add E2E test for API key management

---

## Phase 5: Cross-Cutting Improvements

### 5.1 Backend Integration Tests

- [ ] Create `test/integration/` directory structure
- [ ] Set up test database provisioning
- [ ] Create test fixtures for common data
- [ ] Add HTTP handler integration tests
- [ ] Add service integration tests with real DB
- [ ] Add provider adapter integration tests (with mocks)
- [ ] Set up CI pipeline for integration tests
- [ ] Add test coverage reporting

### 5.2 Documentation

- [ ] Create `docs/architecture/` directory for ADRs
- [ ] Write ADR for container decomposition
- [ ] Write ADR for service interface pattern
- [ ] Write ADR for error handling approach
- [ ] Create component Storybook for UI library
- [ ] Add Storybook deployment to CI
- [ ] Create developer onboarding guide
- [ ] Update README with new architecture overview
- [ ] Document API contracts beyond OpenAPI spec

### 5.3 Performance Monitoring

- [ ] Add Lighthouse CI to frontend build
- [ ] Set performance budgets (bundle size, FCP, TTI)
- [ ] Add slow query logging to database layer
- [ ] Add query timing to observability metrics
- [ ] Set up alerting for performance regressions
- [ ] Add bundle analysis to CI (bundlewatch or similar)
- [ ] Document performance baseline metrics

### 5.4 CI/CD Improvements

- [ ] Add pre-commit hooks for linting
- [ ] Add type checking to CI pipeline
- [ ] Add test coverage thresholds
- [ ] Add breaking change detection for API
- [ ] Set up staging environment for testing
- [ ] Add deployment previews for PRs

---

## Cleanup Tasks

### Remove Dead Code
- [ ] Audit and remove unused exports
- [ ] Remove commented-out code
- [ ] Remove unused dependencies
- [ ] Remove unused CSS classes
- [ ] Remove unused type definitions

### Code Style Consistency
- [ ] Run and fix ESLint on frontend
- [ ] Run and fix golangci-lint on backend
- [ ] Standardize import ordering
- [ ] Standardize file naming conventions
- [ ] Add missing TypeScript strict checks

---

## Maintenance Tasks (Ongoing)

- [ ] Review and update dependencies quarterly
- [ ] Review and update Go version as needed
- [ ] Review and update Node.js version as needed
- [ ] Review security advisories for dependencies
- [ ] Update documentation with each major change
- [ ] Review and clean up TODO comments in code

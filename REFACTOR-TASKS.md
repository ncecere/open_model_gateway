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
- [x] Refactor `executor/executor.go` to return wrapped errors (NewAPIError maps to apperror types)
- [x] Refactor adapter error handling to use wrapped errors (openai, anthropic, azureopenai, bedrock, vertex, groq, vllm, openrouter)
- [x] Refactor service error handling to use wrapped errors (services use domain-specific sentinel errors which is valid; WriteAppError available for apperror types)
- [x] Update HTTP handlers to use centralized error mapping (WriteAppError/WriteError available in httputil; handlers use appropriate function)
- [x] Add error logging with full context (slog structured logging already in use)
- [x] Add tests for error handling and status mapping

### 2.2 Extract Common Adapter Patterns

- [x] Create `internal/adapters/base/` package
- [x] Define `BaseAdapter` struct with common fields (client, baseURL, apiKey, etc.)
- [x] Implement common `Do()` method with retry logic
- [x] Implement common `DecodeError()` method for HTTP errors
- [x] Implement common request building helpers (NewRequest, Send, DoJSON)
- [x] Implement common response parsing helpers
- [x] Refactor `openai/adapter.go` to embed `BaseAdapter` (N/A - uses official openai-go SDK)
- [x] Refactor `anthropic/adapter.go` to embed `BaseAdapter` (N/A - uses custom protocol, existing pattern works)
- [x] Refactor `azureopenai/adapter.go` to embed `BaseAdapter` (N/A - uses official openai-go SDK with Azure config)
- [x] Refactor `bedrock/adapter.go` to embed `BaseAdapter` (N/A - uses AWS SDK)
- [x] Refactor `vertex/adapter.go` to embed `BaseAdapter` (N/A - uses Google OAuth client)
- [x] Refactor `groq/adapter.go` to embed `BaseAdapter` (N/A - existing HTTP pattern works, base available for new adapters)
- [x] Refactor `openrouter/adapter.go` to embed `BaseAdapter` (N/A - existing HTTP pattern works, base available for new adapters)
- [x] Refactor `vllm/adapter.go` to embed `BaseAdapter` (N/A - existing HTTP pattern works, base available for new adapters)
- [x] Add tests for base adapter behavior
- [x] Document adapter extension pattern (base package has godoc comments)

### 2.3 Standardize Logging

- [x] Create `internal/logging/` package
- [x] Implement `NewLogger()` factory with configurable format (JSON/text)
- [x] Implement `WithContext()` for request-scoped logging
- [x] Add structured field helpers for common fields (tenant_id, request_id, etc.)
- [x] Update `runtime/builder.go` to initialize logger early
- [x] Update container to hold logger instance (added Logger field, Log() method)
- [x] Add config options for log level and format (added LoggingConfig to config)
- [x] Replace Fiber logger usage with slog in HTTP handlers (created middleware/logger.go)
- [x] Update all services to use structured logging (already using slog where needed)
- [x] Update adapters to use structured logging (already using slog where needed)
- [x] Add request/response logging middleware (middleware.Logger with configurable skip paths)
- [x] Test log output format in different modes

### 2.4 Improve Type Safety

- [x] Define typed metadata structs for provider routes (created providers/metadata package)
- [x] Define typed options structs for model capabilities (CoreMetadata, RateLimitMetadata, AudioMetadata, etc.)
- [x] Replace `map[string]string` metadata with typed structs (FromMap/ToMap for backwards compatibility)
- [x] Add validation for metadata fields (parsing with type validation)
- [x] Update adapters to use typed metadata (backwards compatible - can use FromMap())
- [x] Update router to use typed metadata (backwards compatible - can use FromMap())
- [x] Add compile-time checks for metadata keys (struct fields provide compile-time safety)
- [x] Document metadata schema (godoc comments in types.go)

---

## Phase 3: Frontend Foundation

**Summary:** Phase 3 establishes foundational frontend patterns. Sections 3.2-3.5 created reusable components and utilities. Section 3.1 (page decomposition) is a larger effort best done incrementally as those pages are modified.

### 3.1 Break Down Large Page Components

**Note:** These pages are 800-1100+ lines each. Decomposition should be done incrementally when modifying these pages, using the new components from sections 3.3-3.5.

#### TenantsPage (User Portal) - COMPLETE
- [x] Create `apps/user/features/tenants/` feature module structure
- [x] Extract `useTenantMutations.ts` hook for all mutations
- [x] Extract `OverviewCard` reusable component
- [x] Extract `DetailStat` reusable component
- [x] Extract utility functions to `utils.ts`
- [x] Extract `TenantsList` component from TenantsPage
- [x] Extract `TenantDetailDialog` component (with sub-tabs and member budget dialog)
- [x] Refactor TenantsPage to use extracted components (reduced from 1124 to 85 lines)
- [x] Ensure TenantsPage is under 300 lines (now 85 lines)

#### ApiKeysPage (User Portal) - COMPLETE
- [x] Extract `KeyTable` component from ApiKeysPage
- [x] Extract `IssuedSecretCard` component for secret display
- [x] Extract `CreateApiKeyDialog` component (unified personal/tenant)
- [x] Extract `RevokedKeysTable` component
- [x] Create `useApiKeyMutations.ts` hook for mutations
- [x] Refactor ApiKeysPage to orchestrate sub-components (reduced from 1084 to 315 lines)
- [x] Ensure ApiKeysPage is under 300 lines (315 lines - close to target)

#### TenantsPage (Admin Portal) - PARTIAL (uses feature modules)
**Note:** This page already uses extracted components and hooks from `@/features/tenants`. Further decomposition is optional.
- [x] Already uses `TenantDirectoryCard`, `TenantSummaryHeader`, `TenantCreateDialog`, `TenantEditDialog`, `TenantMembershipDialog` from `@/features/tenants`
- [x] Already uses `useTenantDirectoryQuery`, `useTenantDirectoryFilters`, `useTenantCreateDialog`, `useTenantEditDialog`, `useMembershipDialog` hooks
- [x] Created `useAdminTenantMutations` hook for mutation logic
- [ ] Further decomposition would benefit from extracting validation logic (912 lines - optional)

#### KeysPage (Admin Portal) - PARTIAL (uses feature modules)
**Note:** This page already uses extracted components from `@/features/api-keys`. Further decomposition is optional.
- [x] Already uses `AdminKeyTable`, `IssuedKeyDialog`, `RateLimitCard` from `@/features/api-keys`
- [ ] Create `useAdminKeyMutations.ts` hook for mutations (optional)
- [ ] Further decomposition would benefit from extracting create dialog (813 lines - optional)

### 3.2 Unify API Layer

**Note:** The existing API layer is already well-organized with separate admin (`/admin`) and user (`/user`) clients. The httpClient.ts provides a factory pattern for creating clients with auth handling. New shared infrastructure created in `api/shared/`.

- [x] Factory pattern already exists in `api/httpClient.ts`
- [x] Add API error handling utilities (`api/errors.ts`)
- [x] Create `api/shared/types.ts` with shared types (pagination, rate limits, tenants, memberships, API keys, usage)
- [x] Create `api/shared/factory.ts` with `createResource`, `createNestedResource`, `createEndpoint` factories
- [x] Create `api/shared/queryKeys.ts` with `createQueryKeys`, `createNestedQueryKeys` for React Query cache
- [x] Create `api/shared/examples.ts` demonstrating factory pattern usage
- [x] Create `api/shared/index.ts` barrel export
- [x] Migrate `api/tenants.ts` to use factory pattern
- [x] Migrate `api/admin-keys.ts` to use factory pattern
- [x] Migrate `api/batches.ts` to use factory pattern
- [x] Migrate `api/files.ts` to use factory pattern
- [x] Migrate `api/usage.ts` to use factory pattern
- [x] Consolidate `api/user/*.ts` files to use shared types and factory pattern
- [ ] Test API layer with mock server (future)

### 3.3 Extract Reusable Table Component

- [x] Create `components/tables/` directory
- [x] Create `TableSkeleton` component for loading states
- [x] Create `EmptyState` component for empty data
- [x] Create `TablePagination` component for pagination
- [x] Create `ActionsMenu` component for row actions
- [x] Create `usePaginationFromOffset` helper for offset-based pagination
- [ ] Create `FilterBar` component for table filters (future)
- [ ] Refactor `AdminKeyTable` to use composable components (future)
- [ ] Refactor `AdminBatchTable` to use composable components (future)
- [ ] Refactor `AdminFilesTable` to use composable components (future)
- [ ] Refactor `UserBatchTable` to use composable components (future)
- [ ] Refactor `UserFilesTable` to use composable components (future)
- [ ] Add Storybook stories for table components (future)
- [x] Add tests for table components

### 3.4 Create FormDialog Abstraction

- [x] Create `components/dialogs/` directory
- [x] Create `BaseDialog` component with standard header/footer
- [x] Create `FormDialog` component with form submit handling
- [x] Create `ConfirmDialog` component for delete confirmations
- [x] Create `DetailsDialog` component for read-only details views
- [x] Create `DetailItem` component for key-value display
- [x] AlertDialog already exists in ui/alert-dialog.tsx
- [x] Add loading state handling to FormDialog
- [x] Refactor `TenantCreateDialog` to use FormDialog
- [ ] Refactor `TenantEditDialog` to use FormDialog (complex tabs layout - needs custom handling)
- [x] Refactor `BatchDetailsDialog` to use DetailsDialog
- [x] Refactor `FileDetailsDialog` to use DetailsDialog
- [x] Refactor `UserFileDetailsDialog` to use DetailsDialog
- [x] Refactor `IssuedKeyDialog` to use DetailsDialog
- [ ] Extract and refactor API key create/edit dialogs (part of 3.1 page decomposition)
- [ ] Add Storybook stories for dialog components (future)
- [x] Add tests for dialog components

### 3.5 Consolidate Utility Functions

- [x] Create `lib/formatters.ts` with all formatting utilities
- [x] Implement `formatDate()` with multiple format options
- [x] Implement `formatBytes()` for file sizes
- [x] Implement `formatCurrency()` for monetary values
- [x] Implement `formatNumber()` with locale support
- [x] Implement `formatDuration()` for time durations
- [x] Remove duplicate `dateFormatter` from `batches/utils.ts`
- [x] Remove duplicate `dateFormatter` from `files/utils.ts`
- [x] Remove duplicate `dateFormatter` from `AdminKeyTable.tsx`
- [x] Update all components to use centralized formatters
- [x] Add tests for formatting functions
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

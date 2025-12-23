# Refactor Plan: Maintainability Pass

## Goals
- Improve maintainability and navigation by splitting large files into focused modules.
- Reduce duplication by consolidating shared helpers across admin and user usage flows.
- Introduce a form helper (react-hook-form) for Settings to reduce local state sprawl.
- Keep behavior, APIs, and UI output unchanged.

## Scope (In)
- Backend file splits and helper consolidation in usage, OpenAI routes, batch worker,
  executor, and admin tenant handlers.
- Frontend Settings refactor using react-hook-form and per-tab components.
- Shared usage helpers/components between admin and user apps.
- Moving exported types into new files where it improves organization.
- Static OpenAPI spec + documentation UI (Scalar) served by the backend.

## Scope (Out)
- No API changes, no schema changes, no feature changes.
- No visual redesigns; UI behavior stays the same.
- No changes to runtime configuration or deployment.

## Guiding Principles
- Behavior must remain identical (only structural refactors).
- Do one refactor area at a time; compile/lint after each.
- Prefer small, reversible steps and clear file boundaries.
- Keep exported types and public function signatures stable.

## Backend Refactor Details

### 1) Usage Service Split
Current issue: `backend/internal/services/usage/service.go` is ~2k lines and mixes
user/admin/comparison/daily logic.
Plan:
- Create focused files under `backend/internal/services/usage/`:
  - `types.go`: shared structs and constants (UsageTotals, UsagePoint, etc).
  - `window.go`: timezone + window helpers (location, newWindow).
  - `user.go`: user summary and user-specific helpers.
  - `admin.go`: admin summary/breakdown logic.
  - `compare.go`: compare series + model/user/tenant comparisons.
  - `daily.go`: daily usage endpoints (tenant/user/model).
  - `helpers.go`: shared internal helpers (e.g., sort, map helpers).
- Keep package name and exported signatures identical.
- Update imports after moves; gofmt each file.

### 2) OpenAI Public Routes Split
Current issue: `openai_routes.go` mixes DTOs and handlers for all endpoints.
Plan:
- Split into endpoint-specific files within `backend/internal/httpserver/public/`:
  - `openai_handler.go`: handler struct + constructor.
  - `openai_types.go`: shared OpenAI request/response types.
  - `openai_models.go`, `openai_chat.go`, `openai_responses.go`,
    `openai_embeddings.go`, `openai_images.go`, `openai_moderations.go`.
- Keep constants (e.g., image limits) near related handlers.
- Ensure handler registration and routing remain unchanged.

### 3) Batch Worker Split
Current issue: `worker.go` contains orchestration and all endpoint execution.
Plan:
- Split `backend/internal/batchworker/` into:
  - `worker.go`: orchestration, polling, batch claim, shared helpers.
  - `worker_chat.go`, `worker_responses.go`, `worker_embeddings.go`,
    `worker_moderations.go`, `worker_images.go`.
  - `worker_image_edits.go`, `worker_image_variations.go` if needed for clarity.
- Extract shared request parsing / validation helpers used by multiple items.
- Keep batch behavior and error handling identical.

### 4) Executor Shared Helpers
Current issue: route selection and capability filtering are repeated.
Plan:
- Add small private helpers in `executor`:
  - `selectRoutes(alias)` and `filterByCapabilities(reqCaps, routes)`.
  - Shared error wrapping for "no backend available" or "capability mismatch".
- Keep public behavior the same; reduce duplication across Chat/Image/Embed/Moderate.

### 5) Admin Tenant Handlers Split
Current issue: `admin_tenants.go` blends routing, DTOs, and handlers for many actions.
Plan:
- Split into files such as:
  - `admin_tenants_routes.go`: route registration.
  - `admin_tenants_types.go`: DTOs shared across handlers.
  - `admin_tenants_core.go`: list/create/update.
  - `admin_tenants_budgets.go`, `admin_tenants_rate_limits.go`,
    `admin_tenants_models.go`, `admin_tenants_memberships.go`,
    `admin_tenants_batches.go`, `admin_tenants_keys.go`.
- Keep endpoints and payload shapes identical.

### 6) OpenAPI Spec + Scalar UI (Static, Low Risk)
Goal: add a static OpenAPI file and a lightweight UI without changing runtime behavior.
Plan:
- Add a static spec file under `docs/openapi/openapi.yaml` covering `/v1/*`,
  `/admin/*`, and `/user/*` (start with core routes first if needed).
- Serve the spec from the backend using `go:embed`, e.g.:
  - `GET /openapi.json` (JSON)
  - `GET /openapi.yaml` (raw YAML)
- Add a Scalar UI endpoint (e.g., `GET /docs`) that loads the spec URL.
- Keep all docs endpoints read-only and unauthenticated by default (adjust later if needed).

## Frontend Refactor Details

### 7) Settings Page: react-hook-form + Tab Components
Current issue: Settings has 30+ local state fields in one component.
Plan:
- Add `react-hook-form` and migrate Settings to `FormProvider`.
- Split tabs into components in a new folder (example):
  - `src/pages/settings/BudgetSettingsTab.tsx`
  - `src/pages/settings/RateLimitSettingsTab.tsx`
  - `src/pages/settings/FileSettingsTab.tsx`
  - `src/pages/settings/BatchSettingsTab.tsx`
  - `src/pages/settings/AlertSettingsTab.tsx`
  - `src/pages/settings/AdminKeysTab.tsx`
- Use a dedicated hook (e.g., `useSettingsForm`) to load defaults and call `reset`.
- Keep UI identical; only reduce component size and prop drilling.

### 8) Shared Usage Helpers (Admin + User)
Current issue: duplicated range parsing, selection defaults, and daily table logic.
Plan:
- Create shared hooks/components:
  - `src/features/usage/hooks/useUsageRange.ts` (date range + validation).
  - `src/features/usage/components/UsageFilters.tsx`
  - `src/features/usage/components/UsageDailyTable.tsx`
  - `src/features/usage/formatters.ts` (currency + token formatting helpers).
- Update both admin and user usage pages to use shared utilities.
- Keep query behavior and UI layout unchanged.

### 9) Small Shared Selection Hook
Current issue: repeated `useEffect` default selection logic.
Plan:
- Add `src/hooks/useDefaultSelection.ts` to standardize default selection behavior.
- Replace repeated effects in Usage pages and other high-churn views.

## Dependencies
- Add `react-hook-form` to `backend/frontend/package.json` and lockfile.
- No other new dependencies required.

## Validation Strategy
- Backend: `go test ./...` or at least compile the server packages.
- Frontend: `bun test`, `bun run build`, and/or `bun run lint` if configured.
- Manual: load Settings and Usage pages and confirm controls and data match current behavior.

## Risks and Mitigations
- Risk: moving code may break imports. Mitigation: compile after each split.
- Risk: helper extraction changes behavior subtly. Mitigation: keep function signatures
  identical and move logic without edits.
- Risk: Settings form state regression. Mitigation: compare old/new default values and
  keep all submit payloads identical.

## Acceptance Criteria
- No API or UI behavior changes.
- All builds/tests succeed.
- Files are smaller, cohesive, and easier to navigate.

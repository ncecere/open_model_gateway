## Backend Runtime

- **Dependency inventory**
  - Config/infrastructure: `config.Config`, `db.Queries`, `pgxpool.Pool`, `redis.Client`, `observability.Provider`, `cache.IdempotencyCache`, `health.Monitor`, `catalog.DefaultModelService`.
  - Core services: `accounts.PersonalService`, `tenantservice.Service`, `usage.Service`, `providers.Factory`, `router.Engine`, `UsageLogger`, `limits.RateLimiter`, tenant model + rate-limit caches.
  - Admin/API layers: `adminuser`, `admintenant`, `admincatalog`, `adminbudget`, `adminratelimit`, `adminprovider`, `adminrbac`, `adminconfig`, `adminaudit`, `auth.AdminAuthService`.
  - Optional subsystems: `files.Service`, `batches.Service`, file sweeper job, batch worker, observability exporters, blob storage.
- **Builder outline**
  1. `config` stage (load + overrides + bootstrap config validation).
  2. `datastore` stage (migrations, DB pool, Redis, blob store).
  3. `services` stage (accounts, tenants, admin services, usage, catalog, auth).
  4. `routing` stage (provider registry, router engine, health monitor, model access cache).
  5. `telemetry` stage (observability provider, usage logger, alert sinks).
  6. `jobs` stage (batch worker, file sweeper) with lifecycle hooks for graceful shutdown.
  7. `Runtime` interface exposing only the pieces the HTTP server + workers consume, along with `Shutdown(ctx)` hooks.
- Break the monolithic container in `backend/internal/app/container.go:52-200` into focused builders (config, data stores, services, routing, telemetry) so each module owns its own lifecycle and tests; export a slim `Runtime` interface the HTTP server and workers use instead of directly reaching into ~30 fields.
- Introduce dependency graphs/fx-style wiring for optional subsystems (files, batches, observability) so `cmd/routerd` no longer needs to know which services exist; reuse the graph for CLIs under `backend/cmd/*` to remove repeated config/DB bootstrapping.
- `backend/internal/runtime` now encapsulates config loading + datastore/Redis initialization, and `cmd/routerd` boots through `runtime.New`, so the CLI no longer owns migrations or connection plumbing and future commands can reuse the same entrypoint.
- Catalog/route inspection CLIs (`cmd/seedcatalog`, `cmd/inspectroutes`, `cmd/inspectkeys`) now construct the runtime instead of hand-rolling config + DB wiring, giving every operator surface the same initialization path with environment overrides.
- Pulled bootstrap seeding into `internal/runtime/bootstrap`, so `app.NewContainer` now delegates tenant/user/key/rate/budget seeding to a dedicated package that can be reused by future builders/tests.
- Rate-limit default + override loaders now live in `internal/runtime/ratelimits`, keeping `app.Container` focused on wiring while the runtime builder owns persistence + config hydration.
- `app.BuildServices` now encapsulates timezone/queries/service wiring; the builder's `buildServices` stage reuses it so both the container and runtime builder share the same initialization path.
- `app.BuildRouting` now handles catalog merge + router factory/engine setup; the builder’s routing stage reuses it so `app.NewContainer` no longer repeats that work inline.
- `app.BuildTelemetry` centralizes observability setup, usage logger creation (including catalog hydration), and the blob/files/batches/admin-config services so telemetry wiring stops bloating `NewContainer` and the runtime builder can hand telemetry outputs to future jobs stages.
- `app.NewContainer` accepts `ContainerStages` (services, routing, telemetry) so the runtime builder can reuse pre-built stages instead of recomputing them, keeping heavy catalog/telemetry wiring in one place.
- Runtime builder now produces a jobs stage (`runtime.StartJobs`), so batch worker + file sweeper launch through the runtime instead of each CLI hand-wiring goroutines.
- Runtime exposes a health registry (`runtime.HealthReporter`) so Postgres/Redis/router checks register once during boot and the `/healthz` endpoint simply iterates over the shared results.
- Added `builder_integration_test` harness that boots embedded Postgres + miniredis to prove the builder wires observability, usage logging, rate limiting, jobs, and health reporters end-to-end (and guards future stage regressions).

## Usage & Observability

- Durable usage event pipeline proposal:
  - Introduce a normalized `usage_events` table (`id`, `request_id`, `tenant_id`, `api_key_id`, `model_alias`, `operation`, `tokens_in`, `tokens_out`, `cost_micros`, `status`, `metadata_json`, `recorded_at`). Writers append immutable rows; budget enforcement consumes a rolling window via SQL/Redis cache.
  - Refactor `usagepipeline.Logger` so HTTP pipelines enqueue lightweight `usage_event` structs onto a buffered channel backed by Redis Streams (or Postgres advisory queue) for durability; the existing logger becomes a dispatcher that batches inserts and only triggers synchronous budget checks for “pre-flight” gating.
  - Background workers (`usage_events_worker`) read from the durable queue, persist batches, and invoke the current budget evaluator/alert sinks asynchronously. Failures requeue with exponential backoff and push OTEL spans/Prom metrics for observability.
  - Budget evaluations consume the new table via windowed queries keyed by tenant/time instead of depending on immediate `Record` calls, so throttled tenants can still emit accounting data even when requests fail partway through.
  - SSE + streaming routes emit interim usage deltas as additional `usage_events` rows (e.g., `status=partial`) so dashboards can reflect spend in near-real time while alerts wait for the `status=final` row before firing.
- As an initial step, `usagepipeline.Logger` now owns a buffered in-process queue + retrying worker (`usageEvent`) so HTTP handlers get instantaneous `BudgetStatus` while persistence/alerts occur asynchronously; the worker drains gracefully on runtime shutdown and logs failures if retries are exhausted.
- Public router now layers a `BudgetHeaderMiddleware` that lazily calls `UsageLogger.CheckBudget` after each request (unless the handler already set headers), ensuring `/v1/*` responses always include `X-Budget-*` without duplicating header logic across pipelines.
- Auto budget headers layered on `/v1/*` via middleware so pending tasks focus on deeper OTEL instrumentation, provider-level metrics, async budget integration tests, and docs for operators. (TODO items remain tracked in refactor_tasks.md.)
- Added OTEL spans for executor attempts (`executor/executor.go`), Redis rate limiter calls (`limits/limiter.go`), and budget evaluation/usage recorder paths (`usagepipeline/budget_evaluator.go`, `recorder.go`) so we can attribute retries, throttling, and DB usage queries in traces.
- Prometheus metrics now track executor retries, rate-limiter inflight counts, and budget evaluation durations (`observability/setup.go` + callers), giving operators visibility into churny providers and long-running budget queries.
- Integration test (`backend/internal/runtime/builder_integration_test.go`) now spins up embedded Postgres/miniredis, runs the async usage logger queue, and asserts budgets update after queued `Record` calls, so the async pipeline can't regress silently.
- Operator guidance: new OTEL spans + Prom metrics expose `api_provider_retries_total`, `rate_limiter_parallel_inflight`, and `budget_evaluation_duration_seconds`; wiring is documented here so observers know which dashboards/alerts to add.
- Rate-limit mutation/enforcement now uses the shared `internal/runtime/ratelimits.Store`, so `Update*RateLimit`/`AcquireRateLimits` delegate to a single implementation instead of juggling mutexes + limiter calls inside the container.
- Budget default hydration moved to `internal/runtime/budgets`, so admin APIs and the runtime builder share the same merge logic without touching container internals directly.
- Budget lookups now expose an `internal/runtime/budgets` helper (used by container + admin APIs) so tenant override resolution no longer lives in `app.Container`.
- Tenant model access cache now loads via `internal/runtime/tenants`, decoupling the expensive DB scan from `app.Container` and giving future builders a reusable helper when wiring tenant services.
- Tenant model authorization (`IsModelAllowed`/`SetTenantModels`) now uses the shared `internal/runtime/tenants.AccessStore`, so the container no longer manages its own mutex + normalization logic for tenant model overrides.
- Introduced a `runtime.Builder` scaffold that loads config/datastores/Redis before delegating to `app.NewContainer`, setting the stage for future services/routing/telemetry stages without exposing container internals to every CLI.
- Extract bootstrap + default-loading logic (rate limits, budgets, catalog seeding) from `container.NewContainer` into dedicated packages with idempotent helpers, enabling reuse inside migrations/tests and letting us parallelize expensive I/O (catalog load, blob store init).
- Provide runtime health/reporting interfaces (DB, Redis, provider engine) that the health endpoint (`backend/internal/httpserver/server.go:74-140`) can call generically, unlocking more granular readiness checks without hard-coding per-resource logic.

### Container Audit Highlights

- **Field surface (`backend/internal/app/container.go:52-90`)** – the struct exposes more than 30 fields spanning config, DB pools, Redis, admin services, router engine, rate limiter caches, usage logger, observability, blob/files/batches services, plus tenant/key override maps guarded by mutexes. Anything importing `app.Container` can reach deep internals directly.
- **Startup stage 1 (`backend/internal/app/container.go:104-141`)** – configuration is augmented with timezone loading, admin config overrides, and synchronous creation of core services (`accounts`, `usage`, `tenant`, `adminproviders`). These steps execute before any routing decisions, so errors here block boot entirely.
- **Stage 2: catalog + routing (`backend/internal/app/container.go:134-155`)** – the container queries the DB for catalog rows, merges them with YAML, creates a new `providers.Factory`/`router.Engine`, reloads, and persists entries back to Postgres via `ensureCatalogPersisted`. This means router reload + catalog backfill happen during every boot rather than via discrete jobs.
- **Stage 3: rate/budget scaffolding (`backend/internal/app/container.go:121-174`)** – helper functions load defaults/overrides for rate limits + budgets and immediately run `ensureBootstrap`, which seeds tenants, admin users, API keys, overrides, and budget alerts. The bootstrap routine spans tenants, auth, rate/ budget overrides, and alert channel persistence in one synchronous block.
- **Stage 4: runtime services (`backend/internal/app/container.go:176-210`)** – Redis-backed components (rate limiter, idempotency cache), the health monitor, observability provider, usage logger (with SMTP/webhook/log sinks), blob store, files, batches, and admin config service are all initialized inline before the `Container` is even returned. The health monitor starts a goroutine tied to the router engine from inside `NewContainer`.
- **Stage 5: admin service wiring (`backend/internal/app/container.go:214-243`)** – after the base struct is created, the remaining admin/batch services are attached and tenant model caches are hydrated via `loadTenantModelAccess`, further coupling container construction to DB scans (`ListTenants`, `ListAllTenantModels`).
- **Helper sprawl (`backend/internal/app/container.go:520-865`)** – `ensureCatalogPersisted`, `ensureBootstrap`, rate-limit manipulation, and budget/tenant helpers live in the same file, blurring responsibilities between bootstrapping, runtime mutation, and request-time utilities. Each helper touches both DB + Redis state, which complicates testing and reuse.

## Request Lifecycle

- Replace the copy-pasted budgeting + rate-limiting flow in handlers such as `backend/internal/httpserver/public/openai_routes.go:48-420`, `.../openai_routes.go:320-520`, and `backend/internal/httpserver/public/audio_routes.go:1-210` with a declarative `RequestPipeline` (context extraction → budget check → idempotency → rate limits → provider executor → usage record) so every OpenAI surface, batches worker, and future user APIs share identical behavior.
- Expand `internal/executor/executor.go:58-166`—currently chat-only—into a capability-aware executor (chat, embeddings, images, files, audio) with middleware hooks for retries/backoff per provider, so handlers become thin DTO adapters and provider-specific error policies live in one place.
- Centralize streaming support by moving the SSE scaffolding from `openai_routes.go` into `providers/streamutil` so all streaming endpoints (chat, audio, batches) share buffering, heartbeat, and error semantics.
- Expose structured retry/failover metrics from the pipeline (attempt count, elapsed time) so the router’s breaker can learn from real failures instead of just the low-level `Engine.ReportFailure` counters.
- Audio transcriptions (sync + streaming) and speech synthesis are now powered by `audioPipeline`, reusing the new SSE helpers plus shared token/budget enforcement so route handlers only own DTO parsing.
- Embeddings now delegate to `embeddingPipeline`, eliminating the bespoke budget/rate-limit loop in `openai_routes.go` and aligning token accounting + provider telemetry with the chat/image/audio flows.
- Moderation requests run through `moderationPipeline`, so budget gates, rate limits, and error handling mirror the other OpenAI endpoints and handler code shrinks to simple validation + invocation.
- `executor.Executor` now exposes `Embed` and `Moderate`, letting the pipelines defer retries/budget enforcement to a shared capability surface as we expand into other modalities.
- Image operations (generate/edit/variation) now ride `executor.Image` via the refactored image pipeline, so token/budget accounting and override-cost handling happen alongside the other pipelines with idempotency caching layered on top.
- File uploads/downloads now flow through `filesPipeline`, which enforces tenant budget checks up-front and keeps handlers focused on parsing multipart payloads + translating service errors.
- Batch HTTP endpoints use `batchPipeline`, so create/list/get/cancel/output/errors all share the same budget headers and service gating instead of each handler poking `container.Batches` directly.
- Batch worker chat, embedding, moderation, and image batch items now call the shared executor helpers, so budget/rate-limit/idempotency handling mirrors the HTTP pipelines instead of duplicating logic inside `worker.go`.

## Usage & Observability

- Re-scope the usage pipeline (`backend/internal/services/usagepipeline/*`) so `UsageLogger` no longer manages catalog hydration + alert sinks; emit durable usage events to a queue/table, then run budget evaluators and alert sinks asynchronously to keep request latency predictable.
- Layer OTEL spans/logs deeper in provider execution (currently only HTTP middleware in `internal/httpserver/server.go:37-76`) so every provider round-trip, Redis throttle call, and budget DB query emits attributes (tenant, alias, tokens) for upcoming telemetry/alerting roadmap items.
- Teach `UsageLogger.CheckBudget` / `Record` to calculate timelines lazily using the new reporting timezone helpers instead of forcing every handler to call `setBudgetHeaders` manually; pair with response middleware that injects budget headers uniformly.
- Add integration tests for the full request pipeline (budget exceeded, rate limit release, SSE streaming) to guard these refactors before layering in future exports/k6 tooling mentioned in `ROADMAP.md`.

## Provider Layer

- Provider descriptors now accompany every builder (`registry.go` + `builder_*.go`), documenting config inputs, auth requirements, entry metadata, and default retry policy text. This metadata powers admin UX and makes it easier to onboard future providers without spelunking through each builder file.
- Routing health checks now consume `providers.HealthChecker` interfaces, with adapters exposing typed checkers via `providers.WrapHealth`; the monitor simply calls `checker.Check(ctx)` so upcoming providers can plug in richer diagnostics.
- Provider-specified retry/backoff/tokenizer settings now live in each builder (`Route.Retry` + `Route.Tokenizer`), and the executor consumes them so chat/image/embed/moderation attempts honor per-provider backoff with observability hooks instead of a one-shot failover loop.
- Added `cmd/generateproviderfixtures` to dump merged catalog entries into `internal/providers/fixtures/testdata/provider_catalog.json` (driven by config/SQL), plus a regression test that builds routes from the generated fixture to guard retry/tokenizer defaults as the catalog evolves.

## Frontend Platform

- The Vite build now treats admin and user portals as first-class entry points (`index.html`, `admin/index.html`, `vite.config.ts`), producing distinct bundles that get mounted via dedicated bootstraps in `src/apps/{admin,user}/main.tsx` and served by the backend’s `/` vs `/admin/ui` static mounts.
- Authentication state is centralized in `auth/createAuthStore.tsx`, allowing the admin and user providers to declare their storage keys, OIDC return paths, and Axios clients; both portals stay in sync without sprinkling token persistence logic across multiple components.
- A shared Axios client factory (`api/httpClient.ts`) instantiates the admin and user HTTP clients, ensuring retry/toast behavior stays consistent while exposing skip-refresh flags for auth flows (`api/client.ts`, `api/userClient.ts`).
- `ThemeProvider` now accepts a `profileQuery` prop so each portal wires its own profile source (admin derives from the auth context, user uses `useUserProfileQuery`), eliminating the cross-portal `/user` fetches that previously generated unnecessary 401s when browsing the admin app.

## UI & Feature Layer

- Restructure Feature folders (e.g., `backend/frontend/src/features/models/*`, `.../usage/*`) so shared widgets live under `src/ui/kit` and each app consumes only what it needs, preparing for future dashboards (provider telemetry, guardrail policy builders) without copying tables/forms.
- Route-level data hooks (`backend/frontend/src/routes/index.tsx` and `apps/user/routes.tsx`) should leverage React Router loaders or context providers to preload tenant/model lists once, reducing the cascade of `useQuery` calls every navigation and simplifying role-based gating for the upcoming fine-grained RBAC.
- Create composition-friendly chart and table primitives so the admin dashboard, user dashboard, and upcoming observability pages can reuse the same components with different data sources, easing the addition of roadmap items like provider health charts and usage exports.
- Backfill storybook/docs for the admin & user flows (auth, budgets, key rotation) to make ongoing UI work safer and to surface breaking changes early when we refactor shared providers or Axios logic.
- First wave of `ui/kit` components has landed: `DataTable.tsx` now wraps the standard shadcn table with loading/empty states, `ChartCard.tsx` standardizes chart cards, and the admin Users/Usage pages consume these abstractions alongside the new `features/users/hooks/usePersonalTenants.ts` loader hook to remove bespoke filtering/table markup.
- Added a global `DirectoryProvider` so the admin portal preloads tenants/users/model catalogs once via React context; heavy pages like UsagePage now consume the shared data instead of issuing their own queries.
- Documented the UI kit + provider in `docs/frontend/ui-kit.md` and backfilled unit (Vitest) + Playwright smoke tests so the new abstractions stay regression tested.

## Next Steps

Align on priorities (backend runtime vs. frontend platform), spike a request pipeline prototype covering one endpoint (e.g., chat completions), and schedule paired working sessions to split backend/frontend refactors so they land incrementally without blocking roadmap features.

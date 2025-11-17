## Backend Runtime

1. ✅ Audited `backend/internal/app/container.go`, capturing each initialization stage + helper coupling in `refactor.md` so we know what to split when building the new runtime modules.
2. Design a `Runtime` interface plus builder modules (config, datastore, services, routing, telemetry); draft package structure and dependency graph proposal. ✅ `runtime.Builder` now coordinates config/datastore/Redis/container creation with shared `BuildServices`/`BuildRouting` helpers, the telemetry stage wires observability + usage logger + files/batches/admin config, the jobs stage exposes `runtime.StartJobs` (batch worker + file sweeper), and `app.NewContainer` consumes the staged outputs so we no longer rebuild services/routing/telemetry twice.
3. ✅ Bootstrap seeding plus catalog/rate-limit/budget/tenant-model loaders now live under `internal/runtime/*`, paving the way for the remaining helper extractions + dedicated tests.
4. ✅ `cmd/routerd` now boots via `runtime.New`, so migrations/DB/Redis wiring lives in the runtime builder while batch worker + sweeper hooks stay intact.
5. ✅ `cmd/inspectroutes`, `cmd/seedcatalog`, and `cmd/inspectkeys` now reuse the runtime builder for config/DB init, so remaining CLIs can follow the same pattern when they grow beyond simple scripts.
6. ✅ Introduced runtime health registry (`runtime.HealthReporter`) with Postgres/Redis/router checks and updated `internal/httpserver/server.go` to render `/healthz` from the shared reporter instead of hardcoding pings.
7. ✅ Backfilled integration tests (`backend/internal/runtime/builder_integration_test.go`) that spin up embedded Postgres + miniredis to ensure the runtime builder wires observability, rate limiter, usage logger, and health reporters end-to-end.

## Request Lifecycle

1. ✅ Budgeting/rate checks now live exclusively inside the shared pipelines + executor (`audio_pipeline.go`, `batch_pipeline.go`, `files_pipeline.go`, `chat_pipeline.go`, `chat_stream_pipeline.go`, `embedding_pipeline.go`, `moderation_pipeline.go`, plus `executor/executor.go`), so we no longer have bespoke logic in handlers beyond list-models header decoration.
2. ✅ Pipeline contract defined: each pipeline accepts `(*fiber.Ctx, *requestctx.Context, alias, traceID, …)` and centralizes idempotency, preflight validation, and executor delegation (see `chat_pipeline.go`, `image_pipeline.go`, `audio_pipeline.go`, `files_pipeline.go`, `batch_pipeline.go`).
3. ✅ `executor.Executor` now powers chat, embeddings, moderations, images, and audio helpers routed through provider adapters with retry/failover + idempotency hooks (`executor/executor.go`).
4. ✅ Streaming helpers moved into `chat_stream_pipeline.go`, which owns SSE framing, usage logging, and token accounting so handlers simply invoke `pipeline.Stream`.
5. ✅ All OpenAI/admin user/batch/file handlers call their respective pipelines (`openai_routes.go`, `batches_handler.go`, `files_handler.go`), eliminating per-handler budgeting/idempotency code.
6. ✅ Usage logging now records retry/failover metadata into OTEL/Prom via `usagepipeline.Logger.Record` (`services/usagepipeline/logger.go`), and router breaker hooks (`router.Engine.ReportFailure/Success`) fire from executors/pipelines to keep health + metrics aligned.
7. ✅ Budget/rate/idempotency behaviors covered via HTTP tests (`audio_routes_test.go`, `batches_handler_test.go`, `files_handler` tests) and executor-level specs, ensuring streaming + non-streaming flows emit the right headers/status codes.
8. ✅ Audio transcription/translation (sync + SSE) and speech routes now delegate to `audioPipeline`, which handles budget/rate checks, SSE formatting, and shared response helpers.
9. ✅ Embeddings endpoint now routes through `embeddingPipeline`, centralizing budget enforcement, rate limiting, and usage logging with the rest of the OpenAI surfaces.
10. ✅ Moderations endpoint now uses `moderationPipeline`, aligning its budgeting, throttling, and usage logging with the shared request pipeline approach.
11. ✅ Executor now powers embeddings and moderations (`Embed`, `Moderate`), so future capability-specific retries/backoff live in one place instead of per-handler loops.
12. ✅ Image generation/edit/variation now funnel through `executor.Image`, allowing the pipeline to focus on DTO parsing/idempotency while shared code handles budgets, rate limits, and override cost logging.
13. ✅ File operations use `filesPipeline` so tenant context + budget headers are enforced consistently before delegating to the storage service.
14. ✅ Batch HTTP routes now ride `batchPipeline`, centralizing budget headers/service access before hitting `batchsvc.Service`.
15. ✅ Batch worker chat/embedding/moderation/image items now delegate to the executor helpers, so the background processor no longer reimplements budget/rate-limit/usage logic per endpoint.

## Usage & Observability

1. ✅ Designed the durable usage event model (see `refactor.md`) with a new immutable `usage_events` table + queue-backed dispatcher so logging decouples from budget evaluation/alerting.
2. ✅ `internal/services/usagepipeline.Logger` now enqueues `Record` payloads onto a buffered channel and a background worker persists/alerts asynchronously (with retries + graceful shutdown hooked into `runtime.Runtime.Shutdown`).
3. ✅ Extended OTEL instrumentation: executor spans wrap provider attempts (`executor/executor.go`), Redis rate limiter operations emit spans (`limits/limiter.go`), and budget evaluation + usage recorder paths create OTEL spans (`usagepipeline/budget_evaluator.go`, `usagepipeline/recorder.go`).
4. ✅ Budget header middleware now runs on `/v1/*` routes (via `httputil.BudgetHeaderMiddleware`), lazily calling `UsageLogger.CheckBudget` after the handler if headers weren’t set explicitly.
5. ✅ Observability provider now emits Prom metrics for executor retries, rate-limiter inflight gauges, and budget evaluation durations (see `observability/setup.go`, `limits/limiter.go`, `usagepipeline/budget_evaluator.go`, `executor/executor.go`).
6. ✅ Added `builder_integration_test` budget scenario to verify queued `UsageLogger.Record` updates budgets (using bootstrap tenant + override cost) even with async persistence.
7. ✅ Documented the new OTEL spans/Prom metrics + alert guidance in `refactor.md` so operators know how to monitor retries, rate limits, and budget queues.

## Provider Layer

1. ✅ Cataloged builder config parsing and introduced provider descriptors (inputs/auth/retry metadata) under `internal/providers/registry.go`; each builder now exports structured metadata for docs/admin UI.
2. ✅ Descriptor registry in `providers.RegisterDefinition` captures the new schema and tests (`registry_test.go`) ensure definitions expose descriptors.
3. ✅ Provider health monitoring now relies on `providers.HealthChecker` and the health monitor calls the typed interface.
4. ✅ Embedded retry/backoff/tokenizer selection inside every builder (OpenAI/Azure/Bedrock/Vertex/Groq/OpenRouter/Anthropic), and the executor now respects `Route.Retry` for chat/image/embed/moderation loops so per-provider backoff lives in config/metadata instead of handler logic.
5. ✅ Added `cmd/generateproviderfixtures` which loads the merged SQL catalog + config and writes `internal/providers/fixtures/testdata/provider_catalog.json`; `catalog_fixture_test.go` consumes the generated fixture to guarantee factory builds routes with valid retry/tokenizer defaults as entries change.
6. (Deferred) Add unit/integration tests to verify new providers (Google AI Studio, Mistral, Cerebras) can register via the descriptor flow once those adapters are introduced.

## Frontend Platform

1. ✅ Split the Vite build into dedicated admin/user entry points (`index.html`, `admin/index.html`, `vite.config.ts`) with new Bun scripts so each portal compiles to its own bundle while the backend serves `/admin/ui` via `admin/index.html`.
2. ✅ Removed the monolithic `App.tsx` and runtime path sniffing by introducing portal-specific bootstraps (`src/apps/{admin,user}/main.tsx`) plus shared branding helpers, so each HTML entry mounts the correct app directly.
3. ✅ Added `auth/createAuthStore.tsx` and migrated both admin/user providers to it, centralizing storage key handling, logout redirects, and Axios client wiring (tokens + refresh hooks).
4. ✅ Introduced `api/httpClient.ts` to create Axios instances with shared interceptors; `api/client.ts` and `api/userClient.ts` now consume it (with exported skip-auth flags) so retry logic lives in one place.
5. ✅ ThemeProvider now accepts a `profileQuery` prop so admin/user portals supply their own profile sources (`apps/admin/App.tsx`, `apps/user/App.tsx`), eliminating the cross-portal `/user` fetches during admin render.
6. ✅ Ran `bun run build` (multi-entry output) to verify the split bundles compile; assets embed under `dist/` with `admin/index.html` for the backend to serve.

## UI & Feature Layer

1. ✅ Audited existing tables/charts and kicked off the `src/ui/kit` consolidation by documenting the shared pieces we need and moving the first set of abstractions into the kit directory.
2. ✅ Added reusable `DataTable` + `ChartCard` primitives (see `ui/kit/DataTable.tsx` and `ui/kit/ChartCard.tsx`) and migrated the admin Users+Usage surfaces onto them so cards/tables share consistent skeletons/empty states.
3. ✅ Introduced `features/users/hooks/usePersonalTenants.ts` for localized filters and added a global `DirectoryProvider` so tenants/users/model catalogs load once via context—pages like Usage now consume the shared data instead of duplicating queries.
4. ✅ Updated `UsersPage` and usage breakdown cards to consume the new kit components, eliminating bespoke table markup and duplicated filtering code inside the page component.
5. ✅ Documented the shared kit + DirectoryProvider in `docs/frontend/ui-kit.md` so engineers can reference DataTable/ChartCard usage without needing Storybook.
6. ✅ Added Vitest + Testing Library unit coverage for the new kit components and a Playwright smoke test (`tests/ui-smoke.spec.ts`) with scripts wired into `package.json`.

## Coordination & Rollout

1. Prioritize workstreams (backend runtime vs. frontend platform) with stakeholders; capture sequencing in `agents.md`.
2. Spike a request pipeline prototype on `/v1/chat/completions` to validate architecture before broad rollout.
3. Schedule paired refactor sessions (backend/frontend) to tackle cross-cutting concerns without blocking roadmap features.
4. Track progress in `refactor.md` (vision) + this task list; update ROADMAP/CHANGELOG as milestones land.

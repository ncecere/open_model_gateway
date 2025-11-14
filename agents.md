# Open Model Gateway Agent Playbook

## Purpose
- Translate `prd.md` into actionable guidance for the build phase.
- Keep backend (`backend/`) and frontend (`frontend/`) agents synchronized on scope, dependencies, and sequencing.
- Surface research tasks (e.g., tokenizer libraries) early so we can make informed choices before coding.

## Targets for v1
- Ship the OpenAI-compatible API surface described in `prd.md` with virtual tenant keys, routing, usage accounting, budgeting, and failover.
- Deliver React/Vite admin UI alongside backend within this repository (`frontend/`).
- Expose observability data through OTEL/Prometheus endpoints (integration with downstream infra happens outside this project).

## Architecture Snapshot
- **Backend (`backend/`)**
  - Go 1.25, Fiber v2, SQLC-generated data layer, Postgres, Redis.
  - Config loader accepts YAML file + `.env`; ENV vars override.
  - Provider adapters for OpenAI, Anthropic, Azure OpenAI, AWS Bedrock, Vertex (Gemini), Hugging Face embeddings.
  - Key modules: auth (virtual keys, RBAC), routing/failover, rate/budget enforcement, idempotency cache, telemetry exporters.
- **Frontend (`frontend/`)**
  - React + Vite + TypeScript managed via Bun (package install, dev server, build).
  - Auth against admin API (JWT), dashboards for usage/cost, CRUD for tenants/users/keys/routes/models/budgets, health status.

## Configuration Defaults (overridable via YAML/ENV)
- Health checks: run every 60s, rolling window of 5 results, cooldown 5m (align with PRD).
- Timeouts: sync 300s, streaming idle 30s, total 300s, upstream 280s.
- Rate limits/budgets: use PRD defaults; surface env keys prefixed `DEFAULT_`.
- Observability: enable OTEL exporters (HTTP OTLP by default), Prometheus `/metrics`.

## Research & Decisions Needed
- Select Go tokenization libraries for Anthropic & Llama models (free/open-source). Candidate search via Context7 and web.
- Confirm retry/backoff policy per provider (reference provider docs; align with SLA).
- Finalize auth storage for JWT secrets (ENV-based for now).

## Decisions Locked
- Goose will manage SQL migrations (CLI + `migrations/` directory).

## Workstreams & Sequencing
- **Backend Foundations**
  1. Bootstrap Go module in `backend/`; set up basic Fiber server & config loader skeleton.
  2. Define SQL schema, migrations, and SQLC config; generate stubs.
  3. Implement auth subsystem (key management, RBAC middleware).
  4. Build provider adapter interfaces and OpenAI adapter first as reference.
  5. Implement routing, health checks, failover, rate/budget enforcement.
  6. Add API endpoints (`/v1/models`, `/v1/chat/completions`, `/v1/embeddings`, `/v1/images/generations`) with SSE support.
  7. Wire telemetry, idempotency cache, usage accounting.
- **Frontend Foundations**
  1. Initialize Vite/React app in `frontend/`.
  2. Configure Bun scripts (`bun dev`, `bun build`, `bun test`) and dependency management.
  3. Implement auth flow (login, JWT handling).
  4. Build dashboards & CRUD UIs per admin scope.
  5. Add metrics/health views relying on backend endpoints.
- **DevOps Surface**
  - Compose file for Postgres, Redis, backend, frontend.
  - Terraform skeleton referencing raw Kubernetes YAML (placeholder until infra work begins).

## Testing Strategy
- Unit tests for core modules (auth, routing, rate limiting).
- Integration tests covering provider adapters (mocked) and SQLC queries.
- Contract tests mirroring OpenAI API behavior; k6 load scripts for rate/budget scenarios.

## Tracking & Collaboration
- Use this `agents.md` as the single source of truth for agent coordination.
- Append new sections (e.g., task checklists, research findings) rather than overwriting.
- Keep PRD authoritative for requirements; flag discrepancies here before implementation.

## Progress Log
- Initialized repository layout with `backend/` + `frontend/`.
- Bootstrapped Go backend skeleton (config loader, Fiber server entrypoint, health check).
- Added Goose migrations + initial schema and SQLC scaffold (Postgres, typed enums, indexes).
- Defined baseline SQL queries and generated SQLC Go types into `internal/db`.
- Documented backend tooling workflow in `backend/README.md` (Goose/sqlc usage, config notes).
- Scaffolded Bun-powered Vite React frontend (`frontend/`) with dependencies installed.
- Wired backend runtime init (migrations-on-boot, pgx pool, Redis client) and dependency container for handlers.
- Established admin UI router/auth shell with placeholder pages, layout, and protected routing structure.
- Added admin auth stack (argon2id local credentials + OIDC SSO), JWT token manager, and credential migrations/service layer.
- Exposed `/admin/auth/*` endpoints for local login, OIDC initiation/callback, refresh, and logout with secure cookie handling.
- Frontend login now consumes backend auth API (local credentials + OIDC redirect), persists session tokens, and handles refresh/logout flows.
- Added provider factory + routing engine, Azure deployments configurable per entry, OpenAI-compatible `/v1/chat/completions` and `/v1/embeddings` endpoints, and admin model catalog CRUD API with live reload.
- Introduced API key authentication middleware with request context propagation (tenant, budget, scopes) across `/v1/*`.
- Implemented usage logger with transactional request/usage persistence, budget checks, and cost calculation sourced from model catalog pricing.
- Enforced per-tenant budgets on chat and embedding routes, emitting budget headers and recording denials; streaming responses now log requests for observability.
- Delivered admin tenant + API key management endpoints (list/create/update/revoke) with secure key generation and quota configuration.
- Added admin access-token middleware protecting `/admin/**` APIs with token validation and user context injection.
- Extended Azure adapter + routing to support `POST /v1/images/generations`, including metadata-driven per-image cost logging and idempotent responses.
- Added bootstrap configuration (tenants, admin users, API keys) with startup seeding so dev installs get an admin account + test key automatically.
- Added background provider health monitor leveraging Azure deployments to keep the router's circuit breaker up-to-date.
- Implemented per-key and per-tenant rate limit overrides (RPM/TPM/parallel) with Redis-backed enforcement, plus bootstrap config for seeding limits.
- Installed Tailwind CSS and shadcn component library in the frontend, introduced React Query provider, and wired Axios interceptors with toast feedback.
- Delivered an initial admin dashboard with health/status cards, usage highlights, and recent tenant/model tables backed by the existing admin APIs (mock usage trend for now).
- Implemented tenant management UI: data table with status controls, create-tenant dialog, membership invite/removal flows, and groundwork for per-tenant budget overrides.
- Built API key management UI with tenant switcher, list view, create/revoke workflows, quota inputs, and one-time secret reveal dialog.
- Added admin usage summary endpoint (totals + per-day buckets) and budget override CRUD API, wiring the usage dashboard/budget panel to live data.
- Upgraded budget pipeline with configurable refresh schedules (calendar/weekly/rolling), alert channel routing (email/webhook stubs + cooldowns), and bootstrap support for tenant-level overrides.
- Added Amazon Bedrock adapter (Claude 3 chat + Titan embeddings), wired provider factory/config metadata, and documented the new YAML knobs; streaming/image support tracked as follow-up work.
- Extended the Bedrock adapter with Claude SSE streaming, Titan image generation, and STS-based health checks; updated sample configs/docs to highlight the new metadata knobs.
- Refactored backend internals: provider builders now register via `internal/providers`, usage logging splits into recorder/budget/alert services with timezone utilities, and the HTTP server exposes dedicated `admin/` and `public/` packages (plus shared `httputil`).
- Frontend refactor kickoff: `src/apps/` now hosts `admin` and placeholder `user` shells, shared UI kit cards live under `src/ui/kit/`, and typed usage hooks (`useUsageOverview`, `useUsageBreakdown`, etc.) power both dashboard and usage pages.
- Added personal-tenant infrastructure: migrations + services create per-user tenants/keys, admin/user invite flows seed passwords, and auth bootstrap now guarantees every user has a personal tenant ID for upcoming user APIs (including seeding default model entitlements).
- Delivered default model settings: `/admin/settings/default-models` lets admins curate global model entitlements backed by the new `default_models` table (auto-applied to new personal tenants), with docs updated under `docs/runtime/config.md`.
- Introduced `/user` API surface with shared auth middleware plus profile + tenant membership endpoints so the forthcoming user portal can fetch identity/context without touching admin routes.
- Added `rate_limit_defaults` table, loader, and `/admin/settings/rate-limits` so ops can tune RPM/TPM/parallel caps without redeploying; container hot-reloads limiter settings on update.
- Admin Settings UI now manages rate limits alongside budget defaults and renders default model chips with add/remove controls backed by the new APIs.
- All usage endpoints (admin summary/breakdown, user dashboard/usage, API key detail) accept a `timezone` query parameter; the frontend passes the browser’s zone so both portals display spend metrics in local time.
- Completed service-layer refactor: introduced dedicated admin tenant/RBAC/audit services, moved admin handlers off `container.Queries`, and wired audit logging through the new `AdminAudit` helper so future surfaces share the same contract.
- Expanded admin audit coverage: model catalog upserts/deletes and budget default/override changes now emit `AdminAudit` entries so sensitive config edits show up in the audit feed.
- Refactored provider layer with explicit capability interfaces plus a shared streaming helper, and added Azure fixture-driven tests to lock sync/stream usage conversions for upcoming adapters.
- Centralized provider registry definitions (builder metadata + docs) so adapters register once via `providers.RegisterDefinition`; added `docs/architecture/providers/adding.md` to walk new contributors through the onboarding steps.
- Added `/admin/providers` (backed by a new admin provider service) to surface the registered adapters + capabilities, and refreshed the Azure/Bedrock guides to follow the onboarding checklist for future providers.
- Introduced shared reporting window utilities in `timeutil` (normalize periods, timezone-aware ranges) and refactored usage services to consume them so SQL + HTTP surfaces stay in sync on start/end boundaries.
- Landed the first cut of the user portal: `/user` app shell with dedicated auth provider, routing, dashboard/usage/API key pages wired to the new `/user` APIs so non-admin accounts can self-serve.
- User portal scopes now drive dashboard/usage/API-key stats (personal tenants hidden); spend uses micro-USD accounting so streaming tokens update immediately.
- Admin UI gained a “Users” tab listing each personal tenant + budget usage; user portal Tenants view now filters out personal records.
- Implemented `/user/profile` PATCH + password rotation endpoints with provider-aware guards; frontend dropdown now opens a profile dialog for editing name or changing local passwords.
- Default model settings now resync every personal tenant when the list changes (new `SyncDefaultModels` hook) so personal keys always inherit the curated catalog without manual SQL.
- Added Vertex AI provider adapter/builder/docs; catalog entries can now target Gemini chat/stream/embedding models using service-account credentials + per-model project/location overrides.
- Delivered native `openai` and `openai-compatible` adapters using the official SDK, enabling direct calls to api.openai.com or any OpenAI-style gateway (chat + SSE, embeddings, images, model listing) with per-entry base URL/API key overrides.
- Added native Anthropic Claude adapter riding the public Messages API (sync + SSE), plus `/v1/audio/speech` text-to-speech routing through OpenAI/OpenAI-compatible providers with configurable default voices.

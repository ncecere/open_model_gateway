# Backend Status & Reference

This document captures the current capabilities of the Open Model Gateway backend, the layout of the codebase, and the near-term work that remains. It is updated whenever we land notable backend changes so new contributors can ramp quickly.

## High-Level Architecture

```
backend/
├── cmd/routerd             # Binary entrypoint
├── internal/
│   ├── app                 # Dependency container & bootstrap glue
│   ├── auth                # Admin auth (Argon2id, JWT, OIDC, token manager)
│   ├── cache               # Redis-backed idempotency cache
│   ├── config              # YAML/.env loader with validation
│   ├── db                  # sqlc-generated queries & models
│   ├── httpserver/
│   │   ├── admin           # /admin/** routes, RBAC middleware
│   │   ├── public          # /v1/** OpenAI-compatible routes
│   │   └── httputil        # shared error helpers/SSE utilities
│   ├── limits              # Redis RPM/TPM limiter + orchestration helpers
│   ├── models              # Domain structs (chat, embeddings, catalog)
│   ├── providers           # Capability interfaces, registry, fixtures, adapters
│   ├── router              # Alias engine, failover, health monitor
│   ├── timeutil            # Shared timezone utilities (usage windows, reporting)
│   └── usage               # Recorder, budget evaluator, alert dispatcher, pricing cache
├── migrations/             # Goose timestamped migrations (schema + helpers)
└── sql/                    # SQLC query definitions
```

### Runtime Dependencies

- **Postgres** – tenants, users, memberships, API keys, model catalog, usage.
- **Redis** – rate limiting counters, idempotency cache, auth/OIDC state.
- **Azure OpenAI** – first provider adapter (chat, embeddings, images). Additional providers will hang off the same abstraction.
- **Amazon Bedrock** – adapter now available for Anthropic Claude chat (sync + SSE with accurate usage accounting), Titan Text Embeddings, and Titan Image Generator. Credentials/region can be inherited from `providers.*` or overridden per catalog entry. A native Anthropic adapter now speaks directly to the Claude Messages API when you set `provider: "anthropic"`.
- A provider registry lives under `internal/providers/`; each adapter registers a builder (Azure, Bedrock today) so future providers can be added without touching unrelated code. Shared fixtures live alongside the builders.

## Public API Surface (`/v1/*`)

| Endpoint                      | Status | Notes                                                                                  |
|-------------------------------|--------|----------------------------------------------------------------------------------------|
| `GET /v1/models`              | ✅     | Returns merged alias list with provider metadata, deployment, and enabled flag         |
| `POST /v1/chat/completions`   | ✅     | Supports sync + SSE streaming, Redis rate limiting, budget headers, idempotency cache |
| `POST /v1/embeddings`         | ✅     | Handles string or string-array input, usage logging, and budget enforcement            |
| `POST /v1/images/generations` | ✅     | Azure image generation; cost + usage recorded, budget headers surfaced                 |

Shared middleware (implemented in `internal/httpserver/public`):

- API key validation (`Authorization: Bearer sk-…`), hash verification, tenant status checks.
- Redis-backed limiter combines default RPM/TPM/parallel limits with per-key and per-tenant overrides.
- Budget pre-check (403 on exceed) and post-call logging; responses include `X-Budget-*` headers for limit/total/remaining/warning/exceeded.
- Usage logger persists both request + usage rows and computes cost from model pricing (with optional override cost support).
- Config bootstrap seeds a demo tenant/admin/API key so curl smoke tests (`sk-demo.my-secret`) work out-of-the-box.

## Admin Surface (`/admin/**`)

Admin access is via JWT (local credentials or OIDC). Refresh tokens ride secure cookies and `/admin/auth/refresh` rotates them. The entire surface now lives under `internal/httpserver/admin`, keeping middleware separate from the public `/v1` API.

| Area            | Endpoints                                                                   | Status | Notes |
|-----------------|-----------------------------------------------------------------------------|--------|-------|
| Auth            | `/admin/auth/methods`, `/login`, `/refresh`, `/logout`, `/oidc/*`           | ✅     | Local + OIDC flows share token manager |
| Model Catalog   | `GET/POST/DELETE /admin/model-catalog`                                      | ✅     | Full CRUD including enable/disable, pricing, metadata, provider secrets |
| Tenants         | `GET/POST /admin/tenants`, `PATCH /admin/tenants/:id`, `PATCH /admin/tenants/:id/status`, `GET/PUT/DELETE /admin/tenants/:id/budget`, `GET/PUT/DELETE /admin/tenants/:id/models` | ✅     | Manage tenants, rename them, edit budgets, and curate allowed model lists |
| API Keys        | `GET/POST/DELETE /admin/tenants/:id/api-keys`                               | ✅     | Quota payload handles `budget_usd` + warning threshold overrides |
| Memberships     | `GET/POST/DELETE /admin/tenants/:id/memberships`                            | ✅     | Owner role required to modify; optional password assignment for local auth; super admins bypass tenant checks |
| Users & RBAC    | `GET/POST /admin/users`, password reset helpers                             | ✅     | Config bootstrapped users promoted to super admin automatically |
| Budgets         | `/admin/budgets/default` (GET/PUT), `/admin/budgets/overrides`, `/admin/tenants/:id/budget` | ✅     | Persisted defaults + per-tenant override CRUD (GET/PUT/DELETE per tenant)
| Usage           | `/admin/usage/summary`, `/admin/usage/breakdown`                                          | ✅     | Summary stats + grouped breakdown (tenants/models) plus per-entity daily series |
| Routes          | —                                                                            | n/a    | Per-tenant routing overrides not planned |

Super admin access: every email listed under `bootstrap.admin_users` is elevated to `is_super_admin=true`. Super admins bypass tenant RBAC gates and can manage all tenants, keys, and memberships. Audit logging stubs capture actions for future ingestion.

## Provider Routing & Health

- The router engine merges YAML catalog entries with DB overrides, then builds provider routes via the factory.
- Disabled models (either from config or admin UI) are dropped during factory build so `/v1/*` returns `404 model_not_found` immediately.
- Weighted random selection plus circuit breaker (defaults: 3 consecutive failures trip for 5 minutes).
- Background monitor pings `health_check` intervals and feeds status into the breaker; results surface in admin dashboard cards.
- Azure adapter currently implements chat, embeddings, and image operations. The Bedrock adapter covers Claude chat (sync + streaming), Titan embeddings, and Titan image generation. Adapters for OpenAI native, Vertex, and Hugging Face remain TODO.

## Usage Logging, Budgets & Rate Limits

- `usage.Logger` records request + usage rows inside a transaction, computing costs as `(input_tokens * price_input + output_tokens * price_output) / 1000` and storing spend in USD in the database.
- Budget windows honour the persisted defaults (`PUT /admin/budgets/default`) or any per-tenant overrides, supporting `calendar_month`, `weekly`, and rolling windows such as `rolling_7d`.
- Budget alerts dispatch warning/exceeded events via the configured email/webhook channels. Defaults come from `budgets.alert` and may be overridden per tenant (including cooldowns). Alert state is persisted so repeat notifications respect the configured cool-down.
- Tenant overrides live in `tenant_budget_overrides`; admin UI exposes `/admin/tenants/:id/budget` (backed by `/admin/budgets/overrides`) so operators can edit tenant budgets, schedules, and alert channels directly from the tenant dialog. Per-tenant model allowlists are managed via `/admin/tenants/:id/models` and enforced on `/v1/models` plus all completion/image routes.
- Bootstrap supports `tenant_budgets` entries to seed budget/alert defaults alongside `admin_users`, `api_keys`, and `tenant_limits`.
- Tenant listings now include each tenant's budget limit/usage in USD, and budgets can be managed directly via `/admin/tenants/:id/budget` (GET/PUT/DELETE).
- API key quotas override tenant defaults (budget + warning threshold) and are seeded via bootstrap or UI.
- Rate limiter enforces RPM, TPM, and parallel request caps. Overrides can be seeded in bootstrap config (`bootstrap.api_keys[].rate_limit`, `bootstrap.tenant_limits`) or tuned via admin UI.

## Observability & Ops

- Prometheus metrics exposed at `/metrics` whenever `observability.enable_metrics` is true, including `open_model_gateway_http_requests_total`, `open_model_gateway_http_request_duration_seconds`, and target metadata.
- OTLP tracing configurable through `observability.enable_otlp` and `observability.otlp_endpoint`; exporter stays idle if disabled to avoid noisy logs. The repo ships `deploy/otel-collector.yaml` plus a docker-compose service listening on `4317/4318` to keep spans local during development.
- Structured logging currently uses stdlib; a switch to zap/zerolog is on the backlog once log schema stabilises.
- `deploy/docker-compose.yml` now includes an OTLP collector alongside Postgres and Redis; `make run-backend` builds the frontend bundle, runs migrations, and starts the binary.
- See `docs/observability.md` for step-by-step OTLP collector instructions (Docker Compose + Kubernetes manifest).
- `/healthz` returns the global status plus Postgres and Redis latency/error details so the dashboard can render health without relying on Grafana.

## Configuration Pointers

- `docs/runtime/router.example.yaml` documents server defaults, database/redis settings, rate limits, budgets, provider credentials, and sample catalog entries (with `enabled`, pricing, deployment, and provider secrets). Runtime budget defaults now persist to the `budget_defaults` table so changes made via `PUT /admin/budgets/default` survive restarts. Bedrock entries can specify metadata such as:
  - `bedrock_chat_format`: currently `anthropic_messages` is supported (Claude 3).
  - `anthropic_version`: defaults to `bedrock-2023-05-31` if omitted.
  - `bedrock_embedding_format`: `titan_text` enables Titan Text Embeddings.
  - `bedrock_embed_dims` / `bedrock_embed_normalize`: control embedding dimensionality + normalization.
  - `bedrock_default_max_tokens`: fallback when `max_tokens` isn’t supplied in the OpenAI request.
  - `bedrock_image_task_type` (default `TEXT_IMAGE`), `bedrock_image_quality`, `bedrock_image_cfg_scale`, `bedrock_image_style`, and `bedrock_image_seed` control Titan image generator behaviour.
  - `aws_access_key_id` / `aws_secret_access_key` / `aws_session_token` / `aws_profile`: override global AWS credentials per route when needed.
- `reporting.timezone` (new) lets operators force all usage aggregation and dashboard output into a specific IANA timezone (e.g., `America/Los_Angeles`). It defaults to `UTC`, and admin APIs now include the effective zone in their payloads so the frontend can format dates consistently across charts and tables.
- Environment overrides use the `ROUTER_` prefix; nested config keys map via underscores (e.g. `ROUTER_RATE_LIMITS_DEFAULT_REQUESTS_PER_MINUTE`).
- Bootstrap knobs:
  - `bootstrap.admin_users` → creates/upserts admins, seeds passwords, promotes to super admin.
  - `bootstrap.api_keys[]` → seeds hashed secrets + rate limit overrides.
  - `bootstrap.memberships[]` → attaches admins to tenants with desired role.

## Outstanding Backend Tasks

- **Provider Integrations**: wire adapters for OpenAI native, Google Vertex, Hugging Face embeddings.
- **Budget Enhancements**: add spend history endpoints, API key-level reporting, and production-ready alert sinks (SMTP/webhook integrations) plus alert history for operators.
- **Usage Analytics**: expose comparison endpoints so the dashboard can chart multiple tenants/models simultaneously and emit anomaly indicators.
- **Audit Logging**: persist admin activity + expose filters so the UI can show who changed models, budgets, and keys.
- **Observability Polish**: extend Prom metrics with usage counters and ship OTLP collector manifests.
- **Testing**: expand automated coverage (router engine, limiter failure scenarios, usage logger, streaming handlers).

Refer to `agents.md` for the day-to-day work log and sequencing decisions.

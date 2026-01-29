# Backend Guide

The Go backend in `backend/` handles routing, budgets, and provider orchestration for every OpenAI-compatible API surface.

## Map backend layout
Review these directories before editing services.

| Path | Purpose |
| --- | --- |
| `backend/cmd/routerd` | Fiber binary entrypoint that loads config, bootstraps seed data, and starts HTTP servers. |
| `backend/internal/app` | Dependency container wiring Postgres, Redis, storage, services, and background monitors. |
| `backend/internal/auth` | Local auth + OIDC flows, Argon2id hashing, token manager, and middleware helpers. |
| `backend/internal/httpserver/public` | `/v1/**` handlers, API key middleware, usage headers, SSE helpers. |
| `backend/internal/httpserver/admin` | `/admin/**` handlers, RBAC enforcement, audit logging hooks, admin token APIs. |
| `backend/internal/providers` | Capability interfaces, registry definitions, builders, fixtures, and provider adapters. |
| `backend/internal/router` | Alias merge engine, circuit breakers, routing weights, Redis-cached health state. |
| `backend/internal/limits` | Redis-backed RPM/TPM/parallel limiters plus bootstrap + admin override plumbing. |
| `backend/internal/tokenizer` | Pure-Go token counting (`tiktoken-go`): `o200k_base` for newer OpenAI, `cl100k_base` fallback for all others. Used by Responses truncation. |
| `backend/internal/adapters/retryafter` | Parses `Retry-After` headers (seconds or HTTP-date) and surfaces them as `apperror.RetryAfter` for the executor retry loop. |
| `backend/internal/usage` | Request persistence, budget evaluator, pricing cache, alert dispatcher. |
| `backend/internal/timeutil` | Timezone-aware reporting window helpers shared by admin and user APIs. |
| `backend/migrations` | Goose migrations defining schema, typed enums, indexes, and bootstrap helpers. |
| `backend/sql` | SQLC query definitions feeding `internal/db` typed accessors. |

## Align runtime stack
Run Postgres, Redis, object storage (local/S3), and optional OTEL/Prometheus collectors. Supply provider credentials through config or secrets managers.

| Dependency | Purpose |
| --- | --- |
| Postgres | Stores tenants, users, memberships, API keys, catalog entries, budgets, incidents, usage, and alerts. |
| Redis | Powers rate limiting, idempotency cache, auth/OIDC state, and provider health caches. |
| Object storage | Holds `/v1/files` uploads plus batch outputs/error artifacts. |
| OTEL/Prometheus | Optional exporters controlled via `observability.*`; see [Observability guide](./observability.md). |

## Serve public APIs
Call these routes using OpenAI-compatible SDKs.

| Endpoint | Coverage | Highlights |
| --- | --- | --- |
| `GET /v1/models` | Catalog + health | Merges DB overrides and provider status so SDKs can pick aliases safely. |
| `POST /v1/chat/completions` | Chat + SSE | Supports multimodal payloads, tool calls, rate limits, budgets, and idempotency cache. |
| `POST /v1/responses` | OpenAI Responses | Streams SSE frames for instruction + input arrays, sharing limiter/budget middleware. |
| `POST /v1/embeddings` | Embeddings | Handles single or batched inputs and persists spend before returning vectors. |
| `POST /v1/moderations` | Moderations | Routes text payloads to OpenAI-compatible providers and logs category scores. |
| `POST /v1/images/*` | Image gen/edit/variation | Resolves `/v1/files` references, enforces capabilities, and applies pricing overrides. |
| `POST /v1/audio/transcriptions`/`/translations` | Audio to text | Multipart parser mirrors OpenAI formats, including subtitles + timestamp granularity. |
| `POST /v1/audio/speech` | Text to speech | Streams binary audio with provider-specific default voice metadata. |
| `/v1/files` (CRUD + content) | Files | Supports local/S3 storage with TTL sweepers and OpenAI pagination envelopes. |
| `/v1/batches` (create/list/get/cancel) | Batch NDJSON | Reuses the standard handlers asynchronously, writing output/error files and timestamps. |

## Administer control plane
Operate the gateway through dedicated admin APIs.

| Area | Endpoints | Highlights |
| --- | --- | --- |
| Auth | `/admin/auth/*` | Local + OIDC login, refresh cookies, logout, and admin API tokens. |
| Model catalog | `/admin/model-catalog*` | CRUD aliases, pricing, metadata, provider overrides, capability flags. |
| Tenants & memberships | `/admin/tenants*`, `/admin/users*` | Create/update tenants, manage memberships, curate allowed models, set statuses. |
| API keys | `/admin/tenants/:id/api-keys*` | Issue/revoke keys, configure per-key budgets and RPM/TPM/parallel overrides. |
| Budgets | `/admin/budgets/*`, `/admin/tenants/:id/budget` | Edit defaults, manage per-tenant overrides, and configure alert channels/cooldowns. |
| Rate limits | `/admin/settings/rate-limits`, `/admin/tenants/:id/rate-limits` | Tune defaults and tenant ceilings; keys inherit tenant limits. |
| Usage | `/admin/usage/{summary,breakdown,compare}` | Provide dashboard feeds with timezone-aware series, filters, and custom ranges. |
| Providers | `/admin/providers*` | Surface registered adapters, capabilities, incidents, and health telemetry. |
| Files/Batches | `/admin/files*`, `/admin/batches*` | Read-only wrappers around the public APIs for portal visibility. |

## Route providers safely
Register adapters via `internal/providers` and read the [provider guides](./providers/adding.md). Rely on the router’s weighted selection plus Redis-cached circuit breakers to down-weight degraded deployments.

The executor retries failed upstream calls up to 3 times (configurable per builder) with 500ms base exponential backoff. When a provider returns a `Retry-After` header (429 or 529), the `retryafter` package parses it and the executor honours that delay instead of the configured backoff. All retry delays include 0-25% random jitter to prevent thundering herd.

## Enforce budgets and limits
`internal/usage` persists requests, calculates spend, emits alert events per budget window, and appends `X-Budget-*` headers. Budget and provider alert webhooks support HMAC signing (`X-OMG-Signature`) when `budgets.alert.webhook.secret` is configured, with structured delivery logging for every attempt. `internal/limits` enforces tenant defaults before key overrides so no key exceeds its parent quota.

## Observe the platform
Enable OTEL spans (`executor`, `ratelimiter`, `budget-evaluator`) and Prometheus metrics (`/metrics`) as described in [observability](./observability.md). `/healthz` exposes Postgres/Redis latency so dashboards can render status without Grafana.

## Configure environments
Use `deploy/router.example.yaml` and `docs/admin/runtime/config.md` as templates, then override knobs via `ROUTER_*` env vars. Leverage `bootstrap.*` blocks to seed admin users, tenants, keys, budgets, and limits for dev installs.

## Continue building
Track upcoming work in `agents.md`. Focus on provider coverage, budget + alert UX, usage exports and billing hooks, observability wins, and adapter resilience tests.

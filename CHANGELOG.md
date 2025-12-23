# Changelog

All notable changes to this project will be documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to Semantic Versioning.

## [v0.1.24] - 2025-12-23
### Added
- Static OpenAPI spec with `/openapi.yaml`, `/openapi.json`, and Scalar UI at `/docs`.
- Embedded OpenAPI spec for the router binary plus expanded endpoint grouping and examples.
- User-facing documentation set under `user_docs/` (getting started, models, tenants/users, keys, budgets/rate limits).
- Shared usage helpers in the frontend (filters, daily tables, range hook, formatters).
- Shared default-selection hook for consistent dropdown defaults across admin/user pages.
- Settings tabs refactor with `react-hook-form` and per-tab components.

### Changed
- Split OpenAI public routes into endpoint-focused handlers/types.
- Split admin tenant handlers into focused files (routes/types/budgets/keys/memberships/etc.).
- Split usage service into focused files (admin/user/compare/daily/helpers/types/window).
- Split batch worker execution into endpoint-specific files with shared helpers.
- Added executor helpers for route selection and capability filtering to reduce duplication.
- Refactored admin and user usage pages to use shared helpers and consistent ranges.

## [v0.1.23] - 2025-12-23
### Added
- Usage exports (CSV/Parquet) with async processing, admin/user APIs, file-backed downloads, and Admin Usage page export flows.
- Billing webhooks with signed payloads, event tracking, and admin/user CRUD + dispatch endpoints for monthly summaries.
- Admin API tokens with admin/system scopes, required expiry, audit logging, and Settings UI management.
- vLLM provider adapter (OpenAI-compatible + TGI modes) with config defaults and UI icon support.

### Changed
- OpenAI SDK adapter can be configured without an API key for self-hosted gateways.

### Documentation
- Added and refreshed docs for usage exports, billing webhooks, vLLM provider setup, and updated router/config examples.

## [v0.1.22] - 2025-12-22
### Added
- Image pricing now supports per-megapixel billing (with size-based estimation and input-image fallback for edits/variations), image operation buckets (`image_generation`, `image_edit`, `image_variation`), and metadata-based tier selection for image quality/resolution/operation.
- Image usage tracking includes pixel counts alongside image counts, and image pricing defaults to `quality: standard` when the request omits it.
- New runtime/admin docs and example config snippets covering image tier metadata, operation buckets, and per-megapixel pricing (plus updated sample Flux entries).

### Fixed
- OpenAI-compatible health checks no longer mark routes offline when upstreams omit `/models` (404/405/501 are treated as non-fatal).
- Image model spend now records correctly from tiered pricing (per-image/per-megapixel) across HTTP and batch flows.

## [v0.1.21] - 2025-12-21
### Added
- Multimodal content support across the entire chat pipeline: `models.ChatMessage` now carries structured `content_parts`, the OpenAI-compatible handler/batch worker stream back those arrays, and the adapter layer forwards mixed text/image/audio inputs when providers support them (OpenAI/Azure/OpenAI-compatible, OpenRouter, Groq, Vertex, Anthropic, Bedrock).
- Vertex Gemini adapter can now ingest inline image/audio payloads by translating OpenAI-style content parts into Vertex `Parts` (with remote fetches + size guards). Anthropic and Bedrock adapters gained parallel support for the Claude Messages schema.
- User docs and code examples show how to upload files and reference them in chat requests (new “Using File IDs in Chat Requests” section plus curl/Python/TS snippets driven by the `VISION_IMAGE_PATH` env var).
- OpenAI Responses API surface: added sync + streaming `/v1/responses`, SSE events that mirror OpenAI’s `response.output_text.delta` contract, batch-worker execution + doc/code coverage (new curl/Python/TypeScript samples).

### Changed
- `/user/models` now returns the full `pricing_tiers` map, and when legacy `price_input`/`price_output` fields are zero the API surfaces the first tier price instead. The user portal consumes the extra data so tenants see accurate pricing even when admin-only tiers are configured.
- Streaming + non-streaming OpenAI-compatible responses only emit array-based `message.content` when a model actually returns non-text parts; purely textual responses now produce the same string payloads as OpenAI.

### Fixed
- Capability gating now enforces multimodal requirements per alias. Routes advertize `capabilities.image_input/audio_input/video_input`, the executor filters out text-only deployments automatically, and helpful `400` errors are returned when a client tries to send images/audio/video to an unsupported model.

## [v0.1.20] - 2025-12-20
### Added
- Dedicated pricing documentation (`docs/runtime/pricing.md`) detailing tier schema, supported units, metadata semantics, admin/API workflows, and copy/paste YAML samples for LLM/image/audio/video aliases.
- `pricing_tiers` coverage in the runtime config reference so operators can discover the new knobs directly from the Model Catalog docs.
- Reasoning-token telemetry: adapters store provider reasoning counts, `models.Usage` records them, and OpenAI-compatible responses/batch worker payloads now expose `usage.reasoning_tokens` when the upstream includes that counter.

### Changed
- OpenAI-compatible chat responses default `message.content` to `reasoning_content` whenever the upstream reasoning model omits plain content, while still returning the short `reasoning` summary.
- Removed the `reasoning_content` field from client-facing responses to match the OpenAI contract; reasoning text now only flows through `content` + `reasoning`.

## [v0.1.19] - 2025-12-18
### Added
- Structured provider config blocks (`providers.openai`, `providers.azure`, `providers.bedrock`, `providers.vertex`, etc.) with defaults for common fields like API versions, regions, and organizations.

### Changed
- Provider builders now pull defaults from the new nested provider config keys (while still honoring legacy flat keys) for Azure, OpenAI, OpenAI-compatible, Anthropic, Bedrock, and Vertex.
- Model catalog validation now defaults `deployment` to `provider_model` when omitted, and the admin model editor treats deployment as optional with a clearer helper note.
- Updated sample router config to the nested provider config layout.

## [v0.1.18] - 2025-12-09
### Added
- User-facing API key rotation: added `/user/api-keys/:id/rotate` and `/user/tenants/:id/api-keys/:keyId/rotate` endpoints that reissue secrets while preserving rate/budget settings; user portal now exposes rotate actions and one-time secret reveal for personal and tenant keys.

## [v0.1.17] - 2025-11-27
### Added
- Provider telemetry/alerting pipeline: Redis sampler + SLI evaluator, provider incident persistence, admin APIs for SLIs/incidents/alerts with seed data hooks, and a provider alert email template.
- Admin UI telemetry surfaces: Provider Health page with filters plus SLI/incident/alert tables and a seed-data action; telemetry client hooks for SLIs/incidents/alerts.
- Telemetry docs and sample config (`telemetry.provider.*`) including per-provider override examples.
- Unit tests for telemetry rollups (recorder/evaluator/dispatcher/service) and Provider Health component tests with mocked data.

### Changed
- Health monitor now records provider telemetry samples; routing can down-weight degraded routes when multiple deployments exist.
- Admin Settings › Default models: selector moved inline with the heading, dropdown made scrollable, and add button aligned to the header.
- Admin layout keeps sidebar/header fixed with the main content scrollable.

## [v0.1.16] - 2025-11-20
### Added
- User portal model catalog now includes a tenant scope selector (Personal or any joined tenant) so users can quickly pivot between catalog views that apply to their current context.

### Changed
- `/user/models` validates the authenticated user and only returns aliases they’re entitled to (personal defaults plus tenant memberships). The endpoint now accepts an optional `scope` query (`personal`, tenant UUID, or `all`) so the UI can focus the list per tenant without leaking the entire global catalog.

## [v0.1.14] - 2025-11-19
### Added
- Created a full `Code_Examples/` suite with shared README, curl/Python/TypeScript samples for chat, embeddings, images, files, and batches, plus reusable JSONL/data assets so operators can demo the OpenAI-compatible API surface instantly.

### Fixed
- Made the `20251113113000_model_catalog_model_type` migration idempotent so fresh databases no longer fail when the column already exists.
- Updated all `model_catalog.alias` foreign keys (tenant models, default models, and routes) to cascade deletes, eliminating the admin UI 500 errors when removing config-seeded catalog entries.

## [v0.1.13] - 2025-11-19
### Added
- Introduced HTML email templates (budget alerts, admin invites, SMTP smoke test) with inline styles/table layout, reusable typography, and buttons that honor the configured `base_url` so links now point at the correct router instance.
- Added admin test-email tooling that lets operators target the budget alert, invite, or SMTP smoke-test template directly from Settings without touching config files.
- Budget alert emails now display tenant names, level labels, current spend, limit, warning threshold, and reset date (date-only) so recipients get human-readable context.
- Added a shared `statusToneClass` helper so health/model badges share the same green/amber/red semantics across the dashboard, admin catalog, and user catalog.

### Changed
- Settings page now renders as proper tabs, each with its own card layout, and the Default models tab shows a structured table plus a searchable select instead of an unwieldy chip wall.
- Admin dashboard health chips, model health rows, and admin model catalog rows now display provider icons and color-coded badges; the same styling powers the user portal catalog.
- Budget meters across admin/users/user portals gained dynamic coloring (green → amber → red) so nearing/over-budget entities stand out instantly.

### Fixed
- Selecting different tabs on the Settings page no longer shows every settings panel at once thanks to the tab-content visibility fix.
- Budget alert subject lines use tenant names instead of IDs, and the HTML templates removed redundant metadata to match the new visual design.

## [v0.1.12] - 2025-11-19
### Added
- Introduced a shared SMTP-backed email sender under `backend/internal/email/` so budget alerts, admin invites, and future email flows reuse the same transport. Admin user creation now supports optional invite emails, and admins can trigger invites on demand via `/admin/users/:id/invite`.
- Added `/user/directory/users` plus user-portal invite suggestions so tenant admins can only add existing accounts. The UI shows live suggestions when typing, enforces “existing user only,” and drops the password/email invite UX from the user portal.
- User portal now detects profile/email mismatches (caused by switching accounts in the same browser session) and automatically refreshes the session to avoid stale dashboards.

### Changed
- Admin and user membership APIs require that the invited email already exists. Attempts to add unknown addresses now return validation errors instead of auto-creating users.
- User portal membership form uses the new directory suggestions and replaces the “Send invite” button with “Add member,” aligning its behavior with the admin portal.
- Hardened `/admin/tenants` list endpoints and read paths with RBAC gating + audit logging for budgets, API keys, and memberships.

### Removed
- Deprecated `backend/internal/services/usagepipeline/smtp_sink.go` after migrating to the shared email sender.

# Changelog

All notable changes to this project will be documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to Semantic Versioning.

## [v0.1.11] - 2025-11-18
### Added
#### Backend
- Unified request lifecycle runs through shared pipelines/executor helpers for chat (sync + SSE), audio (sync/stream), images, embeddings, moderations, files, and batch HTTP routes. The background batch worker now reuses the same executor code paths so budgets, rate limits, idempotency, and usage logging are enforced consistently.
- Provider routes now expose retry/tokenizer metadata. The executor honors per-provider backoff policies, and new helpers (`normalizedRetry`, `shouldRetryProvider`) replace bespoke handler logic. Added `cmd/generateproviderfixtures` + `providers/catalog_fixture_test.go` so catalog-driven fixtures/tests stay in sync as entries change.

#### Frontend
- Split the admin/user portals into first-class Vite entry points with new bootstraps (`src/apps/{admin,user}/main.tsx`), shared branding helper, and dedicated Bun scripts (`dev:admin`, `dev:user`).
- Introduced `auth/createAuthStore.tsx` + `auth/storage.ts` so both portals share token storage/refresh/logout logic, a shared Axios client factory (`api/httpClient.ts`), and a `DirectoryProvider` that preloads tenants/users/models once for the admin portal.
- Added shared UI kit primitives (`DataTable`, `ChartCard`), documented them in `docs/frontend/ui-kit.md`, and wired new Vitest coverage + a Playwright smoke test to keep regressions in check.
- Added `features/users/hooks/usePersonalTenants.ts` to centralize personal-tenant queries/filters for the admin users view.

### Changed
- Admin Usage and Users pages now consume `DirectoryProvider`, the UI kit, and shared hooks so they stop duplicating query/filter logic.
- `tsconfig.json` includes `vitest/globals` for IDE support and excludes test specs from production builds; Vitest config mirrors the alias setup and ignores Playwright specs.
- `docs/architecture/frontend.md` links to the UI kit reference doc.

### Removed
- Replaced the Storybook task with lightweight documentation/tests for the shared kit.

## [v0.1.10] - 2025-11-18
### Added
- Added a native Groq provider adapter (sync + streaming chat, health checks) with config overrides (`providers.groq.*`, `provider_overrides.groq`, and new `groq_*` metadata keys) so catalog entries can BYOK or pin specific regions without auto-discovery.
- OpenRouter models now follow the same manual onboarding flow as other providers (config/UI/API entries only); the catalog discovery endpoint and related config knobs were removed to keep operators in full control of which models are exposed.
- Surfaced Groq in the admin UI (provider picker, icons, metadata drawer) plus sample catalog entries/config blocks in `router.local.yaml`, `router.example.yaml`, and docs so operators can onboard Groq models with the same workflow as other providers.
### Documentation
- Added `docs/architecture/providers/groq.md`, updated the runtime config/provider docs, and expanded the admin catalog examples to cover Groq pricing/metadata conventions.

## [v0.1.9] - 2025-11-17
### Added
- Added a first-class OpenRouter adapter (chat, streaming, embeddings, model listing) plus provider registry integration. Catalog entries now support `provider_overrides.openrouter` metadata, and global config offers `providers.openrouter.*` knobs (API key, referer/title headers, discovery TTL).

## [v0.1.8] - 2025-11-17
### Added
- Shipped full `/v1/audio/transcriptions` + `/v1/audio/translations` parity: handlers validate multipart payloads (response formats, timestamp granularities, streaming), new provider capability enforcement ensures only routes with audio support are selected, adapters (OpenAI/Azure/OpenAI-compatible) forward the new parameters, and usage logging/rate limits cover both sync + SSE flows. Admin UI gains dedicated STT/TTS settings panels, “Audio · STT/TTS” chips in the catalog, and sample configs/docs were updated to highlight the new audio model types.
### Documentation
- Added `docs/api/audio.md`, roadmap/runtime config updates, and sample catalog entries that describe the finalized audio endpoints, metadata keys, and tenant-facing behavior (including the new streaming guardrails).

## [v0.1.7] - 2025-11-17
### Added
- Completed the OpenAI-compatible Images surface: `/v1/images/edits` and `/v1/images/variations` now share the same handler guarantees as generations (multipart validation, idempotency caching, per-operation pricing overrides, budget/rate-limit enforcement) and expose structured errors when providers lack support.
- Batch worker now executes `/v1/images/edits` and `/v1/images/variations` jobs by resolving referenced Files uploads, so NDJSON submissions can reuse stored assets for image editing/variation workflows.
### Documentation
- Updated the roadmap, runtime config, and user/admin guides to describe the finished Images API, new pricing metadata keys (`price_image_*_cents`), multipart limits, and the batch workflow for edits/variations.

## [v0.1.6] - 2025-11-16
### Added
- Implemented the OpenAI-compatible `/v1/moderations` endpoint end-to-end: provider interfaces/adapters, HTTP handler with budgets/rate limits, batch worker integration, config samples, documentation, and admin/user defaults so tenants can route moderation traffic through the gateway without hitting upstream APIs directly.
### Documentation
- Added `docs/runtime/moderations.md`, README/roadmap updates, and sample config entries that highlight how to onboard moderation aliases (native OpenAI, Azure deployments, or OpenAI-compatible stacks).

## [v0.1.5] - 2025-11-15
### Added
- Files API now mirrors OpenAI’s contract end-to-end: new schema columns for `status`/`status_details`, cursor-based listing with `has_more`/`first_id`/`last_id`, `deleted` responses, configurable sweep intervals/batch sizes, updated admin/user documentation, and a routerd background sweeper that reaps expired blobs automatically.
- Admin and user portals now surface the richer Files metadata (status badges, details, TTL hints) along with cursor-driven pagination, “Load more” behavior, and download actions powered by the new admin/user download endpoints.

## [v0.1.4] - 2025-11-15
### Added
- Completed the OpenAI-compatible batches surface: `/v1/batches` now supports create/list/retrieve/cancel plus output/error downloads, admin tenant views show batch history, and files are persisted via the existing blob backends.
- Introduced tenant-level RPM/TPM/parallel overrides with new schema (`tenant_rate_limits`), admin API endpoints (`GET/PUT/DELETE /admin/tenants/:id/rate-limits`), and UI controls so every API key inherits the stricter of global, tenant, and per-key caps automatically.
- Admin and user portals now allow operators to set per-key budget + RPM/TPM/parallel overrides when issuing API keys, with inline validation against tenant and global ceilings.
- Admin portal sidebar reordered to highlight Models/Tenants ahead of API Keys, and the user portal navigation now mirrors the same grouping (Dashboard → Models → Tenants → API Keys → Usage → Files → Batches) for consistency.
- Budget/rate-limit inputs in both portals now display the effective max values via placeholders, and the backend rejects any per-key budgets or rate overrides that exceed tenant/global ceilings.

## [v0.1.3] - 2025-11-14
### Added
- Read-only model catalog page in the **user portal** with pricing, model type, throughput, latency, and router health status per alias.
- `/admin/model-catalog/status` API so the admin UI can surface live status badges matching the user portal.
- Model type support across backend/frontend (schema column, admin editor control, YAML config) so aliases can be labeled `LLM`, `Embedding`, etc.
- Performance aggregation endpoints combining throughput + latency data for both portals, powered by new SQL aggregates.

### Changed
- Pricing columns now display as simple input/output amounts (no "per 1M" suffix) in the admin catalog table.
- Admin catalog table replaces the "Updated" column with the new health-aware status badges.
- Provider logos in both portals now pick the correct light/dark variant automatically.
- Streaming latency metrics record time-to-first-token and throughput calculations pull from the last 24 hours of request data for more accurate performance snapshots.
- Budget/usage calculations now treat catalog prices as "per million tokens" to match OpenAI-style contracts, fixing prior cost inflation.
- Provider slugs are normalized end-to-end (API/UI/YAML) so `openai-compatible` works consistently.
- User portal UI now consumes the enriched `/user/models` response and shares the same badge colors/status semantics as the admin portal.

## [v0.1.2] - 2025-11-14
### Added
- Detailed roadmap docs (provider telemetry/alerting, RBAC, self-service keys, guardrails, plugin tooling with MCP examples).
- `docker-compose.yml` health checks and conditional dependencies so the router waits for Postgres/Redis.

### Changed
- Dockerfile now provides default platform args and release workflow builds/pushes `linux/amd64` + `linux/arm64` images.
- Admin/user login pages show placeholders instead of prefilled demo credentials.
- README/architecture docs mention multi-provider image support and display the project logo; README also links to the GLWT license.
- Local compose config listens on `8090` to match the forwarded port.

### Fixed
- Release workflow caches Go modules by pointing setup-go at `backend/go.mod`/`go.sum`.
- Router container no longer fails to find its config when using docker compose.

## [v0.1.1] - 2025-11-14
### Added
- Theme preference storage (`light`, `dark`, `system`) persisted per user, shared by admin and user portals with a unified theme provider.
- Open Model Gateway logomark across admin/user sidebars, login pages, and favicon for consistent branding.
- Multi-architecture Docker build support (linux/amd64 + linux/arm64) via BuildKit-aware Dockerfile.

### Changed
- Dashboard provider icons now honor light/dark variants, improving contrast in dark mode.

## [v0.1.0] - 2025-11-13
### Added
- Initial release of the Open Model Gateway router, including the Go backend, React admin UI, provider routing, tenant/key management, budgets, usage tracking, and supporting docs.

[v0.1.8]: https://github.com/ncecere/open_model_gateway/compare/v0.1.7...v0.1.8
[v0.1.7]: https://github.com/ncecere/open_model_gateway/compare/v0.1.6...v0.1.7
[v0.1.6]: https://github.com/ncecere/open_model_gateway/compare/v0.1.5...v0.1.6
[v0.1.5]: https://github.com/ncecere/open_model_gateway/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/ncecere/open_model_gateway/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/ncecere/open_model_gateway/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/ncecere/open_model_gateway/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/ncecere/open_model_gateway/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/ncecere/open_model_gateway/releases/tag/v0.1.0

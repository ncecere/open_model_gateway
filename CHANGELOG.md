# Changelog

All notable changes to this project will be documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to Semantic Versioning.

## [Unreleased]
### Added
- Completed the OpenAI-compatible batches surface: `/v1/batches` now supports create/list/retrieve/cancel plus output/error downloads, admin tenant views show batch history, and files are persisted via the existing blob backends.
- Introduced tenant-level RPM/TPM/parallel overrides with new schema (`tenant_rate_limits`), admin API endpoints (`GET/PUT/DELETE /admin/tenants/:id/rate-limits`), and UI controls so every API key inherits the stricter of global, tenant, and per-key caps automatically.
- Admin and user portals now allow operators to set per-key budget + RPM/TPM/parallel overrides when issuing API keys, with inline validation against tenant and global ceilings.
- Admin portal sidebar reordered to highlight Models/Tenants ahead of API Keys, and the user portal navigation now mirrors the same grouping (Dashboard → Models → Tenants → API Keys → Usage → Files → Batches) for consistency.
- Budget/rate-limit inputs in both portals now display the effective max values via placeholders, and the backend rejects any per-key budgets or rate overrides that exceed tenant/global ceilings.

## [v0.1.3] - 2025-02-20
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

## [v0.1.2] - 2025-02-20
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

## [v0.1.1] - 2025-02-20
### Added
- Theme preference storage (`light`, `dark`, `system`) persisted per user, shared by admin and user portals with a unified theme provider.
- Open Model Gateway logomark across admin/user sidebars, login pages, and favicon for consistent branding.
- Multi-architecture Docker build support (linux/amd64 + linux/arm64) via BuildKit-aware Dockerfile.

### Changed
- Dashboard provider icons now honor light/dark variants, improving contrast in dark mode.

## [v0.1.0] - 2025-02-20
### Added
- Initial release of the Open Model Gateway router, including the Go backend, React admin UI, provider routing, tenant/key management, budgets, usage tracking, and supporting docs.

[Unreleased]: https://github.com/ncecere/open_model_gateway/compare/v0.1.2...HEAD
[v0.1.2]: https://github.com/ncecere/open_model_gateway/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/ncecere/open_model_gateway/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/ncecere/open_model_gateway/releases/tag/v0.1.0

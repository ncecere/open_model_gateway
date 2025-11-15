# Changelog

All notable changes to this project will be documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to Semantic Versioning.

## [Unreleased]
### Added
- Guardrail webhook moderation adapter: tenants and API keys can now configure a webhook provider (URL, auth header/value, timeout) to classify prompts/responses; admin UI exposes the new controls.
- Guardrail events feed (`/admin/guardrails/events`) with tenant/API key/action/stage filters and a corresponding Usage → Guardrails tab in the admin portal.
- Guardrail alerts reuse budget alert channels (email/webhook) so blocked requests trigger notifications with cooldown enforcement.

### Changed
- Usage dashboard summary now shows a "Guardrail blocks" KPI and includes guardrail counts in top tenants/users/models lists; daily usage points expose `guardrail_blocks` for charting.
- Tenant/API key guardrail dialogs reorganized to surface moderation provider/action + webhook settings inline, matching the new backend capabilities.

### Docs
- `docs/runtime/guardrails.md` now documents the webhook contract, events feed, and guardrail alerts; README highlights the guardrail UI/alerting improvements.

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

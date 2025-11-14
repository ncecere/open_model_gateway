# Changelog

All notable changes to this project will be documented in this file. The format is inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres to Semantic Versioning.

## [Unreleased]
_No changes yet._

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

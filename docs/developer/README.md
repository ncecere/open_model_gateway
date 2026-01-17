# Developer Guide

This guide describes how to extend Open Model Gateway, from architecture concepts to day-to-day workflows, contribution expectations, and deployment surfaces.

## Clarify purpose and goals

- Provide an OpenAI-compatible proxy that routes requests across OpenAI, Azure, Anthropic, Bedrock, Vertex, OpenRouter, Groq, and other OpenAI-style providers.
- Guarantee tenant isolation with virtual API keys, scoped roles, per-tenant model entitlements, and budget/rate-limit enforcement.
- Capture usage telemetry (tokens, costs, errors, latency) and emit OTEL/Prometheus signals for downstream monitoring.
- Deliver React/Vite portals so operators and end users can manage tenants, models, limits, alerts, and usage without CLI access.

## Architecture overview

| Layer | Description |
|-------|-------------|
| Go/Fiber backend | `cmd/routerd` boots config, migrations, provider registry, HTTP servers for `/v1`, `/admin`, and `/user`, plus background workers (health monitor, usage pipeline, telemetry exporters). |
| Postgres | Stores tenants, users, API keys, rate/budget overrides, usage rows, provider incidents, audit logs, and system settings (managed via Goose + SQLC). |
| Redis | Tracks rate-limit counters, idempotency tokens, provider health snapshots, and distributed locks for usage/budget enforcement. |
| Providers | Adapter interfaces (chat, embeddings, images, audio, files, batches) fan out to OpenAI, Azure, Anthropic, Bedrock, Vertex, OpenRouter, Groq, or any OpenAI-compatible upstream with routing weights, failover cooldowns, and retry policies. |
| React/Vite frontend | Admin/user portals share UI kit primitives, call the backend with JWT or session cookies, and surface dashboards for usage, budgets, provider health, models, keys, and tenants. |
| Observability | OTLP exporters ship traces/metrics/logs; Prometheus `/metrics` exposes gauge/counter panels for health, rate-limit, and budget state. |

## Configuration structure

- Source of truth: YAML file (typically `deploy/router.local.yaml`) merged with environment variables (`ROUTER_*`).
- Key sections: `server`, `database`, `redis`, `providers.<slug>`, `model_catalog`, `rate_limits`, `budgets`, `bootstrap`, `observability`, `health`, `admin`.
- Provider definitions register through `internal/providers` and optionally load secret overrides from ENV (e.g., `ROUTER_PROVIDERS_AZURE_OPENAI_KEY`).
- `bootstrap` is idempotent; edits re-sync tenants, keys, default models, and limits on restart.
- Store per-environment overlays (dev/stage/prod) under `deploy/` or `docs/admin/runtime/` to keep reviewed configs alongside the repo.

## Daily workflows

### Local prerequisites

```bash
make compose-up          # Postgres + Redis via Docker
bun install --cwd backend/frontend
cp deploy/router.example.yaml deploy/router.local.yaml
export ROUTER_CONFIG_FILE=$(pwd)/deploy/router.local.yaml
```

### Run the stack

- `make run-backend` builds the UI and runs `go run ./cmd/routerd` with config/env helpers.
- `make run-backend CONFIG=/path/to/custom.yaml` swaps configs; use `ROUTER_DB_URL` / `ROUTER_REDIS_URL` overrides for remote services.
- Frontend HMR: `cd backend/frontend && bun run dev --host` then point `VITE_GATEWAY_BASE_URL` to the running backend.

### Code generation & database changes

1. Update SQL in `sql/queries/*.sql` or create migrations via `goose -dir migrations create feature_name sql`.
2. Run `sqlc generate` from `backend/` (targets `internal/db`).
3. Re-run `go test ./internal/...` to confirm updated structs compile.
4. Seed fixtures through `backend/cmd/generateproviderfixtures` if provider metadata changes.

### Testing strategy

- Unit: `make test-backend` (runs `go test ./...` under `backend/` with race detector optional via `GOFLAGS="-race"`).
- Integration: `make test-admin-ui` (Vitest) and `make test-e2e` (Playwright) once the frontend assets build.
- Contract: `backend/cmd/inspectroutes` ensures `/v1` parity with OpenAI; `Code_Examples/` smoke scripts should run clean before PRs.
- Lint (pending CI wiring): `golangci-lint run` and `bun run lint` for the frontend. Document extra linters under `docs/developer/linting.md` if added.

## Contribution expectations

- Branch naming: `feat/<area>-<slug>`, `fix/<bug-id>`, or `docs/<topic>`; reference work items when available.
- Keep PRs scoped (backend vs frontend vs docs) and mention cross-cutting impacts in the description.
- Always run `make test-backend` and the relevant frontend tests before pushing; attach output snippets in the PR.
- Document new config knobs or provider behaviors in `docs/admin/runtime/config.md` and `docs/architecture/providers/adding.md`.
- Update `Code_Examples/` when adding endpoints or headers so operators have copy/pasteable snippets.
- Use `docs:`-prefixed commit messages for documentation-only changes to keep history readable.

## Deployment targets

| Target | Notes |
|--------|-------|
| Single binary | Ships with embedded UI and migrations. Suitable for VM/bare-metal installs where operators manage Postgres/Redis externally. Use `ROUTER_CONFIG_FILE` + ENV secrets. |
| Docker Compose | `deploy/docker-compose.yml` orchestrates router + Postgres + Redis + OTEL collector for local or small prod environments. Includes `docker-compose.dev.yml` for source builds. |
| Kubernetes (DIY) | Use the published GHCR image, mount `router.yaml` via ConfigMap/Secret, and supply Postgres/Redis services. Observability integrates via OTLP + Prometheus annotations. Helm/Terraform scaffolds will land under `deploy/` per roadmap. |
| Managed DB/cache | Point `ROUTER_DB_URL` and `ROUTER_REDIS_URL` at managed services; disable `database.run_migrations` if migrations happen elsewhere. |

## Documentation organization guidance

- `docs/developer/` should host process docs (this guide), lint/test references, and architecture deep dives. Suggested structure:
  - `docs/developer/README.md` – overview (this file).
  - `docs/developer/workflows.md` – optional drill-down on SQLC, providers, telemetry (future split).
  - `docs/developer/contributing.md` – reuse contribution section when policies grow.
- `docs/admin/runtime/` remains the authoritative config reference (sample YAMLs + explanations).
- `docs/developer/backend.md` and `docs/developer/providers/` capture backend + provider diagrams/notes; keep onboarding steps centralized there.
- `docs/admin/` now owns admin, tenant, and user guides (see subfolders). Cross-link from README + developer docs whenever exposing UX flows.
- Keep `Code_Examples/` in sync with docs by referencing script names in guides (e.g., `curl/chat.sh`, `curl/admin-tenants.sh`).

## Troubleshooting

- **Router fails on boot**: confirm migrations ran (`goose status`) and `bootstrap` data references existing tenants/providers.
- **Provider 5xx spikes**: inspect `/admin/providers` UI or Redis health cache, adjust routing weights, or disable the provider in the catalog.
- **Budget denies unexpected calls**: check `usage_budget` tables, confirm UTC vs tenant timezone conversions, and ensure rate/budget overrides were synced after config edits.
- **Frontend auth loops**: ensure HTTPS termination preserves cookies (`Secure`, `SameSite=None`) and that admin/user session secrets differ between environments.

Reach out in `agents.md` for open questions or to log new research tasks (e.g., tokenizer libraries, retry policies).

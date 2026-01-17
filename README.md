# <img src="backend/frontend/src/assets/system/open_model_gateway.svg" alt="Open Model Gateway logo" width="64" height="64" style="vertical-align:middle;margin-right:8px;"> Open Model Gateway

Open Model Gateway is a programmable inference router that speaks the OpenAI API while adding tenant isolation, multi-provider routing, budget controls, and usage metering.

## Overview

- Go/Fiber backend handles routing, rate/budget enforcement, usage logging, and provider failover.
- React/Vite admin + user portals surface tenant management, API key issuance, model catalogs, and telemetry views.
- Postgres tracks tenants, usage, incidents, and budget states; Redis backs rate limits, idempotency, and health probes.
- OTEL + Prometheus exporters provide traces/metrics so downstream observability stacks can ingest router health.

The gateway proxies OpenAI-style requests to OpenAI, Azure OpenAI, Anthropic, AWS Bedrock, Vertex, OpenRouter, Groq, or any OpenAI-compatible deployment while keeping tenants isolated.

## When to use

- Need consistent OpenAI-compatible endpoints with automatic failover across multiple providers/regions.
- Want to enforce per-tenant/per-key budgets, rate limits, and alerting without re-implementing accounting.
- Require virtual API keys plus admin/user portals for issuing secrets, rotating credentials, and auditing actions.
- Need detailed cost/usage telemetry exported through OTEL/Prometheus while keeping infra self-hosted.

## Key capabilities

- OpenAI-compatible `/v1/*` surface (chat, embeddings, moderations, images, audio, files, batches, responses) plus `/admin/*` automation APIs.
- Provider catalog that merges static config + persisted overrides with health-aware weighted routing and failover cooldowns.
- Tenant isolation enforced via API keys and admin access tokens, including per-key budgets, per-tenant rate limits, and alerting hooks.
- Usage metering and cost computation persisted per request, exposed via headers and exported through OTEL/Prometheus for dashboards.
- React/Vite admin + user portals for catalog edits, tenant onboarding, API-key issuance, and spend visibility without touching YAML.

## Repository layout

```
/
├── backend/          # Go router, providers, SQLC layer, embedded UI assets
│   └── frontend/     # React/Vite admin + user portals built with Bun
├── deploy/           # Docker Compose stacks, OTEL configs, router.example.yaml
├── migrations/       # Goose migrations synced with sql/ schema
├── docs/             # Developer references plus admin, tenant, and end-user playbooks
├── Code_Examples/    # Copy/paste curl/Python/TypeScript samples
└── agents.md         # Coordination log
```

## Prerequisites

| Component | Requirement |
|-----------|-------------|
| Go toolchain | 1.25+ (router build/test)
| Bun | 1.1+ (frontend build + dev server)
| Postgres | 14+ (schema uses generated columns + enums)
| Redis | 7+ (rate limits, idempotency, health cache)
| Tooling | `goose`, `sqlc`, `make`, Docker (optional for stack)

Install helper CLIs:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Install & deploy

### Binary release workflow

1. Download the latest `open-model-gateway_<tag>_<os>_<arch>.tar.gz` from [Releases](https://github.com/ncecere/open_model_gateway/releases).
2. Extract into `/opt/open-model-gateway` (or another directory) and copy `deploy/router.local.yaml` to your desired config path.
3. Set environment overrides (database URL, Redis URL, OTEL target, provider credentials):

   ```bash
   export ROUTER_CONFIG_FILE=/opt/open-model-gateway/router.yaml
   export ROUTER_DB_URL=postgres://user:pass@db:5432/open_gateway?sslmode=disable
   export ROUTER_REDIS_URL=redis://cache:6379/0
   ```

4. Run the binary; migrations run on boot when `database.run_migrations` is true:

   ```bash
   cd /opt/open-model-gateway
   ./router
   ```

### Docker Compose workflow

1. Update `deploy/router.local.yaml` with bootstrap tenants, budgets, provider keys, and OTEL endpoints.
2. From `deploy/`, launch the stack:

   ```bash
   docker compose up -d
   ```

   - `router` container reads `/config/router.yaml` (bind-mount of `router.local.yaml`).
   - Postgres, Redis, and OTEL collector stand up alongside the router.
   - Use `docker compose -f docker-compose.dev.yml up --build` to rebuild from local sources.

3. For multi-arch images, run `docker buildx build --platform linux/amd64,linux/arm64 --push ghcr.io/<org>/open_model_gateway:latest .`.

Refer to `docs/deployment/releases.md` for publishing and `deploy/docker-compose.dev.yml` for local iteration flags.

## Configuration highlights

| Section | Purpose |
|---------|---------|
| `server` | Listener, idle/read/write timeouts, streaming idle guard, graceful shutdown delay. |
| `database` | Connection string, pool sizes, migration directory, run-on-boot toggle. |
| `redis` | URL/db for rate limits, idempotency, and provider health cache. |
| `providers.<slug>` | API base URLs, keys, retry knobs for OpenAI, Azure, Anthropic, Bedrock, Vertex, OpenRouter, Groq, Hugging Face. |
| `model_catalog` | Aliases, provider bindings, weights, per-model pricing, modality metadata. |
| `rate_limits` | Default RPM/TPM/parallel caps plus overrides seeded via `bootstrap.rate_limits`. |
| `budgets` | Default USD budget, refresh cadence (rolling/weekly/calendar), alert channels (email/webhook) + cooldowns. |
| `bootstrap` | Declarative tenants, admin users, memberships, keys, per-tenant limits/budgets, and default models. |
| `observability` | OTLP exporter toggle/endpoint, Prometheus `/metrics`, sampling settings. |

See `docs/admin/runtime/router-example.md` for a fully annotated reference plus comments for each field.

## Quick verification

1. Run `go run ./cmd/routerd` (or `docker compose up`) with the sample config.
2. Use the seeded key `sk-demo.my-secret` from `bootstrap.api_keys`:

```bash
curl -s http://localhost:8090/v1/models \
  -H "Authorization: Bearer sk-demo.my-secret" | jq

curl -s http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-5-mini",
        "messages": [
          {"role": "system", "content": "You are a friendly assistant."},
          {"role": "user", "content": "Give me a short status update on the gateway service."}
        ],
        "stream": false
      }'

curl -s http://localhost:8090/v1/embeddings \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "text-embedding-3-small",
        "input": "The quick brown fox jumps over the lazy dog."
      }'

curl -s http://localhost:8090/v1/images/generations \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-image-1-mini",
        "prompt": "A futuristic research laboratory overlooking a neon city skyline"
      }' \
  | jq -r '.data[0].b64_json' | base64 --decode > gpt-image.png
```

3. Verify admin auth by calling `POST /admin/auth/login` with seeded credentials and confirm `Authorization: Bearer <token>` works for `/admin/tenants`.
4. Check OTEL exporter hits `http://localhost:4318` (default) and Prometheus metrics show up at `/metrics`.

## Roadmap summary

- Expand provider coverage (Cerebras, Google AI Studio, Ollama, vLLM) with capability-aware routing metadata.
- Ship tenant guardrails policy engine and user-facing consent flows for sensitive models.
- Harden autoscaling: queue-aware retry/backoff, circuit-breaker tuning per provider, and SLA-driven health scoring.
- Finish CI contract tests mirroring OpenAI responses plus Playwright regression for admin/user portals.
- Publish Terraform + Helm starting points that reuse the Docker image + `router.example.yaml` config structure.

## Additional resources

- `docs/developer/README.md` – end-to-end developer workflows, config structure, and contribution guide.
- `docs/admin/install.md` – system administrator install/upgrade guide.
- `docs/admin/runtime/README.md` – runtime config reference and bootstrap annotations.
- `docs/admin/` – onboarding guides for admins, tenant owners, and standard users.
- `docs/personal/guide.md` – personal tenant and API key walkthrough.
- `Code_Examples/` – curl/Python/TypeScript snippets that hit models, chat, embeddings, images, audio, and admin APIs.
- `CHANGELOG.md` – feature timeline and migration hints between releases.

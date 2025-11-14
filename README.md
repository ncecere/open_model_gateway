# Open Model Gateway

Open Model Gateway is a programmable inference router that exposes an OpenAI-compatible API surface with tenant isolation, usage metering, and multi-provider failover. The project coordinates a Go/Fiber backend, a React/Vite admin UI, and supporting infrastructure (Postgres, Redis, OTEL/Prometheus) to deliver an enterprise-grade LLM gateway.

## Repository Layout

```
/
├── backend/          # Go 1.25 router service (Fiber, SQLC, Redis, OTEL)
│   └── frontend/     # React + Vite admin & user portals (built via Bun)
├── deploy/           # Local tooling (docker-compose, router.local.yaml, OTEL config)
├── migrations/       # Database migrations (managed via Goose)
├── agents.md         # Agent coordination journal (working log)
├── prd.md            # Product requirements document
└── docs/             # Supplemental documentation (config, architecture, UI)
    └── runtime/router.example.yaml
```

## Current Capabilities (v1 backlog in flight)

- OpenAI-compatible public API:
  - `GET /v1/models`
  - `POST /v1/chat/completions` (including SSE streaming)
- `POST /v1/embeddings`
- `POST /v1/images/generations` (Azure `gpt-image-1`, base64 responses)
- `POST /v1/audio/{transcriptions,translations,speech}` (Whisper + GPT-4o-mini-tts text-to-speech)
- Provider routing & failover:
  - Model catalog merge between static config and persisted overrides
  - Azure OpenAI adapter (chat + embeddings) as the first supported provider
  - Weighted routing with health-state tracking and failover cooldowns
- Tenant access control:
  - Virtual API keys (`sk-<prefix>.<secret>`) with hash verification
  - Redis-backed idempotency cache and rate limiting (RPM per model alias)
- Budget enforcement with configurable default limits, per-tenant overrides, rolling/weekly windows, and alert routing (email/webhook with cooldowns)
- Usage metering:
  - Request + usage tables populated per call (tokens, latency, cost)
  - Cost computation derived from model catalog pricing (per 1K tokens)
  - Budget status headers (`X-Budget-*`) returned on every response
- Admin surface (protected by JWT access tokens):
  - Auth: local credentials + OIDC SSO, refresh token rotation, secure cookies
  - Model catalog CRUD (aliases, deployments, pricing metadata)
  - Tenant CRUD (create, status updates) and API key lifecycle (issue/revoke)
- Config loader with YAML + `.env` merge and strict validation of runtime defaults.

See [`docs/backend-status.md`](docs/backend-status.md) for a detailed feature inventory and upcoming tasks.

## Installation

You can either download a prebuilt release bundle (router binary + migrations + sample config) or run the full stack via Docker Compose. Both flows expect Postgres, Redis, and a config file that defines provider credentials and bootstrap data.

### Option A: Download a Release Bundle

1. Grab the latest archive from the [GitHub Releases page](https://github.com/ncecere/open_model_gateway/releases) (each tag publishes `open-model-gateway_<tag>_linux_amd64.tar.gz`).
2. Unpack it somewhere on your host:

   ```bash
   tar -xzf open-model-gateway_<tag>_linux_amd64.tar.gz -C /opt/open-model-gateway
   ```

   The tarball contains the `router` binary, `backend/migrations`, and `deploy/router.local.yaml` as a starting config.

3. Edit `/opt/open-model-gateway/deploy/router.local.yaml` (or copy it elsewhere) to supply your database URLs, Redis URL, OTEL endpoint, provider keys, and bootstrap tenants.

4. Run the binary, pointing at the config file:

   ```bash
   cd /opt/open-model-gateway
   ROUTER_CONFIG_FILE=/path/to/router.yaml \
   ROUTER_DB_URL=postgres://user:pass@host:5432/open_gateway?sslmode=disable \
   ROUTER_REDIS_URL=redis://host:6379/0 \
   ./router
   ```

   Migrations execute on boot when `database.run_migrations` is `true`. You can also run the bundled SQL with Goose by pointing it at `backend/migrations`.

### Option B: Run via Docker Compose

The `deploy/docker-compose.yml` file now includes the router service alongside Postgres, Redis, and the OTEL collector. After filling out `deploy/router.local.yaml`, run:

```bash
cd deploy
docker compose build router
docker compose up -d
```

This will build the multi-stage image defined in the root `Dockerfile`, seed the migrations inside the container, and expose the admin/public APIs on `http://localhost:8090`. The router service automatically reads `/config/router.yaml`, which is a bind-mount of `deploy/router.local.yaml`.

Refer to [`docs/deployment/releases.md`](docs/deployment/releases.md) for more detail on the release/packaging pipeline and the GitHub Container Registry images.

## Getting Started

### Prerequisites

- Go **1.25** or newer
- Bun (for the React admin UI; WIP)
- PostgreSQL 14+
- Redis 7+
- `goose` and `sqlc` binaries on your `PATH`:

  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
  ```

### Backend Bootstrap (Source Build)

1. Copy `docs/runtime/router.example.yaml` to `deploy/router.local.yaml` (or provide your own `ROUTER_CONFIG_FILE`).
2. Populate required secrets:

   ```bash
   export ROUTER_DB_URL="postgres://user:pass@localhost:5432/open_gateway?sslmode=disable"
   export ROUTER_REDIS_URL="redis://localhost:6379/0"
   export ROUTER_ADMIN_SESSION_JWT_SECRET="change-me"
   ```

   Optional provider/env overrides include `ROUTER_PROVIDERS_AZURE_OPENAI_ENDPOINT`, `ROUTER_PROVIDERS_AZURE_OPENAI_KEY`, etc.

3. Run the router service:

   ```bash
   cd backend
   go run ./cmd/routerd
   ```

   Migrations execute automatically on boot when `database.run_migrations` is `true`. To run them manually:

   ```bash
   goose -dir migrations postgres "$ROUTER_DB_URL" up
   ```

4. Verify the health check:

   ```
   curl http://localhost:8080/healthz
   ```

### Bootstrap Users & API Keys

`deploy/router.local.yaml` demonstrates the optional `bootstrap` section:

```
bootstrap:
  tenants:
    - name: "demo"
  admin_users:
    - email: "admin@example.com"
      name: "Demo Admin"
      password: "admin-password"
  api_keys:
    - tenant: "demo"
      prefix: "demo"
      secret: "my-secret"
      name: "Demo API Key"
      rate_limit:
        requests_per_minute: 60
        tokens_per_minute: 60000
        parallel_requests: 5
  memberships:
    - tenant: "demo"
      email: "admin@example.com"
      role: "owner"
  tenant_limits:
    - tenant: "demo"
      limits:
        requests_per_minute: 120
        tokens_per_minute: 120000
        parallel_requests: 20
  tenant_budgets:
    - tenant: "demo"
      budget_usd: 150.0
      warning_threshold: 0.75
      refresh_schedule: "weekly"
      alert_emails:
        - "finance@example.com"
      alert_cooldown: "90m"
```

On startup the router ensures tenants exist, seeds admin users (hashing the plaintext password), creates API keys (storing the hashed secret), grants memberships (owner/admin/viewer/user), and applies optional rate-limit overrides per key/tenant. The example above yields the API key `sk-demo.my-secret` for testing, makes `admin@example.com` the owner of the `demo` tenant, and caps that tenant at 120 RPM / 120k TPM / 20 parallel requests (with a stricter per-key override of 60 RPM / 60k TPM / 5 parallel).

### Quick Curl Smoke Tests

With the bootstrap API key you can exercise the Azure-backed endpoints directly:

```bash
curl -s http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-demo.my-secret" | jq

curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-5-mini",
        "messages": [
          {"role": "system", "content": "You are a friendly assistant."},
          {"role": "user", "content": "Give me a short status update on the gateway service."}
        ]
      }'

curl -s http://localhost:8080/v1/embeddings \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "text-embedding-3-small",
        "input": "The quick brown fox jumps over the lazy dog."
      }'

curl -s http://localhost:8080/v1/images/generations \
  -H "Authorization: Bearer sk-demo.my-secret" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-image-1-mini",
        "prompt": "A futuristic research laboratory overlooking a neon city skyline"
      }' \
  | jq -r '.data[0].b64_json' \
  | base64 --decode > gpt-image.png
```

The final command writes `gpt-image.png`; open it locally to verify image generation.

### Admin Authentication Flow (local credentials)

1. Seed an admin user/credential via SQLC helpers (example script forthcoming).
2. Hit `POST /admin/auth/login` with JSON `{"email":"...","password":"..."}` to receive an access/refresh token pair.
3. Use the returned access token as `Authorization: Bearer <token>` for any `/admin/**` requests. Refresh tokens are stored in an HTTP-only cookie (`og_admin_session` by default).

### Configuration Highlights

| Section             | Key Fields                                                                                   |
|---------------------|-----------------------------------------------------------------------------------------------|
| `server`            | Timeouts (sync/stream), body size, graceful shutdown delay                                    |
| `database`          | Connection string, pool sizes, migration directory, run-on-boot flag                          |
| `redis`             | URL, logical DB, pool size                                                                    |
| `rate_limits`       | Default RPM/TPM caps and parallel request constraints (overrides via `bootstrap.*`)          |
| `budgets`           | Default budget (USD), warning threshold, refresh cadence (calendar/weekly/rolling), alert defaults (enabled/emails/webhooks/cooldown) |
| `providers`         | Credential slots (OpenAI, Azure OpenAI, Anthropic, Bedrock, Vertex, Hugging Face)             |
| `model_catalog`     | Alias metadata (deployment, endpoint, per-model pricing, weight, modalities, metadata)        |
| `observability`     | OTLP endpoint toggle and Prometheus metrics flag                                              |
| `health`            | Interval/cooldown for background provider probes                                              |
| `admin`             | JWT secrets + TTLs, local/oidc feature toggles, OIDC client configuration                     |

Refer to [`docs/runtime/router.example.yaml`](docs/runtime/router.example.yaml) for a fully annotated sample.

## Development Workflows

- **Code generation**: `sqlc generate` after editing `sql/queries/*.sql`.
- **Formatting**: `gofmt` for Go, `bun format` (pending) for the frontend.
- **Testing**: `go test ./...` (Redis/Postgres not required for unit compilation).
- **Agents**: Coordination, outstanding TODOs, and feature notes live in `agents.md`.

## Roadmap Snapshot

- Provider coverage: Anthropic, AWS Bedrock, Google Vertex, Hugging Face embeddings.
- Observability: OTLP exporter wiring, `/metrics` Prometheus endpoint, latency/error dashboards.
- Frontend: Admin UI (usage dashboards, budgets, health views) served from `frontend/` via Bun.
- Rate limiting: Per-key TPM and parallel request semaphores derived from quota metadata.
- Tokenization backfills: Anthropic/Llama token counters for accurate metering fallback.
- DevOps: Docker Compose stack, Terraform scaffolding, CI smoke tests and contract suite.

For the full requirement set consult [`prd.md`](prd.md). Progress updates and coordination details continue to accumulate in [`agents.md`](agents.md).

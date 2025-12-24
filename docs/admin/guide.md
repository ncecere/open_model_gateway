# Platform administrator guide

Use this handbook when you are responsible for deploying, operating, and auditing Open Model Gateway. It consolidates every operator-facing workflow—deployment, defaults, model catalogs, budgets, automation, and troubleshooting—so you can run the router without jumping between multiple docs.

## Overview & responsibilities

- Keep the router healthy (Postgres, Redis, router binary/containers, OTEL exporters, optional object storage).
- Manage provider credentials and model catalogs so tenants have the right aliases with failover policies.
- Onboard tenants/admins, grant roles, and issue scoped API keys with budgets + rate limits.
- Respond to incidents (budget exhaustion, provider failures, auth errors) and keep stakeholders informed.
- Automate repetitive admin flows using the `/admin/*` API surface or the provided curl scripts.

### Architecture & dependencies

| Component | Purpose |
| --- | --- |
| Go/Fiber backend (`routerd`) | Hosts `/v1/*`, `/admin/*`, `/user/*`, background workers (batches, file sweeper), and OTEL exporters. |
| React/Vite portals | Admin & user SPAs embedded inside the binary (built via `make build-ui`). |
| Postgres 15+ | Tenants, usage, budgets, provider incidents, catalog state, admin tokens. |
| Redis 6+ | Rate limits, idempotency, provider health cache, session helpers. |
| Object storage (local/S3) | `/v1/files` uploads and batch output/error NDJSON. |
| OTEL collector / Prometheus | Optional tracing + metrics sinks for observability. |

## Deploy & upgrade

### Option A – release bundle / systemd

1. Download `open-model-gateway_<tag>_<os>_<arch>.tar.gz` from GitHub Releases.
2. Extract into `/opt/open-model-gateway` (or another prefix) and copy `deploy/router.local.yaml` to your config path.
3. Set ENV secrets:
   ```bash
   export ROUTER_CONFIG_FILE=/etc/open-model-gateway/router.yaml
   export ROUTER_DB_URL=postgres://user:pass@db:5432/open_gateway?sslmode=disable
   export ROUTER_REDIS_URL=redis://cache:6379/0
   export ROUTER_ADMIN_SESSION_JWT_SECRET=<hmac-secret>
   ```
4. Run migrations (let `routerd` run them via `database.run_migrations: true` or invoke `goose up`).
5. Supervise with systemd, Nomad, or Kubernetes and expose the chosen `server.listen_addr` (default `:8090`).
6. Smoke test using [Code_Examples/curl](../../Code_Examples/curl/README.md) (`models.sh`, `chat.sh`, `admin.sh`).

### Option B – Docker / container platforms

1. Fill `deploy/router.local.yaml` with provider keys, bootstrap tenants, budgets, rate limits, and OTEL targets.
2. Start the stack:
   ```bash
   cd deploy
   docker compose up -d
   ```
3. Use `docker compose -f docker-compose.dev.yml up --build` to bake local edits.
4. Publish multi-arch images via `docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/<org>/open_model_gateway:latest --push .`.

### Configuration hygiene

- Keep env-specific configs under `deploy/` and reference them with `ROUTER_CONFIG_FILE`.
- Store secrets (DB/Redis URLs, provider keys, JWT secrets) in your secret manager and inject via ENV (`ROUTER_DB_URL`, `ROUTER_PROVIDERS_AZURE_OPENAI_KEY`, etc.).
- Use the `bootstrap.*` block for idempotent tenants/admins/keys. Restarting applies deltas safely.
- Document any overrides from `docs/runtime/router.example.yaml` in your runbooks so on-call responders know what changed.
- Upgrades: rebuild `routerd` after `make build-ui`, redeploy, and let migrations run (`database.run_migrations: true`) or run `goose` manually. Catalog edits hot-reload; send `SIGHUP` or restart after YAML changes.

## Configure defaults & runtime settings

`docs/runtime/config.md` lists every key. Operators regularly touch the sections below:

| Block | What to tune | Notes |
| --- | --- | --- |
| `rate_limits.*` | Default RPM/TPM/parallel ceilings for tenants + keys. | Tenant overrides can only lower the ceiling; keys never exceed tenant limits. |
| `budgets.*` | Default USD budget, refresh cadence (`calendar_month`, `weekly`, `rolling_30d`), alert channels (`emails`, `webhooks`). | Alerts require SMTP/webhook config; see “Budgets & alerts”. |
| `providers.*` | Shared credentials/endpoints (OpenAI, Azure, Bedrock, Vertex, Anthropic, OpenRouter, Groq, vLLM, openai-compatible). | Catalog entries can override per alias via metadata. |
| `files.*` | Upload size, TTLs, storage backend (local/S3), encryption key, sweeper cadence. | Applies to `/v1/files` and batch output/error artifacts. |
| `batches.*` | `max_requests`, `max_concurrency`, TTLs for output files. | Keep within Redis/Postgres capacity. |
| `observability.*` | OTEL + Prometheus toggles, endpoints, sampling, metrics enablement. | Enable TLS for external collectors. |
| `admin.*` | Session TTLs, local/SSO auth, OIDC roles. | `allowed_roles` gates login; `admin_roles` map IdP groups to super-admin. |
| `bootstrap.*` | Seed tenants, admin users, memberships, API keys, budgets, limits. | Idempotent; edit YAML + restart to rotate secrets or add bootstrap keys. |

## Manage tenants, memberships, and keys

| Task | Portal path / API |
| --- | --- |
| Create tenant | **Admin → Tenants → New** or `POST /admin/tenants` |
| Manage members/roles | **Admin → Tenants → Members** or `POST/DELETE /admin/tenants/{id}/memberships` |
| Invite admins | **Admin → Users → Invite** or `POST /admin/users` |
| Issue API keys | **Admin → Tenants → API Keys** or `POST /admin/tenants/{id}/api-keys` |
| Reset tenant rate limits | **Admin → Tenants → Rate limits → Clear override** or `DELETE /admin/tenants/{id}/rate-limits` |

Guidance:
- Leave tenant rate-limit fields blank to inherit `rate_limits.*`. Overrides clamp all keys in that tenant.
- When creating keys, specify budgets and RPM/TPM/parallel overrides per key; the UI shows the effective ceiling based on tenant + global defaults.
- Audit membership/role changes under **Admin → Settings → Audit log**.
- Tenant owners can self-serve within **Tenants → Members** but only super admins can edit global catalog entries or provider settings.

## Providers & model catalog

1. Load credentials under `providers.<slug>` and keep secrets in ENV.
2. Create aliases via the UI or `model_catalog` YAML, capturing deployment IDs, supported modalities, pricing, routing weight, health policy, and tenant assignment flags.
3. Monitor **Admin → Providers** or `/admin/providers` for health/incidents. The router polls upstreams, records incidents, and automatically down-weights unhealthy deployments.
4. Adjust routing weights or disable affected aliases during incidents. Responses stay OpenAI-compatible, so clients retry without code changes.
5. Reference `docs/admin/model-catalog-examples.md` (copied from the legacy library) for per-provider YAML templates, pricing tiers, and capability overrides.

### Images capability matrix

| Provider | Generations | Edits | Variations | Notes |
| --- | --- | --- | --- | --- |
| OpenAI / openai-compatible | ✅ | ✅ | ✅ | Full Images API parity; use `/v1/files` IDs for edits/variations. |
| Azure OpenAI | ✅ | 🚫 | 🚫 | Only generations for `gpt-image-1`; edits/variations return `image_operation_unsupported`. |
| Bedrock Titan | ✅ | 🚫 | 🚫 | Text-to-image only. |
| Bedrock Stable Diffusion | ✅ | ✅ | ✅ | Configure `metadata.bedrock_image_*` fields for mask/variation behavior. |
| Vertex Imagen / Imagen Nano Banana | ✅ | ✅ | ✅ | Requires service-account JSON; see Vertex provider doc for mask/variation knobs. |

Set `price_image_cents`, `price_image_edit_cents`, and `price_image_variation_cents` metadata fields when you need to override billing for each operation.

### Audio capability matrix

| Provider | `/v1/audio/transcriptions` & `/translations` | `/v1/audio/speech` | Notes |
| --- | --- | --- | --- |
| OpenAI / openai-compatible | ✅ | ✅ | Configure default voice/format with `metadata.audio_voice` / `metadata.audio_format`; supports streaming when `metadata.audio_streaming=true`. |
| Azure, Bedrock, Vertex | 🚫 | 🚫 | Wire additional adapters when providers expose compatible endpoints. |

Define audio aliases with `model_type: audio_transcription` or `audio_speech` so the router enforces modality support.

## Budgets, alerts, and billing workflows

- Default budgets come from `budgets.default_usd`, `warning_threshold_perc`, and `refresh_schedule`. Tenants inherit them unless you add overrides via `bootstrap.tenant_budgets` or the portal.
- Per-key budgets can be set at key creation time; they can never exceed the tenant budget.
- Alert routing:
  - **Email**: configure `budgets.alert.smtp.*` (host, credentials, TLS). Alerts include spend, limit, warning threshold, and tenant metadata.
  - **Webhook**: configure `budgets.alert.webhook.*` (timeout, retries). Payloads mirror the email body and include HMAC signatures.
  - Alert history lives in `budget_alert_events` and will surface in future UI releases.
- Every API response includes `X-Budget-*` and `X-RateLimit-*` headers, so you can verify changes without opening the UI.
- Usage exports + billing webhooks (`/admin/usage-exports`, `/admin/billing-webhooks`) are documented in `docs/runtime/usage.md`. Use them to hand finance teams CSV/Parquet exports or to post monthly summaries into billing systems.

## Storage, files, and batch jobs

### Files

- `files.*` config sets upload limits, TTL, storage backend, encryption, and sweeper cadence.
- `/v1/files` mirrors OpenAI responses (`limit`, `after`, `purpose` filter, `{has_more, first_id, last_id}` envelope). Metadata includes `status` and `status_details` so you can see whether uploads finished processing.
- `DELETE /v1/files/:id` returns `{id, object:"file", deleted:true}` and the sweeper removes blobs once `expires_at` passes.

### Batches

- `/v1/batches` accepts NDJSON job definitions (chat, embeddings, responses, moderations, images). Upload input files via `/v1/files` with `purpose=batch` and reference file IDs in the batch payload.
- Worker throughput is governed by `batches.max_concurrency` and DB pool size. Monitor logs for `batch worker` lines.
- Batch list/detail responses now match OpenAI (cursor pagination, `request_counts`, timestamp fields, `errors` lists, metadata cap of 16 key/value pairs).
- Output/error files are stored via the files subsystem; download them through `/v1/files/{id}/content` or the portal buttons.

## Monitoring, incidents, and analytics

- Health endpoints: `/healthz` (JSON), `/metrics` (Prometheus, gated by `observability.enable_metrics`).
- OTEL: set `observability.enable_otlp=true` and point `observability.otlp_endpoint` at your collector (enable TLS for remote collectors).
- Provider incidents: **Admin → Providers** lists each deployment, last probe, error rate, and any incidents. `/admin/providers` exposes the same data for automation.
- Usage dashboards: **Admin → Usage** and `/admin/usage/compare` overlay tenants/models. Query params `tenant_ids`, `model_aliases`, `period`, `start`, `end`, `timezone` mirror the API (capped at 10 entities per request). The user portal uses `/user/usage/compare` scoped to the caller’s tenants.
- Logs: monitor budget alerts, provider incidents, batch workers, and file sweepers via application logs (structured output includes request IDs).

## Automation & admin API tokens

- Use `POST /admin/auth/login` with JSON `{"email":"...","password":"..."}` to obtain an access token and call `/admin/*` endpoints.
- Admin API tokens (Settings → Admin tokens) remove the need for browser sessions. Tokens are only shown once, scoped to the issuing admin or super-admin, and require an expiry.
  ```bash
  curl -sS -X POST http://localhost:8090/admin/admin-keys \
    -H "Authorization: Bearer $ADMIN_SESSION" \
    -H "Content-Type: application/json" \
    -d '{"name":"ops-export","scope":"system","expires_in_seconds":2592000}'
  ```
  Use the returned token for automation:
  ```bash
  curl -sS http://localhost:8090/admin/usage-exports \
    -H "Authorization: Bearer $ADMIN_TOKEN"
  ```
- Scripts in `Code_Examples/curl` (`admin.sh`, `models.sh`, etc.) provide copy/paste workflows for provisioning tenants, keys, budgets, and batches.
- Rotate service accounts regularly and scope them to the minimum required roles. Record token issuance in your internal inventory.

## Security, backup, and troubleshooting

### Backup & restore

- **Postgres** is the system of record; use `pg_dump`, `pgbackrest`, or managed backups.
- **Redis** is ephemeral. Recreate it and let routerd rebuild state.
- **Files storage**: back up the S3 bucket or `./data/files` directory if you retain long-lived uploads/batch artifacts.

### Security reminders

- Rotate `admin.session.jwt_secret`, provider keys, bootstrap API keys, and Admin API tokens.
- Restrict `/admin/**` via load balancer ACLs when possible.
- Enable TLS for OTLP collectors and any external storage backends.
- Keep `ROUTER_CONFIG_FILE` and ENV overrides outside source control.

### Troubleshooting & useful commands

| Symptom | Actions |
| --- | --- |
| `authorization required` when downloading | Ensure session cookies/headers are forwarded; fall back to `/user/*` APIs with bearer tokens. |
| `provider_unavailable` | Check `/admin/providers`, adjust routing weights, communicate fallback aliases. |
| Batch worker `context_error` | Tenant or key was deleted; inspect `/v1/batches/{id}` metadata for failures. |
| Stale UI after deploy | Re-run `make build-ui` before rebuilding `routerd`. |
| High Redis usage | Investigate bursty keys; revisit `rate_limits.*` to spread load. |

| Command | Purpose |
| --- | --- |
| `make build-ui` | Rebuild embedded admin/user portals. |
| `ROUTER_CONFIG_FILE=/path routerd` | Start routerd with an alternate config (staging, QA, etc.). |
| `goose status` | Inspect migration state before/after deploys. |
| `curl -H "Authorization: Bearer..." http://host:8090/v1/chat/completions` | Smoke test public API with a bootstrap key. |

## Appendices

- **Model catalog examples**: see `docs/admin/model-catalog-examples.md` for provider-specific YAML snippets, capability overrides, and pricing tier templates.
- **Runtime reference**: `docs/admin/runtime/config.md`, `docs/admin/runtime/router-example.md`, and `docs/admin/runtime/usage.md` remain the authoritative sources for config keys, sample configs, and usage export/billing webhook payloads.

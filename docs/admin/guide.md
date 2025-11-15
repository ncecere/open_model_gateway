# Administrator Guide

This guide covers deployment, configuration, and ongoing operations for Open Model Gateway.

## Architecture Recap

- **Backend**: Go 1.25, Fiber, Postgres (SQLC), Redis, optional S3.
- **Frontend**: React + Vite served from the backend binary (embedded assets) plus an admin/user portal.
- **Workers**: Batch worker runs inside `routerd`, sharing the same config/database connections.

External dependencies:

| Component | Purpose |
| --- | --- |
| Postgres 15+ | Metadata, usage, config state. |
| Redis 6+ | Rate limiting, idempotency caches. |
| Object storage (local/S3) | File uploads, batch outputs. |
| OTEL collector (optional) | Exports traces/metrics. |

## Deployment Checklist

1. **Provision Postgres & Redis** – ensure TLS/encryption settings meet your policies.
2. **Compile the binary**:
   ```bash
   make build-ui
   cd backend && CGO_ENABLED=0 go build -o routerd ./cmd/routerd
   ```
3. **Create a config** – copy `docs/runtime/router.example.yaml` to `/etc/open-gateway/router.yaml` (or start from `deploy/router.local.yaml`) and fill in secrets. Use ENV overrides (`ROUTER_DB_URL`, `ROUTER_REDIS_URL`, etc.) for sensitive values.
4. **Run migrations** – either let `routerd` run them on startup (`database.run_migrations: true`) or execute `goose up` manually.
5. **Bootstrap** – configure initial tenants, admin users, API keys via the `bootstrap.*` block. Subsequent changes are idempotent.
6. **Service manager** – wrap `routerd` in systemd, Kubernetes, Nomad, etc. Expose port `8090` (or your chosen `server.listen_addr`).

## Configuration Highlights

- All available keys are documented in `docs/runtime/config.md` and illustrated in `docs/runtime/router.example.yaml` (server/database/redis/observability/files/audio/batches/retention/health/providers/bootstrap).
- `model_catalog` now includes reference entries for Azure GPT-4o, OpenAI `gpt-image-1`, OpenAI-compatible gateways, Bedrock Titan + Stable Diffusion, and Vertex Imagen/Nano Banana models.
- Place secrets in ENV if you use a config repo. Example systemd unit fragment:
  ```
  Environment="ROUTER_CONFIG_FILE=/etc/open-gateway/router.yaml"
  Environment="ROUTER_DB_URL=${SECRET_DB_URL}"
  Environment="ROUTER_REDIS_URL=redis://:pass@cache:6379/0"
  ```
- Common production overrides:
  - `server.body_limit_mb` (raise for larger file uploads)
  - `observability.enable_otlp=true` with a secure collector endpoint
  - `files.storage=s3` with encryption enabled
  - `batches.max_requests` tuned to your workload

### Images API Capability Matrix

| Provider | `/v1/images/generations` | `/v1/images/edits` | `/v1/images/variations` | Notes |
| --- | --- | --- | --- | --- |
| OpenAI / OpenAI-compatible | ✅ | ✅ | ✅ | Uses the native Images API; edits/variations require multipart payloads. |
| Azure OpenAI | ✅ | 🚫 | 🚫 | Azure only supports generations for `gpt-image-1` today; edits/variations return `image_operation_unsupported`. |
| Bedrock (Titan) | ✅ | 🚫 | 🚫 | Titan only supports text-to-image flows. |
| Bedrock (Stable Diffusion) | ✅ | ✅ | ✅ | Enable by setting `metadata.bedrock_image_task_type` to a diffusion model; use `bedrock_image_strength/mask_source/...` to control image-to-image and inpainting. |
| Vertex Imagen / Imagen Nano Banana | ✅ | ✅ | ✅ | Requires service-account JSON; metadata fields (`vertex_edit_mode`, `vertex_mask_mode`, etc.) configure mask editing, guidance, and variation prompts. |

When a provider does not support an operation the router returns a structured error so clients can fall back automatically.

### Audio Capability Matrix

| Provider | `/v1/audio/transcriptions` / `/translations` | `/v1/audio/speech` | Notes |
| --- | --- | --- | --- |
| OpenAI / OpenAI-compatible | ✅ | ✅ | Uses Whisper + TTS (`gpt-4o-mini-tts`/`tts-1`). Configure default voice/format via `metadata.audio_voice` + `metadata.audio_format`. |
| Azure / Bedrock / Vertex | 🚫 | 🚫 | Speech adapters are not wired yet; add follow-up aliases when those providers expose compatible endpoints. |

## Upgrades & Schema Changes

- Binaries embed the UI, so redeploying `routerd` automatically updates the portals.
- Migrations shipped under `backend/migrations/`. Review the changelog, then restart with `database.run_migrations=true` or run `goose` manually.
- When rolling out new provider catalogs, update the YAML and reload (`SIGHUP` or restart). The router hot-reloads model metadata automatically.

## Operations

### Health & Metrics

- `/healthz` – JSON response containing Postgres/Redis status.
- `/metrics` – Prometheus endpoint (guarded by `observability.enable_metrics`).
- OTEL exporter – set `observability.enable_otlp=true` and `observability.otlp_endpoint=https://collector:4317`.

### Managing Tenants & Keys

- Admin portal (`/admin`) lets you manage tenants, rate limits, budgets, model catalog entries, and bootstrap settings.
- Tenant create/edit dialogs expose RPM, TPM, and parallel request inputs. Leaving the fields blank inherits the global defaults (`rate_limits.*`); setting all three persists a tenant-level override via `PUT /admin/tenants/:id/rate-limits`. The cap applies to every key under that tenant before per-key overrides are considered, so keys can never exceed the tenant ceiling.
- Use the “Clear rate limit override” action (or `DELETE /admin/tenants/:id/rate-limits`) to fall back to defaults after tightening limits for an incident.
- API key dialogs let operators specify per-key budgets and RPM/TPM/parallel overrides. The form highlights the effective tenant and global ceilings so you can see the maximum allowed values before issuing the key; the backend enforces the same limits for requests made via the API.
- User portal (`/`) allows non-admin accounts to access personal tenants, API keys, usage dashboards, and batch artifacts.
- API endpoints under `/admin/**` and `/user/**` mirror the UI functionality; use them for automation.

### Files & Storage

- `files.*` config controls storage:
  - `files.sweep_interval` / `files.sweep_batch_size` drive the background sweeper that now runs inside `routerd` to remove expired blobs and mark their metadata as `status="deleted"`. Increase the interval if you retain large data sets, or shrink it when you want near-real-time cleanup.
  - `files.storage=s3` uses the configured bucket/prefix; local mode keeps blobs under `./data/files`.
  - Set `files.encryption_key` for client-side AES-GCM encryption regardless of backend.
  - Monitor the sweeper logs (`files.Service.SweepExpired`) to verify expired objects are being reclaimed; errors are surfaced in the router logs so you can spot misconfigured credentials early.
- `/v1/files` now mirrors the OpenAI response contract: list calls support `limit` (1–100), cursor-based `after` parameters, and purpose filters. Responses include `has_more`, `first_id`, `last_id`, and per-file `status` / `status_details` fields so operators can see whether a file is `uploading`, `uploaded`, `processed`, or `deleted`.
- `DELETE /v1/files/:id` returns `{id, object:"file", deleted:true}` to match client expectations, and the admin/user portals display the same `status` metadata when browsing tenant files.

### Batches

- `/v1/batches` accepts NDJSON job definitions. The worker writes output/error NDJSON files into the `files` store.
- **Monitoring**: look for `batch worker:` log lines. Errors are surfaced in `/v1/batches/:id` and the admin/user portals.
- **Throughput**: tune `batches.max_concurrency` and the database pool to match your workload.
- The admin portal exposes per-tenant batch tables with output/error download buttons; the user portal defaults to each user’s personal tenant and keeps downloads inline (no more blank pages or extra tabs).
- **API parity**: list responses now support `limit` (1–100) + `after` cursors and return OpenAI-style `has_more`, `first_id`, and `last_id` metadata, plus the new timestamp fields (`cancelling_at`, `expired_at`) and `errors` lists. Metadata payloads are capped at 16 key/value pairs (64/512 characters each) to match the upstream spec.

### Budget Alerts

- Email alerts require `budgets.alert.smtp.host` and `budgets.alert.smtp.from`. Provide credentials if your relay enforces auth; TLS/timeout knobs live under the same block.
- Webhooks receive a JSON payload with tenant, level, spend/limit, and metadata. Tune delivery via `budgets.alert.webhook.timeout` + `max_retries`.
- Every alert (success or failure) is persisted to `budget_alert_events`, so future admin surfaces can show alert history per tenant.

### Single Sign-On (OIDC)

- Configure the OIDC block under `admin.oidc` (issuer, client ID/secret, redirect URL). The redirect URL should point to the backend callback (e.g., `https://gateway.example.com/admin/auth/oidc/callback`). The router exchanges the code, drops a refresh cookie, and then redirects to the requested UI path.
- Both the admin portal and the user portal reuse this flow. The portals pass a `return_to` hint (admin → `/admin/ui/auth/oidc/callback`, user → `/auth/oidc/callback`) so the callback can bounce the browser back to the correct SPA. Make sure those paths are allowed in your IDP (same origin, relative paths only).
- If local auth is still enabled, users can choose between “Continue with SSO” and email/password; flip `admin.local.enabled` (and eventually `user.local.enabled`, once exposed) off to enforce SSO-only logins.
- `roles_claim` chooses which ID token/userinfo claim contains roles or groups. Populate `allowed_roles` to restrict sign-in to specific roles, and `admin_roles` to map one or more roles to Open Gateway “super admin” access. Leave the lists empty to allow everybody / manage super admins manually.

### Usage Comparison API

- `GET /admin/usage/compare` returns a multi-series payload so dashboards can overlay tenants and models without chaining requests.
- Query params:
  - `tenant_ids` – comma-separated UUIDs (must match tenants you can view).
  - `model_aliases` – comma-separated aliases (optional provider-wide comparison).
  - `period` / `timezone` – same semantics as `/admin/usage/summary`.
  - `start` + `end` – optional RFC3339 timestamps for custom date ranges. When provided, both values are required, capped at 180 days, and take precedence over `period`. Useful for billing cycles (e.g., `start=2025-01-01T00:00:00Z&end=2025-01-31T23:59:59Z`).
- Response includes `series[]` objects with `kind` (`tenant`/`model`), display labels, totals, and day-level `points[]` arrays.
- Hard cap: 10 total entities per request. Frontend overlays should limit the number of simultaneous selections to stay under that ceiling.
- User portal calls `/user/usage/compare`, which auto-filters to the caller’s personal + membership tenants; admins can hit both endpoints for debugging scopes.
- **UI behavior**: the Admin Usage tab exposes tenant and model selection dropdowns plus a “Custom range” picker that wires directly to `start`/`end`. Selections are disabled until both dates are applied so you always know the chart is honoring the chosen window.

### Backup / Restore

- **Postgres** is the source of truth (usage, configs, model catalog). Use native tooling (`pg_dump`, `pgbackrest`, etc.).
- **Redis** can be reprovisioned; state is ephemeral.
- **Files**: back up the S3 bucket or `./data/files` directory if you rely on long-lived uploads.

### Security Notes

- Rotate `admin.session.jwt_secret`, provider keys, and bootstrap API keys regularly.
- Restrict access to `/admin/**` via load balancer ACLs if possible.
- Enable OTLP TLS when sending telemetry over the network.

## Troubleshooting

| Symptom | Actions |
| --- | --- |
| `authorization required` errors in user portal downloads | Ensure session cookies are being forwarded; in headless downloads use `userApi` or add `Authorization: Bearer <token>`. |
| Batch worker logs `context_error` | API key or tenant was deleted; inspect `/v1/batches/:id` meta. |
| Stale UI after deploy | Re-run `make build-ui` before compiling routerd. |
| High Redis usage | Check for bursty traffic or increase `rate_limits.*` defaults; Redis only stores counters so memory spikes usually indicate abusive keys. |

## Useful Commands

| Command | Purpose |
| --- | --- |
| `make build-ui` | Rebuilds the embedded admin/user portals. |
| `ROUTER_CONFIG_FILE=/path routerd` | Run with a different config (useful for staging). |
| `curl -H "Authorization: Bearer..." http://host:8090/v1/chat/completions` | Smoke test the public API. |
| `goose status` | Inspect migration state. |

Document any org-specific policies (TLS termination, secret managers, etc.) around this guide to keep operators aligned.

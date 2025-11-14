# Runtime Configuration Reference

`routerd` loads configuration from a YAML file (default `router.yaml`, override with `ROUTER_CONFIG_FILE`) and then overlays any `ROUTER_*` environment variables. All nested keys map to uppercase, underscore-delimited env vars (e.g., `server.listen_addr` → `ROUTER_SERVER_LISTEN_ADDR`). This document covers every supported block.

## Server (`server.*`)

| Key | Description | Default |
| --- | --- | --- |
| `listen_addr` | HTTP listen address. | `:8080` |
| `body_limit_mb` | Max request size. | `20` |
| `sync_timeout` | Non-streaming timeout. | `300s` |
| `stream_idle_timeout` | SSE idle timeout. | `30s` |
| `stream_max_duration` | Hard cap on streaming requests. | `300s` |
| `provider_timeout` | Upstream provider HTTP timeout. | `280s` |
| `read_header_timeout` | HTTP header read deadline. | `5s` |
| `graceful_shutdown_delay` | Wait before force-killing in-flight work during shutdown. | `5s` |

## Database (`database.*`)

Postgres via pgxpool.

| Key | Default |
| --- | --- |
| `url` | **required** |
| `run_migrations` | `true` (uses Goose) |
| `migrations_dir` | `./migrations` |
| `max_conns` / `min_conns` | `20` / `2` |
| `max_conn_idle_time` | `10m` |
| `max_conn_lifetime` | `1h` |

Env shorthands: `ROUTER_DB_URL`, `ROUTER_DATABASE_MAX_CONNS`, etc.

## Redis (`redis.*`)

| Key | Default |
| --- | --- |
| `url` | **required** |
| `db` | `0` |
| `pool_size` | `20` |

Used for rate limiting and idempotency caches.

## Observability (`observability.*`)

| Key | Default |
| --- | --- |
| `enable_metrics` | `true` (exposes `/metrics`) |
| `enable_otlp` | `true` |
| `otlp_endpoint` | `http://localhost:4317` |

Set `ROUTER_OBSERVABILITY_ENABLE_OTLP=false` to disable tracing locally.

## Health Checks (`health.*`)

Controls the background provider health monitor.

| Key | Default |
| --- | --- |
| `check_interval` | `60s` |
| `rolling_window` | `5` samples |
| `cooldown` | `5m` |

## Rate Limits (`rate_limits.*`)

Defaults used when a tenant/key has no custom overrides (persisted in `rate_limit_defaults`).

| Key | Default |
| --- | --- |
| `default_tokens_per_minute` | `1_000_000` |
| `default_requests_per_minute` | `1000` |
| `default_parallel_requests_key` | `10` |
| `default_parallel_requests_tenant` | `100` |

## Budgets (`budgets.*`)

| Key | Default |
| --- | --- |
| `default_usd` | `100` |
| `warning_threshold_perc` | `0.8` |
| `refresh_schedule` | `calendar_month` (`weekly`, `rolling_30d`, etc. also supported) |
| `alert.enabled` | `true` |
| `alert.emails`, `alert.webhooks` | `[]` |
| `alert.cooldown` | `1h` |
| `alert.smtp.host` / `port` / `username` / `password` / `from` / `use_tls` / `skip_tls_verify` / `connect_timeout` | Configure SMTP delivery. Set `host` + `from` (and optionally credentials) to enable email alerts. |
| `alert.webhook.timeout`, `alert.webhook.max_retries` | Control JSON webhook delivery behavior (per-URL timeout + retry count). |

## Reporting (`reporting.timezone`)

Single IANA timezone used for aggregating usage dashboards. Default `UTC`.

## Providers (`providers.*`)

Shared credential fallbacks for adapters:

- `openai_key`, `anthropic_key`, `hugging_face_token`
- `azure_openai_endpoint`, `azure_openai_key`, `azure_openai_version`
- `aws_access_key_id`, `aws_secret_access_key`, `aws_region`
- `gcp_project_id`, `gcp_json_credentials`
- `openai_compatible.base_url` + `api_key`

These values seed provider factories; individual catalog entries can override them via `metadata` or provider-specific sub-blocks.

## Files (`files.*`)

Configures storage for `/v1/files`, batch outputs, etc.

| Key | Description | Default |
| --- | --- | --- |
| `storage` | `local` or `s3`. | `local` |
| `max_size_mb` | Hard upload limit. | `200` |
| `default_ttl` | TTL applied when callers omit `expires_in`. | `168h` |
| `max_ttl` | Ceiling TTL even if caller requests more. | `720h` |
| `encryption_key` | Optional base64 AES key (16/24/32 bytes) for envelope encryption at rest. | _empty_ |
| `local.directory` | Filesystem root when `storage=local`. | `./data/files` |
| `s3.bucket/prefix/region/endpoint/use_path_style` | S3 backend details. | _empty_ |

Expired records are swept periodically; both S3 objects and metadata rows are removed.

## Audio (`audio.*`)

Currently only enforces upload size for transcription/translation endpoints.

| Key | Default |
| --- | --- |
| `max_upload_mb` | `50` |

## Batches (`batches.*`)

Controls `/v1/batches` ingestion + worker TTLs.

| Key | Default |
| --- | --- |
| `max_requests` | `5000` items per batch |
| `max_concurrency` | `50` worker goroutines per batch |
| `default_ttl` | `168h` (window for output/error files) |
| `max_ttl` | `720h` |

## Retention (`retention.*`)

| Key | Default |
| --- | --- |
| `metadata_days` | `30` (minimum days to retain usage metadata) |
| `zero_retention` | `false` (set true to skip writing usage rows entirely) |

## Admin Auth (`admin.*`)

`admin.session.*`, `admin.local.enabled`, and `admin.oidc.*` control dashboard authentication. Key env overrides:

- `ROUTER_ADMIN_SESSION_JWT_SECRET`
- `ROUTER_ADMIN_LOCAL_ENABLED=false`
- `ROUTER_ADMIN_OIDC_ISSUER`, `ROUTER_ADMIN_OIDC_CLIENT_ID`, etc.
- `ROUTER_ADMIN_OIDC_ROLES_CLAIM`, `ROUTER_ADMIN_OIDC_ALLOWED_ROLES`, `ROUTER_ADMIN_OIDC_ADMIN_ROLES`

**OIDC roles**

- `roles_claim`: name of the claim containing roles/groups (e.g., `roles`, `groups`, `custom:roles`). Values are normalized to lowercase strings; arrays, comma-separated strings, or `{role: true}` maps are supported.
- `allowed_roles`: optional whitelist; when provided a user must have at least one of these roles to sign in (applies to both admin and user portals).
- `admin_roles`: optional list of roles that should map to Open Gateway “super admin” privileges. When configured, the user’s `is_super_admin` flag is synced on every OIDC login based on whether they possess any of the listed roles.
- Leave `allowed_roles` empty to permit any authenticated user; leave `admin_roles` empty to manage admin privileges manually.

## Model Catalog (`model_catalog[]`)

Each entry registers a public alias:

| Field | Description |
| --- | --- |
| `alias` | Public name (`gpt-4o`, `gemini-flash`). |
| `provider` | `openai`, `azure`, `bedrock`, `vertex`, `openai_compatible`, etc. |
| `provider_model` | Provider-specific identifier. |
| `context_window` / `max_output_tokens` | Token metadata. |
| `modalities` | e.g., `["text","image"]`. |
| `supports_tools` | Enables tool/function calling. |
| `price_input` / `price_output` / `currency` | Used by the usage logger. |
| `deployment`, `endpoint`, `api_key`, `api_version`, `region` | Optional overrides. |
| `metadata` or provider-specific block | Adapter-specific knobs (Azure deployments, Vertex credentials, Bedrock image options, etc.). |

See `docs/architecture/providers/*.md` for per-provider metadata tables.

### Provider-Specific Metadata Cheatsheet

| Provider | Key | Purpose |
| --- | --- | --- |
| Azure | `deployment`, `endpoint`, `api_version`, `region` | Override defaults per alias when you host multiple Azure deployments. |
| Bedrock | `region`, `aws_access_key_id`, `aws_secret_access_key`, `aws_session_token`, `aws_profile` | Override credentials/region when not inherited from `providers.*`. |
| Bedrock Images | `bedrock_image_task_type`, `bedrock_image_quality`, `bedrock_image_cfg_scale`, `bedrock_image_strength`, `bedrock_image_init_mode`, `bedrock_image_mask_source`, `bedrock_image_variation_prompt` | Tune Titan/Stable Diffusion behavior, including image-to-image strength, default init mode, mask handling, and variation prompts. |
| Vertex | `gcp_project_id`, `vertex_location`, `vertex_publisher`, `vertex_edit_mode`, `vertex_mask_mode`, `vertex_mask_dilation`, `vertex_guidance_scale`, `vertex_base_steps`, `vertex_variation_prompt`, `vertex_person_generation` | Target the right Vertex project/location plus configure Imagen edit/variation defaults (mask behavior, guidance scale, base steps, variation prompt, person policy). |
| Vertex Credentials | `gcp_credentials_json`, `gcp_credentials_format` (`json` or `base64`) | Supply service-account JSON; base64 encoding supported for env vars/metadata. |
| Anthropic | `anthropic_base_url`, `anthropic_version`, `api_key` | Override the Claude API base URL/version or inject a per-alias API key (falls back to `providers.anthropic_key`). |
| Audio aliases | `audio_voice`, `audio_default_voice`, `audio_format` | Provide default TTS voice/format for `/v1/audio/speech` if clients omit them. |
| OpenAI-compatible | `base_url`, `api_key`, `openai_organization` | Required when the alias points at a third-party gateway. |
| Cost overrides | `price_image_cents` | Optional per-alias image pricing override (used by usage logger). |

## Bootstrap (`bootstrap.*`)

Seed data applied on startup:

| Block | Notes |
| --- | --- |
| `tenants[]` | `name`, optional `status`. |
| `admin_users[]` | `email`, `name`, `password`. |
| `api_keys[]` | `tenant`, `name`, optional `scopes`, `rate_limits`, `budget`. |
| `memberships[]` | Link users to tenants (`role`: `owner`, `admin`, `viewer`). |
| `tenant_limits[]` | Overrides for RPM/TPM per tenant. |
| `tenant_budgets[]` | Tenant-specific budgets + alert channels. |

The seeder is idempotent; updates are applied whenever records change in the YAML.

## Environment Variable Cheatsheet

| Key | Example |
| --- | --- |
| `ROUTER_CONFIG_FILE` | `/etc/open-gateway/router.yaml` |
| `ROUTER_DB_URL` | `postgres://user:pass@db/router?sslmode=disable` |
| `ROUTER_REDIS_URL` | `redis://cache:6379/0` |
| `ROUTER_SERVER_LISTEN_ADDR` | `0.0.0.0:8090` |
| `ROUTER_OBSERVABILITY_ENABLE_OTLP` | `false` |
| `ROUTER_PROVIDERS_OPENAI_KEY` | `sk-...` |
| `ROUTER_FILES_STORAGE` | `s3` |
| `ROUTER_BATCHES_MAX_REQUESTS` | `10000` |

Any nested field can be overridden the same way—uppercase the path and join with underscores.

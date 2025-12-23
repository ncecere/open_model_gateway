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

## Telemetry (`telemetry.provider.*`)

Provider telemetry/SLI evaluator settings. Env overrides follow `ROUTER_TELEMETRY_PROVIDER_*` (or the shorter `DEFAULT_PROVIDER_TELEMETRY_*` helpers).

| Key | Default | Notes |
| --- | --- | --- |
| `enabled` | `true` | Toggle the provider telemetry pipeline. |
| `evaluation_interval` | `1m` | How often to roll up SLI windows. |
| `window_size` | `5m` | Sliding window duration for SLI calculations; must be ≥ `evaluation_interval`. |
| `incident_retention_days` | `30` | How long to retain provider incidents; also configurable via admin settings. |
| `downweight_when_degraded` | `true` | When multiple catalog entries back the same alias, automatically down-weight unhealthy instances until they recover. |
| `defaults.latency_p95_ms` | `5000` | Default p95 latency SLI threshold (ms). |
| `defaults.error_rate_threshold` | `0.1` | Default error-rate threshold (0–1). |
| `defaults.timeout_rate_threshold` | `0.05` | Default timeout-rate threshold (0–1). |
| `defaults.min_samples` | `50` | Minimum samples before SLI checks fire. |
| `overrides.{provider}.latency_p95_ms` | | Optional per-provider override (e.g., `overrides.openai.latency_p95_ms: 3000`). |
| `overrides.{provider}.error_rate_threshold` | | Optional per-provider override (e.g., `overrides.azure.error_rate_threshold: 0.03`). |
| `overrides.{provider}.timeout_rate_threshold` | | Optional per-provider override (e.g., `overrides.azure.timeout_rate_threshold: 0.02`). |
| `overrides.{provider}.min_samples` | | Optional per-provider override (e.g., `overrides.openrouter.min_samples: 20`). |

Example:

```yaml
telemetry:
  provider:
    enabled: true
    evaluation_interval: 1m
    window_size: 5m
    incident_retention_days: 30
    downweight_when_degraded: true
    defaults:
      latency_p95_ms: 5000
      error_rate_threshold: 0.1
      timeout_rate_threshold: 0.05
      min_samples: 50
    overrides:
      openai:
        latency_p95_ms: 3000
        error_rate_threshold: 0.05
        timeout_rate_threshold: 0.02
        min_samples: 20
      azure:
        latency_p95_ms: 3500
```

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

## Public (`public.base_url`)

Set the externally reachable hostname for the gateway (for example `https://gateway.example.com`). HTML email templates (budget alerts, invites, etc.) use this to build CTA links so recipients land on the correct UI even when the request that triggered the email originated behind a proxy. Leave blank to fall back to the request `BaseURL`, but background jobs and invite flows work best when this value is configured.

## Reporting (`reporting.timezone`)

Single IANA timezone used for aggregating usage dashboards. Default `UTC`.

## Providers (`providers.*`)

Shared credential fallbacks for adapters:

- `openai_key`, `anthropic_key`, `hugging_face_token`
- `azure_openai_endpoint`, `azure_openai_key`, `azure_openai_version`
- `aws_access_key_id`, `aws_secret_access_key`, `aws_region`
- `gcp_project_id`, `gcp_json_credentials`
- `openai_compatible.base_url` + `api_key`
- `openrouter.base_url`, `api_key`, `referer`, `app_name`
- `groq.base_url`, `api_key`, `region`
- `vllm.base_url`, `mode`, `api_key`, `auth_header`

These values seed provider factories; individual catalog entries can override them via `metadata` or provider-specific sub-blocks.

The OpenRouter block defaults to `https://openrouter.ai/api/v1` with blank attribution headers. Catalog entries continue to be curated manually through config/UI/API—there is no automated discovery, so the config simply provides shared credentials/headers.

The Groq block simply sets default credentials for Groq's OpenAI-compatible endpoint (`https://api.groq.com/openai/v1` by default) and an optional `region` hint forwarded via `X-Groq-Region`. Catalog entries remain the source of truth for Groq models—there is no automatic discovery.

The vLLM block targets self-hosted deployments. Set `providers.vllm.mode` to `openai` when vLLM exposes `/v1/chat/completions` and `/v1/embeddings`, or to `tgi` when pointing at Hugging Face Text Generation Inference (`/generate` + `/generate_stream`). Per-entry overrides live in `metadata.vllm_mode` and `metadata.auth_header`, and you can still override `endpoint`/`api_key` per catalog entry.

## Files (`files.*`)

Configures storage for `/v1/files`, batch outputs, etc.

| Key | Description | Default |
| --- | --- | --- |
| `storage` | `local` or `s3`. | `local` |
| `max_size_mb` | Hard upload limit. | `200` |
| `default_ttl` | TTL applied when callers omit `expires_in`. | `168h` |
| `max_ttl` | Ceiling TTL even if caller requests more. | `720h` |
| `sweep_interval` | How often expired files are reaped. | `15m` |
| `sweep_batch_size` | Number of expired rows to delete per sweep. | `200` |
| `encryption_key` | Optional base64 AES key (16/24/32 bytes) for envelope encryption at rest. | _empty_ |
| `local.directory` | Filesystem root when `storage=local`. | `./data/files` |
| `s3.bucket/prefix/region/endpoint/use_path_style` | S3 backend details. | _empty_ |

Expired records are swept periodically; both S3 objects and metadata rows are removed.

## Audio (`audio.*`)

Configures `/v1/audio/transcriptions` and `/v1/audio/translations`. The gateway now mirrors the OpenAI contract (see `docs/api/audio.md`) supporting `response_format` (`json`, `verbose_json`, `text`, `srt`, `vtt`, `diarized_json`), per-request timestamp granularities, and both `file`/`audio` multipart fields. Requests larger than the configured limit are rejected with `413`.

| Key | Default | Notes |
| --- | --- | --- |
| `max_upload_mb` | `50` | Maximum accepted audio payload size per request. |
| `metadata.audio_formats` | `json,text,srt,vtt,verbose_json,diarized_json` (auto-populated for OpenAI routes) | Comma-separated response formats the handler will allow. Configure via the admin UI’s Audio section when editing a model. |
| `metadata.audio_timestamp_granularities` | `word,segment` | Comma-separated list of timestamps the route supports. Required for clients that request `timestamp_granularities[]`. |
| `metadata.audio_streaming` | `true` for OpenAI, `false` elsewhere | Enables `/v1/audio/transcriptions?stream=true`. Only OpenAI adapters implement streaming today. |

The admin UI now exposes these metadata settings whenever the model type is `audio_transcription`. Each toggle writes the same `audio_*` metadata keys shown above, so you no longer need to edit the raw key/value table for common audio overrides.

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
| `provider` | `openai`, `azure`, `bedrock`, `vertex`, `openai-compatible`, etc. (`openai_compatible` is accepted but normalized) |
| `provider_model` | Provider-specific identifier. |
| `model_type` | Optional workload classification. Supported values include `llm`, `embedding`, `image`, `audio_transcription`, `audio_speech`, `video`, `moderation`, etc. Defaults to `llm` if omitted. Use `audio_transcription` for ASR/translation workloads and `audio_speech` for text-to-speech deployments so capability validation and routing hints stay accurate. |
| `context_window` / `max_output_tokens` | Token metadata. |
| `modalities` | e.g., `["text","image"]`. |
| `supports_tools` | Enables tool/function calling. |
| `price_input` / `price_output` / `currency` | Used by the usage logger (values represent USD per 1M tokens). |
| `pricing_tiers` | Tiered or per-unit overrides; see [pricing tiers](pricing.md) for full guidance (including image buckets). |
| `deployment`, `endpoint`, `api_key`, `api_version`, `region` | Optional overrides. |
| `metadata` or provider-specific block | Adapter-specific knobs (Azure deployments, Vertex credentials, Bedrock image options, etc.). |

`pricing_tiers` supersedes `price_input`/`price_output` when you need multi-step pricing (e.g., per-request floors plus per-token rates) while the legacy fields still work for flat billing. Use the tiered map whenever budgets or invoices depend on multiple meters so the usage logger can emit accurate micro-USD totals. Image tiers can also live under `image_edit`, `image_variation`, or `image_generation` buckets (falling back to `image` when omitted) and may include metadata selectors like `quality`, `resolution`, and `operation`.

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
| Cost overrides | `price_image_cents`, `price_image_edit_cents`, `price_image_variation_cents` | Optional per-alias image pricing overrides (used by the usage logger when providers omit usage numbers). |

> Tip: if a provider bills differently for “standard” vs “HD” quality, you can keep a single alias and add multiple `pricing_tiers.image` rows with `metadata.quality` selectors (or split into separate aliases if you prefer). The gateway will match the request’s `quality` and `size` to the right tier when calculating image spend.

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

## Example Configurations

### Minimal Core Services
```yaml
server:
  listen_addr: ":8090"
  sync_timeout: 300s

database:
  url: ${ROUTER_DB_URL}
  run_migrations: true

redis:
  url: ${ROUTER_REDIS_URL}
  pool_size: 40
```

### Observability + Files
```yaml
observability:
  enable_metrics: true
  enable_otlp: true
  otlp_endpoint: "https://otel.example.com:4317"

files:
  storage: "s3"
  max_size_mb: 200
  default_ttl: 168h
  s3:
    bucket: "omg-files-prod"
    prefix: "uploads/"
    region: "us-east-1"
```

### Provider Credentials
```yaml
providers:
  openai_key: ${OPENAI_KEY}
  azure_openai_endpoint: "https://my-azure.openai.azure.com"
  azure_openai_key: ${AZURE_KEY}
  azure_openai_version: "2024-07-01-preview"
  aws_access_key_id: ${AWS_ACCESS_KEY_ID}
  aws_secret_access_key: ${AWS_SECRET}
  aws_region: "us-west-2"
  gcp_project_id: "my-gcp-project"
  gcp_json_credentials: ${GCP_CREDENTIALS_JSON}
  openai_compatible:
    base_url: "https://partner-gateway.example.com/v1"
    api_key: ${PARTNER_KEY}
  openrouter:
    base_url: "https://openrouter.ai/api/v1"
    api_key: ${OPENROUTER_KEY}
    referer: "https://your-app.example.com"
    app_name: "Open Model Gateway"
```

### Model Catalog Samples
See `docs/admin/model-catalog-examples.md` for per-provider snippets covering LLM, embedding, image, audio, and moderation aliases. Those examples can be pasted into the `model_catalog` array or mirrored in the Admin UI.

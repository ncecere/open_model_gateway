---
title: runtime config
description: Detailed router.yaml options
---
[**Open Model Gateway**](/) loads every runtime control from `router.yaml` before layering `ROUTER_*` env overrides.

---

#### Map top blocks
| Block | Highlights |
| --- | --- |
| `server.*` | `listen_addr`, body limits, sync + streaming timeouts, provider deadline, header read deadline, and graceful shutdown delay keep Fibre and upstream clients healthy. |
| `database.*` | pgxpool URL, Goose migration toggle, pool sizing, and aging knobs (`max_conn_idle_time`, `max_conn_lifetime`) govern Postgres access. |
| `redis.*` | Centralize rate limiting and idempotency caches by pointing to your Redis URL, DB index, and pool size. |
| `observability.*` | Enable `/metrics`, toggle OTLP exporting, and set an OTLP endpoint for the collector. |
| `telemetry.provider.*` | Gate the SLI pipeline, tune evaluation/window durations, toggle automatic down-weighting, and override latency/error thresholds per provider. |
| `health.*` | Configure provider health monitor cadence, rolling samples, and cooldown before reopening a circuit. |
| `public.base_url` | Set the externally reachable hostname so invites, alerts, and background jobs build absolute URLs correctly. |
| `reporting.timezone` | Pick the IANA zone used for daily usage rollups; admin and user dashboards default to this zone unless the caller overrides it. |

---

#### Enforce tenants
| Block | Highlights |
| --- | --- |
| `rate_limits.*` | Defaults for RPM, TPM, and parallel caps used whenever tenants or keys have not defined stricter overrides. |
| `budgets.*` | Global USD ceiling, warning threshold, refresh schedule, plus alert channels (SMTP + webhook) with per-channel cooldowns and delivery retries. |
| `retention.*` | Minimum metadata retention window and a `zero_retention` escape hatch when compliance forbids storing usage rows. |
| `admin.session.*` | JWT secret, cookie, and TTL pairings for admin access tokens and refresh cookies. |
| `admin.local.*` | Boolean guard for username/password logins when you rely only on OIDC. |
| `admin.oidc.*` | Issuer, client credentials, callback, scopes, HTTP timeout, and role claim mappings for enforcing role-based access. |
| `model_catalog[]` | Alias definitions containing provider, deployment overrides, capability metadata, tiered pricing, and modality hints used by the router and UI. |

---

#### Configure providers
| Provider block | Key knobs |
| --- | --- |
| `providers.openai_key` / `anthropic_key` / `hugging_face_token` | Shared API tokens for direct SDK adapters. |
| `providers.azure_openai_*` | Resource endpoint, API key, version, and region defaults for Azure deployments. |
| `providers.aws_*` | Access key, secret, and region for Bedrock plus optional STS session tokens; per-alias metadata can still override. |
| `providers.gcp_*` | Project ID and credentials JSON (raw or base64) for Vertex adapters. |
| `providers.openai_compatible.*` | Base URL and API key for third-party OpenAI-style gateways (e.g., hosted vLLM). |
| `providers.openrouter.*` | Endpoint along with attribution headers (`referer`, `app_name`) forwarded with every request. |
| `providers.groq.*` | Default Groq URL, API key, and target region header for Groq’s OpenAI-compatible endpoints. |
| `providers.vllm.*` | Base URL, adapter mode (`openai` vs `tgi`), API key, and custom auth header for self-hosted deployments. |

---

#### Manage storage
| Block | Highlights |
| --- | --- |
| `files.*` | Select `local` vs `s3` storage, enforce upload size caps, TTLs, sweep cadence, encryption key, and S3 bucket settings for `/v1/files`, output artifacts, and batch results. |
| `audio.*` | Cap upload size, and advertise supported `audio_formats`, `audio_timestamp_granularities`, and streaming controls mirrored in the Admin UI metadata toggles. |
| `batches.*` | Set `max_requests`, worker concurrency, and TTLs for `/v1/batches` outputs to match provider quotas. |

---

#### Seed bootstrap data
| Entry | Purpose |
| --- | --- |
| `bootstrap.tenants[]` | Names and statuses for initial tenants; suspended tenants block routing immediately. |
| `bootstrap.admin_users[]` | Emails, names, and Argon2id-hashed passwords for super admins created during startup. |
| `bootstrap.memberships[]` | Role assignments (`owner`, `admin`, `viewer`, `user`) linking admins to tenants for UI scoping. |
| `bootstrap.api_keys[]` | Generate hashed API keys with optional prefixes, scopes, rate limit overrides, and per-key budgets. |
| `bootstrap.tenant_limits[]` | RPM, TPM, and parallel ceilings applied before per-key overrides. |
| `bootstrap.tenant_budgets[]` | Tenant-only budget overrides, alert channels, and refresh cadence to complement the global defaults. |
| `bootstrap.default_models[]` | Optional curated aliases every personal tenant inherits automatically (set via admin settings or YAML). |

---

#### Sample config
Use this trimmed baseline to confirm your values track the maintained `deploy/router.example.yaml` sample.

```yaml
server:
  listen_addr: ":8080"
  sync_timeout: 300s
  stream_idle_timeout: 30s
  stream_max_duration: 300s
  provider_timeout: 280s

observability:
  enable_metrics: true
  enable_otlp: true
  otlp_endpoint: "http://otel-collector:4317"

telemetry:
  provider:
    enabled: true
    evaluation_interval: 1m
    window_size: 5m
    downweight_when_degraded: true
    defaults:
      latency_p95_ms: 5000
      error_rate_threshold: 0.1
      timeout_rate_threshold: 0.05
      min_samples: 50

providers:
  openai_key: ${OPENAI_KEY}
  azure_openai_endpoint: "https://example.openai.azure.com"
  azure_openai_key: ${AZURE_KEY}
  azure_openai_version: "2024-07-01-preview"
  anthropic_key: ${ANTHROPIC_KEY}
  aws_region: "us-west-2"
  aws_access_key_id: ${AWS_ACCESS_KEY_ID}
  aws_secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  gcp_project_id: "vertex-project"
  gcp_json_credentials: ${VERTEX_CREDS_JSON}
  openrouter:
    base_url: "https://openrouter.ai/api/v1"
    api_key: ${OPENROUTER_KEY}
    referer: "https://portal.example.com"
    app_name: "Open Model Gateway"
  groq:
    base_url: "https://api.groq.com/openai/v1"
    api_key: ${GROQ_KEY}
    region: "us-east-1"
  vllm:
    base_url: "http://localhost:8000/v1"
    mode: "openai"
    auth_header: "Authorization"

files:
  storage: "s3"
  max_size_mb: 200
  default_ttl: 168h
  s3:
    bucket: "omg-files"
    prefix: "uploads/"
    region: "us-east-1"

batches:
  max_requests: 5000
  max_concurrency: 50
  default_ttl: 168h
```

---

#### Research
- Pulled authoritative values from `deploy/router.example.yaml` and the config defaults used by the router.
- Cross-referenced provider metadata in `docs/architecture/providers/*.md` and recent admin settings releases noted in `AGENTS.md`.

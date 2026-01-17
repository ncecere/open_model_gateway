---
title: router sample
description: Annotated YAML baseline for operators
---
[**Open Model Gateway**](/) ships a full router configuration you can copy, trim, and version per environment.

---

#### Review block highlights
| Block | Focus |
| --- | --- |
| `server` | Default HTTP port, sync and streaming timeouts, upstream deadline, and graceful shutdown buffer. |
| `database` / `redis` | Postgres URL plus pool sizing knobs and Redis URL/DB/pool size for rate limits + idempotency. |
| `observability` / `telemetry` / `health` | Enable `/metrics`, OTLP tracing, provider SLI monitors, and interval/cooldown settings for health checks. |
| `rate_limits` / `budgets` | Per-key + tenant RPM/TPM/parallel defaults and tenant budget refresh cadence with alert channels. |
| `providers.*` | Shared credentials for OpenAI, Azure, Anthropic, Bedrock, Vertex, OpenRouter, Groq, vLLM, and OpenAI-compatible adapters. |
| `files` / `audio` / `batches` / `retention` | Files backend, audio upload and metadata caps, batch concurrency, and usage retention windows. |
| `admin.*` | Session JWT secret, cookie names, and OIDC role mapping. |
| `model_catalog[]` | Ready-to-use aliases covering chat, images, audio TTS/STT, OpenRouter, Groq, Bedrock, Vertex, and moderation workloads with tiered pricing. |
| `bootstrap.*` | Seed demo tenant, admin user, API key, tenant limits, and tenant budgets for local installs. |

---

#### Copy annotated YAML
Use this trimmed excerpt as a starting point; the comments mirror new capabilities (audio speech, Groq routing, per-quality image tiers, tenant bootstrap).

```yaml
server:
  listen_addr: ":8080"
  body_limit_mb: 20
  sync_timeout: 300s
  stream_idle_timeout: 30s
  stream_max_duration: 300s
  provider_timeout: 280s
  graceful_shutdown_delay: 5s

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
    overrides:
      openai:
        latency_p95_ms: 3000
        error_rate_threshold: 0.05
        timeout_rate_threshold: 0.02
        min_samples: 20
      azure:
        latency_p95_ms: 3500

rate_limits:
  default_tokens_per_minute: 1000000
  default_requests_per_minute: 1000
  default_parallel_requests_key: 10
  default_parallel_requests_tenant: 100

budgets:
  default_usd: 100
  warning_threshold_perc: 0.8
  refresh_schedule: "calendar_month"
  alert:
    enabled: true
    emails: []
    webhooks: []
    cooldown: 1h
    smtp:
      host: ""
      port: 587
      from: ""
    webhook:
      timeout: 5s
      max_retries: 3

providers:
  openai_key: ${OPENAI_KEY}
  anthropic_key: ${ANTHROPIC_KEY}
  azure_openai_endpoint: "https://your-resource.openai.azure.com"
  azure_openai_key: ${AZURE_KEY}
  azure_openai_version: "2024-07-01-preview"
  aws_access_key_id: ${AWS_ACCESS_KEY_ID}
  aws_secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  aws_region: "us-west-2"
  gcp_project_id: "vertex-project"
  gcp_json_credentials: ${VERTEX_CREDS_JSON}
  openai_compatible:
    base_url: "https://partner-gateway.example.com/v1"
    api_key: ${PARTNER_KEY}
  openrouter:
    base_url: "https://openrouter.ai/api/v1"
    api_key: ${OPENROUTER_KEY}
    referer: "https://your-app.example.com"
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
  storage: "local"
  max_size_mb: 200
  default_ttl: 168h
  max_ttl: 720h
  sweep_interval: 15m
  sweep_batch_size: 200
  local:
    directory: "./data/files"
  s3:
    bucket: ""
    region: ""

audio:
  max_upload_mb: 50

batches:
  max_requests: 5000
  max_concurrency: 50
  default_ttl: 168h

model_catalog:
  - alias: "gpt-image-1"
    provider: "openai"
    provider_model: "gpt-image-1"
    model_type: "image"
    pricing_tiers:
      image:
        - unit: per_image
          price_per_unit: 0.01
          metadata:
            quality: "low"
            resolution: "512x512"
        - unit: per_image
          price_per_unit: 0.04
          metadata:
            quality: "standard"
            resolution: "1024x1024"
        - unit: per_image
          price_per_unit: 0.17
          metadata:
            quality: "hd"
            resolution: "2048x2048"
      image_edit:
        - unit: per_image
          price_per_unit: 0.08
          metadata:
            quality: "standard"
  - alias: "gpt-4o-mini-tts"
    provider: "openai"
    provider_model: "gpt-4o-mini-tts"
    model_type: "audio_speech"
    metadata:
      audio_voice: "alloy"
      audio_format: "mp3"
  - alias: "openrouter-qwen72b"
    provider: "openrouter"
    provider_model: "qwen/qwen2.5-72b-instruct"
    supports_tools: true
    metadata:
      openrouter_referer: "https://your-app.example.com"
      openrouter_app_name: "Open Model Gateway"
  - alias: "groq-llama3-70b"
    provider: "groq"
    provider_model: "llama-3.3-70b-versatile"
    metadata:
      groq_region: "us-east-1"
  - alias: "omni-moderation-latest"
    provider: "openai"
    model_type: "moderation"

bootstrap:
  tenants:
    - name: "demo"
      status: "active"
  admin_users:
    - email: "admin@example.com"
      name: "Demo Admin"
      password: "change-me"
  api_keys:
    - tenant: "demo"
      name: "demo-shared"
      scopes: ["chat", "embeddings", "images", "files", "batches"]
  memberships:
    - tenant: "demo"
      email: "admin@example.com"
      role: "owner"
  tenant_limits:
    - tenant: "demo"
      tokens_per_minute: 2000000
      requests_per_minute: 2000
  tenant_budgets:
    - tenant: "demo"
      budget_usd: 250
      warning_threshold: 0.85
      refresh_schedule: "weekly"
      alert_emails:
        - "billing@example.com"
```

---

#### Extend catalog entries
Add, duplicate, or disable aliases through YAML, the Admin UI, or the Admin API; each flow reloads the router and refreshes catalog caches across admin/user portals.
Keep `pricing_tiers` aligned with upstream contracts so usage exports and per-tenant budgets remain accurate for chat, embeddings, images, audio, files, and batch workloads.

---

#### Research
- Rebased every snippet from `deploy/router.example.yaml` to keep parity with the canonical sample.
- Ensured provider coverage matches current adapters listed in `AGENTS.md` and `docs/architecture/providers/adding.md`.

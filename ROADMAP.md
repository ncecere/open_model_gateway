# Open Model Gateway Roadmap

This roadmap highlights upcoming initiatives that build on the existing routing, budget, and portal foundations. Each section summarizes the goal, suggested implementation steps, and expected benefits.

## Provider Telemetry & Alerting — ✅ Completed

Shipped an end-to-end telemetry pipeline that records provider samples, evaluates SLIs, persists incidents, dispatches alerts, and surfaces health in the admin UI.

- Provider registry now emits structured metrics and writes rolling windows to Redis; evaluator opens/resolves incidents in Postgres with retention/overrides.
- Admin APIs expose SLIs/incidents/alerts (with seed + clear-seed helpers); health monitor records results and can down-weight degraded routes.
- Alerts reuse email/webhook sinks with cooldowns; new provider alert email template added.
- Admin Provider Health page lists SLIs/incidents/alerts with filters, seed/clear seed controls; telemetry docs/config examples updated.

**Benefits**
- Faster incident detection for noisy providers.
- Gives operators objective data to decide when to fail-over or throttle tenants.
- Aligns with enterprise reliability expectations.

## Usage Exports & Billing Hooks

**Goal**: Help finance/RevOps ingest usage data into their billing pipelines without direct DB access.

**Implementation ideas**
- Provide signed CSV/Parquet exports per tenant/time range with token/spend breakdowns.
- Add a scheduled job that posts monthly/weekly spend summaries to tenant-defined webhooks.
- Optionally integrate with S3/GCS for longer retention and offloading.

**Benefits**
- Reduces manual reporting work for admins.
- Unlocks downstream tooling (Chargebee, NetSuite, internal billing) that needs structured inputs.
- Strengthens the value proposition for multi-tenant deployments.

## Fine-Grained Tenant RBAC

**Goal**: Move beyond owner/admin/viewer roles so teams can grant least-privilege access.

**Implementation ideas**
- Extend the `membership_role` enum and service layer to support scoped permissions (e.g., `operator`, `analyst`, `support`).
- Guard sensitive admin endpoints (budget overrides, model catalog edits) behind new permissions.
- Update the admin UI to show role capabilities, invite flows, and quick role changes.
- Expand audit logging to capture role updates for compliance.

**Benefits**
- Safer collaboration across large orgs.
- Matches enterprise expectations for access control reviews.
- Creates a foundation for future custom policies (per-tenant API scopes, etc.).

## Model A/B Testing & Shadow Traffic

**Goal**: Evaluate new provider deployments with real traffic before a full cutover.

**Implementation ideas**
- Add routing metadata that defines experiment buckets (e.g., 90% primary, 10% variant) and a shadow mode that mirrors requests without affecting responses.
- Persist experiment assignment per API key/tenant to keep results consistent.
- Extend the usage service to track request/outcome metrics per experiment.
- Visualize results in the admin Usage or Models page so ops can compare latency/cost/quality deltas.

**Benefits**
- De-risks provider migrations.
- Enables data-driven tuning of weights/cost heuristics.
- Helps tenants justify spend on premium models.

## Self-Service API Key Rotation for Users — ✅ Completed

**Goal**: Empower end users to manage their personal API keys without admin intervention.

**Implementation ideas**
- Expose `/user/api-keys` CRUD endpoints (list/create/rotate/revoke) that reuse the key service but are scoped to the caller’s personal tenant + membership; return last-used timestamp, expiry, and rate-limit overrides for each key.
- Add a rotate action that issues a new secret while preserving key metadata and revocation auditing; enforce configurable TTLs + optional rotation reminders in the auth bootstrap/defaults.
- Update the user portal with a keys tab that shows status/health, last-used, expiry/limits, and a one-click rotate + copy flow (one-time reveal), including inline validation and optimistic UI updates.
- Wire audit logging and budget/rate-limit enforcement to the new endpoints; surface rotation/revocation events in the audit feed for admins and a lightweight “recent activity” widget for users.
- Add contract tests for `/user/api-keys` parity with admin flows, plus UI tests for rotation/revoke to ensure secrets are not re-exposed after creation.

**Benefits**
- Reduces the operational load on admins.
- Encourages better credential hygiene (shorter-lived keys, easy revocation).
- Makes the user portal more compelling for self-service adoption.

## Observability Dashboards

**Goal**: Offer prebuilt visualizations for budgets, provider health, and usage hotspots.

**Implementation ideas**
- Ship Grafana dashboards (JSON) covering budgets, latency, provider errors, and per-model utilization.
- Embed lightweight charts within the admin portal using the existing Usage services for teams that don’t run Grafana.
- Provide Terraform/kubernetes snippets to wire OTEL → Prometheus → Grafana with sane defaults.

**Benefits**
- Shortens time-to-value for new installs.
- Gives SREs a clear view into gateway performance without custom query work.
- Complements the planned telemetry/alerting work.

## Plugin & Tool Execution

**Goal**: Allow tenants to register custom tool endpoints that the router can invoke (similar to OpenAI function calling) for enriched responses.

**Implementation ideas**
- Define a tool schema (name, input JSON schema, invocation URL, auth headers) stored per tenant or model, supporting both manual HTTP tools and MCP-backed tools synced from tenant MCP servers.
- Extend the routing pipeline to detect tool calls in provider responses and execute them securely with timeouts/retries.
- Log tool invocations in usage records for auditing and cost attribution.
- Provide SDK examples showing how to register and consume tools.

**Benefits**
- Unlocks advanced workflows (enterprise RAG, data lookups, transactional actions) without tenants building their own broker.
- Differentiates the gateway versus vanilla OpenAI-compatible proxies.
- Creates upsell opportunities around managed tool catalogs.

## Tenant Guardrails & Policy Engine

**Goal**: Give admins the ability to enforce tenant-specific safety rules so every request and response automatically passes through a guardrail layer.

**Implementation ideas**
- Introduce a guardrail configuration per tenant/API key that references moderation providers, regex/keyword filters, and optional custom webhooks.
- Wrap all `/v1/*` request handling with a guardrail pipeline: pre-request filters (block or redact prompts) and post-response filters (moderation, PII stripping, disclaimer injection).
- Provide an admin UI to manage policies (enable categories, set severities, decide block vs. warn) and show violation metrics.
- Record guardrail actions alongside usage logs for auditing, and surface them via webhooks/alerts.

**Benefits**
- Centralizes trust & safety controls instead of relying on individual tenants to do their own filtering.
- Supports compliance requirements (HIPAA, FINRA, EDU) with tenant-specific policies.
- Creates hooks for future premium offerings (managed guardrail libraries, integration with third-party policy engines).

## Provider Coverage Expansion

**Goal**: Give tenants more model diversity so they can optimize for cost, latency, and regional compliance.

### Google AI Studio
- **Implementation**: Reuse the existing Vertex adapter structure but target the new Google AI Studio REST API (Gemini 2.0, Imagen). Handle OAuth service-account flows plus per-project quotas.
- **Benefits**: Direct access to Google’s latest foundation models without full Vertex setup; helpful for teams experimenting quickly.

### Mistral AI
- **Implementation**: Add REST adapters for chat (Mistral Large / Small) and embeddings, including hosted and custom endpoint support. Map tokenization to their standardized pricing schema.
- **Benefits**: Popular European option with competitive pricing and strong multilingual support.

### Groq
- **Status**: ✅ Completed — Native Groq adapter (chat + SSE) with region-aware metadata is now available in the catalog, config samples, and admin UI.
- **Implementation**: Implement the Groq HTTP API with SSE streaming optimizations to showcase low-latency routing; expose hardware region selection metadata.
- **Benefits**: Ultra-fast inference for assistants; highlights the gateway’s ability to span heterogeneous providers.

### OpenRouter
- **Status**: ✅ Completed — Adapter, config overrides, and catalog examples are live. Operators onboard OpenRouter models via the existing catalog workflow.
- **Benefits**: Offers long-tail models (Qwen, DeepSeek, etc.) without custom adapters for each variant. Future work: reasoning passthrough (see `reasoning_providers.md`).

### Cerebras
- **Implementation**: Integrate the Cerebras Inference API (chat/compute) with support for custom fine-tuned weights. Include health checks for on-prem cluster deployments.
- **Benefits**: Appeals to customers running private LLM infrastructure who still want a unified gateway API.

## OpenAI-Compatible Surface Completeness

**Goal**: Become a drop-in replacement for the full OpenAI REST surface so SDKs work without code changes.

**Implemented today**
- `GET /v1/models`
- `POST /v1/chat/completions` (sync + SSE; tool calling limited to JSON schema outputs)
- `POST /v1/embeddings`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/images/variations`
- `POST /v1/audio/speech`
- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `GET/POST /v1/files` (including content streaming + delete)
- `POST /v1/batches`
- `/v1/moderations` 

**Missing / planned**
- `POST /v1/responses`, Assistants/Threads/Runs APIs (future stretch goal)

### Batches API
- **Implementation**: Mirror `/v1/batches` upload/list/retrieve/delete endpoints, storing batch metadata and job statuses in Postgres. Reuse the existing internal batch runner but add response payload compatibility (file references, errors array).
- **Status**: ✅ Completed — `/v1/batches` now matches the OpenAI API, persists artifacts, and surfaces batches in both admin and user portals.
- **Benefits**: Unlocks high-volume offline workloads and parity with the latest OpenAI SDKs.

### Files API
- **Status**: ✅ Completed — `/v1/files` now implements full OpenAI parity (upload/list/get/delete/content). Schema tracks purpose/status/status_details, cursor pagination mirrors OpenAI’s `limit/after` contract, and both public/admin portals expose status chips, TTL hints, and download actions wired to the new admin/user download endpoints.
- **Benefits**: Required for Assistants, fine-tuning, batches, and future Responses/tooling flows. Brings first-class parity so SDKs can drop-in against the gateway without code forks.

### Moderations
- **Status**: ✅ Completed — `/v1/moderations` now routes through OpenAI, Azure OpenAI, and OpenAI-compatible adapters, enforces budgets/rate limits, records usage, plugs into `/v1/batches`, and ships with sample catalog/default-model entries.
- **Benefits**: Completes the trust & safety story so tenants don’t have to dual-home requests and unlocks the forthcoming guardrail/policy engine work.

### Audio
- **Status**: ✅ Completed — `/v1/audio/transcriptions` and `/v1/audio/translations` now mirror OpenAI’s contract (multipart parsing, response formats, timestamp granularities, SSE streaming) with provider capability validation and budget/rate-limit integration. Admins can curate audio metadata from the new STT/TTS panels in the model editor, and the catalog lists surface “Audio · STT/TTS” chips for quick scanning.
- **Benefits**: Supports speech-to-text use cases and keeps parity with OpenAI SDK helpers. Future work (tracked separately) will focus on inline PCM outputs/responses expansions as upstream providers expose them.

### Images
- **Status**: ✅ Completed — `/v1/images/edits` and `/v1/images/variations` now match OpenAI’s contract alongside generations. The shared handler validates multipart payloads (including `image[]`/`mask[]` aliases), enforces 4 MB per-upload caps, propagates idempotency keys, and records per-operation pricing so budgets stay accurate. Provider adapters advertise edits/variations support explicitly (OpenAI/OpenAI-compatible, Vertex Imagen, Bedrock diffusion), while unsupported routes return structured errors for automatic failover.
- **Benefits**: Enables creative workflows end-to-end, keeps SDKs unmodified, and lets admins set per-operation pricing plus batch workflows for edits/variations.

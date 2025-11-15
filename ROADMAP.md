# Open Model Gateway Roadmap

This roadmap highlights upcoming initiatives that build on the existing routing, budget, and portal foundations. Each section summarizes the goal, suggested implementation steps, and expected benefits.

## Provider Telemetry & Alerting

**Goal**: Detect upstream degradation (latency spikes, error storms, saturation) before it impacts tenants.

**Implementation ideas**
- Extend the provider registry to emit structured metrics per request (latency, tokens, upstream error codes) into OTEL/Prometheus.
- Add configurable Service Level Indicators (SLIs) with thresholds for each adapter; store recent windows in Redis/Postgres.
- Reuse the existing alert channels (email/webhook) to fire incidents when thresholds are breached.
- Surface live provider health charts and incident history in the admin dashboard.

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

## Self-Service API Key Rotation for Users

**Goal**: Empower end users to manage their personal API keys without admin intervention.

**Implementation ideas**
- Expose a `/user/api-keys` CRUD API backed by the existing key service, scoped to the user’s personal tenant.
- Allow per-key rate limit overrides or expiration dates.
- Update the user portal to show key health, last-used timestamps, and one-click rotation.

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
- **Implementation**: Implement the Groq HTTP API with SSE streaming optimizations to showcase low-latency routing; expose hardware region selection metadata.
- **Benefits**: Ultra-fast inference for assistants; highlights the gateway’s ability to span heterogeneous providers.

### OpenRouter
- **Implementation**: Build a generic adapter that accepts tenant-provided OpenRouter API keys and forwards to OpenRouter’s meta-routing layer. Expose the entire model catalog through our admin UI via their discovery endpoint.
- **Benefits**: Offers long-tail models (Qwen, DeepSeek, etc.) without custom adapters for each variant.

### Cerebras
- **Implementation**: Integrate the Cerebras Inference API (chat/compute) with support for custom fine-tuned weights. Include health checks for on-prem cluster deployments.
- **Benefits**: Appeals to customers running private LLM infrastructure who still want a unified gateway API.

## OpenAI-Compatible Surface Completeness

**Goal**: Become a drop-in replacement for the full OpenAI REST surface so SDKs work without code changes.

- **Implemented today**
- `GET /v1/models`
- `POST /v1/chat/completions` (sync + SSE; tool calling limited to JSON schema outputs)
- `POST /v1/embeddings`
- `POST /v1/images/generations`
- `POST /v1/audio/speech`
- `POST /v1/batches` (create/list/retrieve/cancel, output/error downloads, and portal surfaces)

- **Missing / planned**
- `POST /v1/audio/transcriptions`, `POST /v1/audio/translations`
- `POST /v1/images/edits`, `POST /v1/images/variations`
- `POST /v1/moderations`
- `GET/POST /v1/files` (complete CRUD + content streaming)
- `POST /v1/responses`, Assistants/Threads/Runs APIs (future stretch goal)

### Batches API
- **Implementation**: Mirror `/v1/batches` upload/list/retrieve/delete endpoints, storing batch metadata and job statuses in Postgres. Reuse the existing internal batch runner but add response payload compatibility (file references, errors array).
- **Status**: ✅ Completed — `/v1/batches` now matches the OpenAI API, persists artifacts, and surfaces batches in both admin and user portals.
- **Benefits**: Unlocks high-volume offline workloads and parity with the latest OpenAI SDKs.

### Files API
- **Implementation**: Extend `/v1/files` to accept uploads, list files with pagination, retrieve content, and delete. Store file blobs in the current storage backend while maintaining OpenAI’s metadata schema (purpose, bytes, status). Support streaming downloads for large files.
- **Benefits**: Required for Assistants, fine-tuning, and soon the Responses API when tool outputs reference assets.

### Moderations
- **Implementation**: Add `/v1/moderations` that routes to a configured provider (OpenAI, Azure, or third-party). Persist moderation logs for auditing and optionally allow tenant-specific policies.
- **Benefits**: Completes the trust & safety story so tenants don’t have to dual-home requests.

### Audio
- **Implementation**: Beyond existing text-to-speech, add `/v1/audio/transcriptions` and `/v1/audio/translations` with Whisper-compatible request bodies, file uploads, and SSE partials. Plan for the new Responses audio outputs (inline PCM) once providers expose them.
- **Benefits**: Supports speech-to-text use cases and keeps parity with OpenAI SDK helpers.

### Images
- **Implementation**: Generations already work across Azure OpenAI, OpenAI native, Bedrock Titan/SD, and Vertex Imagen; expand to `/v1/images/edits` and `/v1/images/variations` with multipart inputs (image + mask files) and output formats (base64 URLs). Align response JSON with OpenAI’s schema.
- **Benefits**: Enables creative workflows and removes the last gaps in the image surface.

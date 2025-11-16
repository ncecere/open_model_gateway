# Groq Provider Integration Plan

## Research
- Audit Groq OpenAI-compatible endpoints (chat, streaming) and note unsupported surfaces (no embeddings) plus region/latency metadata we should surface. `/models` only needs to work for lightweight health probes.
- Capture authentication + rate-limit requirements (Bearer secrets, hardware region hints, retry/backoff guidance) to drive adapter defaults and documentation.
- Decide on the health-check contract (cheap `/models` probe vs. zero-token chat call) so the provider monitor keeps accurate status.

## Backend Adapter & Config
- Extend `ProviderOverrides` with a Groq config block (API key, base URL, region, retry profile) and add shared defaults under `providers.groq` in `config.go`.
- Implement `internal/adapters/groq` wrapping Groq’s OpenAI-style chat API: sync + SSE streaming plus normalized usage/error payloads. `/models` should be used strictly for health checks (no discovery/caching).
- Register the Groq builder in `internal/providers` to wire credentials/metadata precedence, assign chat/chat_stream capabilities, and expose health checks.
- Add metadata helpers/constants for Groq (region, accelerator hints) so routing + UI surfaces stay consistent.

## Model Catalog & Routing
- Add Groq model examples (e.g., `llama3-70b-8192`, `mixtral-8x7b`) with pricing/context data to `docs/admin/model-catalog-examples.md` and seed local entries in `deploy/router.local.yaml`.
- Update runtime config/docs/default-model settings to mention `provider: "groq"` and document recommended metadata keys (`groq_region`, etc.).
- Verify `/admin/providers` and catalog UIs display Groq-specific metadata; extend APIs/UI if additional labels are needed (region, latency).

## Validation & Observability
- Add builder unit tests covering credential precedence, metadata propagation, and capability enforcement.
- Write adapter tests (httptest fakes) for sync/streaming chat, health checks/error translation, and cancellation.
- Update provider fixtures with Groq payloads so CI catches regressions in token accounting or finish reasons.
- Perform manual end-to-end validation (local config) to confirm budgets/rate limits/usage logging capture Groq pricing inputs accurately.

## Documentation & Rollout
- Create `docs/architecture/providers/groq.md` capturing API quirks, retries, onboarding steps, and metadata guidance.
- Update runtime config docs, admin guide, and roadmap/provider sections to reflect Groq availability and any limitations.
- Add CHANGELOG + task tracking entries, and coordinate staging rollout (enable Groq models, validate observability alerts) before exposing to tenants.

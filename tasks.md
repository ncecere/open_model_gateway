# OpenRouter Provider Integration

## Detailed Plan
1. **Research + Requirements Lock**
   - Confirm OpenRouter REST surface (base URL, `/chat/completions`, SSE behavior, embeddings/images availability, retry/backoff policy).
   - Document mandatory headers (`Authorization`, `HTTP-Referer`, `X-Title`) and per-tenant API key expectations so config/schema work is scoped.
   - Inventory the discovery endpoint (`GET /models`) payload and map pricing/context/tool flags to our catalog metadata.
   - Capture any capability gaps (e.g., no moderations) in `agents.md` for downstream awareness.
2. **Configuration + Catalog Prep**
   - Extend `config.Providers` with an `openrouter` block plus env overrides.
   - Add `ProviderOverrides.OpenRouter` and metadata keys for API key + policy headers; validate catalog entries enforce OpenRouter requirements.
   - Ship doc/sample YAML entries showing how tenants register their OpenRouter key and pricing.
3. **Adapter Implementation**
   - Create `backend/internal/adapters/openrouter` with an `Options` struct, OTEL-aware HTTP client, and capability implementations (`Chat`, `ChatStream`, optionally embeddings/images/models/health).
   - Use `streamutil.Forward` for SSE and normalize usage (prompt/completion tokens) from OpenRouter responses.
   - Add fixtures + unit tests covering sync + stream conversions and health checks.
4. **Provider Builder + Registry Integration**
   - Add `builder_openrouter.go`, resolve API key/header precedence (entry → override → metadata → config), set capabilities, register definition.
   - Ensure builder populates `Route.Metadata` with base URL/header data and wires health endpoints.
5. **Model Discovery + Admin UX**
   - Backend: add an admin service/endpoint that caches OpenRouter’s model catalog (with pricing/context) for UI consumption.
   - Frontend: enhance the admin model catalog editor to list OpenRouter models and prefill catalog forms with selected entries.
6. **Usage, Budgeting, and Routing Glue**
   - Normalize token accounting + cost mapping so budgets/rate limits tag `provider=openrouter`.
   - Decide on tenant-scoped credential storage (bootstrap + admin UI) and ensure secrets follow existing encryption patterns.
7. **Testing & Validation**
   - Adapter + builder unit coverage, factory integration tests, and (optional) mocked end-to-end smoke hitting a fake OpenRouter server.
   - Verify observability metrics/labels and health monitor integration.
8. **Docs + Coordination**
   - Add `docs/architecture/providers/openrouter.md`, update runtime config + README, and log the work in `agents.md` + `CHANGELOG.md`.

## Task List
- [x] Research OpenRouter API surface, headers, retry semantics, and discovery payload.
- [x] Update config structs/envs and catalog metadata to support OpenRouter-specific credentials/headers.
- [x] Implement `backend/internal/adapters/openrouter` with chat + streaming (and any supported extra capabilities) plus tests/fixtures.
- [x] Add OpenRouter provider builder/definition, ensure factory + health monitor integration.
- [x] Build cached model discovery service + admin endpoint and wire frontend catalog UI to import models.
- [x] Align usage/budget accounting and tenant credential flows with OpenRouter responses.
- [x] Expand docs (`docs/runtime/config.md`, `docs/architecture/providers/openrouter.md`, `backend/README.md`) and update `agents.md`/`CHANGELOG.md`.

## Follow-ups
- [ ] Implement reasoning passthrough across providers per `reasoning_providers.md` (schema changes, adapter updates, SSE support).

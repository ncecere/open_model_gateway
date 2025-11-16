# OpenRouter Adapter Notes

Source material: OpenRouter public docs (overview, streaming, limits, errors, embeddings, attribution) fetched via `https://r.jina.ai/https://openrouter.ai/docs/...` plus `GET https://openrouter.ai/api/v1/models`.

## API Surface & Authentication
- Base URL: `https://openrouter.ai/api/v1`.
- Auth: `Authorization: Bearer <OPENROUTER_API_KEY>` (per-tenant Bring-Your-Own-Key). Keys can optionally enforce credit limits; `GET /api/v1/key` returns quota + credit metadata for telemetry.
- Primary endpoints mirror OpenAI: `POST /chat/completions` (messages/prompt schema), `POST /embeddings` (multi-input supported), `POST /responses` (alpha) and provider-specific extras like transforms/provider routing knobs on the request body.
- Discovery: unauthenticated `GET /models` returns the public catalog (id, canonical slug, pricing prompt/completion/request/image/web_search, context windows, supported parameters). Response includes provider metadata we can map into our catalog.

## Headers & Attribution
- Optional but encouraged headers: `HTTP-Referer` (app URL for attribution) and `X-Title` (display name). Docs highlight they unlock analytics/rankings; some providers require non-local referers for rate-limit reasons.
- Plan: allow catalog metadata to override referer/title per route while config provides safe defaults (e.g., ops-owned domain + app label). Enforce validation so we never emit blank referers for production deployments.

## Request Parameters & Capabilities
- Chat schema largely matches OpenAI with extras: `transforms`, `models[]`, `route: "fallback"`, `provider` preference block (order, allow_fallbacks, data collection, etc.), `prediction` hints, `structured_outputs`.
- Tool calling is passed through when provider implements OpenAI semantics; otherwise OpenRouter converts `tools` to YAML templates. We should pass the user payload directly and rely on OpenRouter to handle mapping.
- Embeddings endpoint accepts batch arrays and exposes provider routing controls. Streaming is **not** available for embeddings.
- Streaming (chat): set `stream: true` to receive SSE. Streams occasionally include SSE comments (":" lines) to keep idle connections active; clients should ignore them. `streamOptions.includeUsage=true` can attach final usage stats.
- Streaming cancellation: aborting HTTP request cancels upstream compute when supported.

## Error & Retry Semantics
- Standard error payload: `{"error": {"code": <int>, "message": "...", "metadata": {...}}}` with HTTP status matching `code` (400 invalid params, 401 invalid key, 402 insufficient credits, 403 moderation flag, 408 timeout, 429 rate limit, 502 provider, 503 no provider meets routing requirements).
- Streaming mid-flight errors are sent as SSE chunks with `object: "chat.completion.chunk"`, `choices.finish_reason="error"`, and a top-level `error`. HTTP status remains 200 post-headers.
- Rate limits: global + per-model quotas; free SKUs limited to 20 RPM plus daily caps (50/day <10 credits, 1000/day ≥10 credits). Cloudflare DDoS guard triggers if bursts exceed policy. Recommended approach: exponential backoff with jitter on 429 and inspect `/api/v1/key` for remaining credits.

## Usage Accounting
- `/models` pricing fields include prompt/completion, request, image, web_search, input cache read/write, reasoning tokens, etc. We'll need to map each to our internal usage schema (prompt/completion micro-USD) and expand metadata to capture extras (reasoning charges, search add-ons).
- Streaming chunks can include `usage` when `streamOptions.includeUsage=true`. Non-stream responses follow OpenAI-like `usage: {prompt_tokens, completion_tokens, total_tokens}`.

## Integration Considerations
- Support BYOK: tenants configure their own OpenRouter key; we simply forward it plus the required headers. Our config should also allow ops-owned fallback keys for testing.
- Discovery caching: store `/models` response (plus timestamp) so admin UI can display long-tail catalog entries. Response is large; plan to limit stored fields → id/name/friendly_name/provider/pricing/context/capabilities.
- Health checks: `/models` is publicly accessible but we should prefer an authenticated call (ensures key validity). Non-200 should mark route unhealthy.
- Retries: respect upstream guidance—OpenRouter multiplexes to providers, so repeated retries on 429/503 should include jitter and optionally adjust provider preferences (e.g., allow fallback). Errors after stream start already include finish_reason `error`; we must propagate to clients intact.

These notes feed the adapter + builder implementation plus docs updates tracked in `tasks.md`.

## Configuration Block
- Base keys live under `providers.openrouter`: `api_key`, `base_url`, `referer`, `app_name`.
- `models_cache_ttl` (default `10m`) controls how long `/models` discovery responses stay warm before admins trigger another fetch from OpenRouter.
- Catalog entries can provide per-model overrides via `provider_overrides.openrouter` or `metadata` (`openrouter_api_key`, `openrouter_referer`, `openrouter_app_name`) once the builder is in place.
- Admins can fetch the cached catalog via `GET /admin/providers/openrouter/catalog`; the response includes the refresh timestamp, expiry, and the parsed pricing/context metadata used to seed model entries.

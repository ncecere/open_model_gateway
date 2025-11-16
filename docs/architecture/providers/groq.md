# Groq Adapter Notes

Source material: [OpenAI compatibility guide](https://console.groq.com/docs/openai), [API reference](https://console.groq.com/docs/api-reference#chat), and [latency guide](https://console.groq.com/docs/production-readiness/optimizing-latency).

## Capabilities

- **Surface**: OpenAI-compatible `/openai/v1/chat/completions` with SSE streaming. `/openai/v1/models` is only used for health checks; we do **not** auto-discover catalog entries.
- **Supported model types**: Chat-only today. Embeddings, images, and moderation routes are not available on Groq, so the adapter only advertises `chat` and `chat_stream`.
- **Usage payload**: Matches OpenAI's schema with extra Groq metrics (`queue_time`, `prompt_time`, etc.). We keep the token counters for spend accounting and drop the rest.

## Request details

- Auth is `Authorization: Bearer $GROQ_API_KEY`.
- Groq recommends using `max_completion_tokens`. We also send the deprecated `max_tokens` to preserve compatibility with older SDKs.
- `messages[].name`, `logprobs`, `logit_bias`, and `n != 1` are rejected (per the compatibility doc), so the adapter strips names before sending.
- Streaming uses data-only SSE frames terminated with `data: [DONE]`. Set `stream_options.include_usage=true` so the last frame carries `usage`.
- Region pinning is optional. When `providers.groq.region` or `metadata.groq_region` is set we forward it via `X-Groq-Region` so operators can compare latency per PoP (see the latency guide).

## Metadata keys

| Key | Description |
| --- | --- |
| `groq_region` | Preferred hardware region/PoP for the catalog entry. Shown in the UI and forwarded as `X-Groq-Region`. |
| `groq_api_key` | Per-model override that wins over the catalog entry and global default. Useful for BYOK tenants. |
| `base_url` | Override for dedicated Groq endpoints if you are on a private deployment or proxy. Defaults to `https://api.groq.com/openai/v1`. |

All other metadata behaves like any OpenAI-compatible adapter. If you need to track accelerator type or service tiers, add custom metadata via the admin UI—the gateway will preserve those key/value pairs even though the adapter does not consume them yet.

## Health

- Health checks hit `/models` with the configured key. Groq returns an `invalid_api_key` body when credentials are wrong, which the adapter surfaces directly.

We intentionally skip automatic discovery/caching for Groq; operators add catalog entries manually just like Azure or OpenAI deployments.

## Limitations / TODO

- No embeddings/images/audio for Groq yet (not exposed in their OpenAI compatibility surface). We can extend once those endpoints exist.
- Tool calling is passed through untouched—Groq enforces the `n == 1` constraint, so we rely on upstream validation instead of duplicating checks here.
- Responses API is still on the backlog; for now, tenants must stick to chat completions on the Groq adapter.

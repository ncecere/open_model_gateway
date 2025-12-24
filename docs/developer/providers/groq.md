# Groq Provider

The `groq` adapter targets Groq's OpenAI-compatible `/openai/v1/chat/completions` API while preserving gateway-wide limiter, budget, and SSE semantics.

## Know the basics

| Field | Value |
| --- | --- |
| Provider key | `groq` |
| Coverage | Chat + SSE streaming (no embeddings/images/moderations yet) |
| Auth | `Authorization: Bearer <GROQ_API_KEY>` |
| Region header | Optional `X-Groq-Region` to pin pop locations |

## Configure metadata
Define these keys per catalog entry.

| Key | Description |
| --- | --- |
| `groq_api_key` | Per-model API key override; falls back to global secret. |
| `groq_region` | Preferred region/PoP string surfaced in `/admin/providers` and forwarded to Groq. |
| `base_url` | Overrides `https://api.groq.com/openai/v1` for private deployments or proxies. |

Always set `modalities: [text]`, `supports_tools: true` when using tool calls (Groq enforces `n == 1`), and keep `model_type: llm`.

## Deploy sample entry

```yaml
model_catalog:
  - alias: groq-llama3-70b
    provider: groq
    provider_model: llama-3.3-70b-versatile
    model_type: llm
    modalities: [text]
    supports_tools: true
    price_input: 0.00059
    price_output: 0.00079
    currency: USD
    metadata:
      groq_region: us-east-1
```

## Verify behavior
Strip `messages[].name`, `logprobs`, `logit_bias`, and `n != 1` before dispatch (the adapter enforces this), enable `stream_options.include_usage=true` so the final SSE chunk carries usage, and monitor `/admin/providers` to catch invalid key responses (`invalid_api_key`) emitted during health checks.

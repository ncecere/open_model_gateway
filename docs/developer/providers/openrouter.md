# OpenRouter Provider

The `openrouter` adapter connects to OpenRouter's marketplace while preserving OpenAI-compatible `/v1/*` semantics, attribution headers, and retry logic.

## Know the basics

| Field | Value |
| --- | --- |
| Provider key | `openrouter` |
| Coverage | Chat + SSE streaming, embeddings |
| Base URL | Defaults to `https://openrouter.ai/api/v1` |
| Auth | `Authorization: Bearer <OPENROUTER_KEY>` plus attribution headers |

## Configure globals

| Key | Description |
| --- | --- |
| `providers.openrouter.base_url` | Override when targeting a dedicated shard or proxy. |
| `providers.openrouter.api_key` | Fallback API key for entries without overrides. |
| `providers.openrouter.referer` | Default `HTTP-Referer` header for attribution/rate-limit allowlists. |
| `providers.openrouter.app_name` | Default `X-Title` value shown in OpenRouter analytics. |

## Set catalog metadata

| Key | Description |
| --- | --- |
| `api_key` / `openrouter_api_key` | Per-entry credential override (metadata or `provider_overrides.openrouter`). |
| `base_url` | Entry-specific base URL for tenant shards. |
| `openrouter_referer` | Overrides the Referer header. |
| `openrouter_app_name` | Overrides the `X-Title` header. |

Remember to flag `modalities` (usually `[text]` or `[text, embedding]`) so routing stays accurate.

## Deploy sample entries

```yaml
model_catalog:
  - alias: openrouter-qwen-72b
    provider: openrouter
    provider_model: qwen/qwen2.5-72b-instruct
    model_type: llm
    modalities: [text]
    supports_tools: true
    price_input: 0.0004
    price_output: 0.0008
    metadata:
      openrouter_referer: https://chat.example.com
      openrouter_app_name: ChatOps Dashboard
  - alias: openrouter-llama3-tenant
    provider: openrouter
    provider_model: meta-llama/Meta-Llama-3-70B-Instruct
    model_type: llm
    modalities: [text]
    api_key: ${TENANT_OPENROUTER_KEY}
    metadata:
      base_url: https://openrouter.ai/api/v1
      openrouter_referer: https://tenant.example.com
      openrouter_app_name: Tenant Portal
```

## Verify behavior
Include attribution headers to avoid throttling, rely on the adapter’s automatic retry/backoff for 429/5xx responses, monitor `/admin/providers` for incident data sourced from `/api/v1/key`, and note that streaming errors arrive via the final SSE chunk with `finish_reason: "error"`.

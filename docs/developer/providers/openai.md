# OpenAI & Compatible Providers

Use these adapters to reach either api.openai.com or any OpenAI-style gateway without duplicating routing logic.

## Know the options

| Provider key | Description |
| --- | --- |
| `openai` | Calls api.openai.com using the official SDK, supporting chat + SSE, embeddings, images, audio, and model listing for health. |
| `openai-compatible` | Targets any OpenAI REST-compatible endpoint; you supply the base URL and credentials per entry. |

## Configure globals

| Key | Description |
| --- | --- |
| `providers.openai_key` | Default API key used for both adapters when metadata omits `api_key`. |
| `providers.openai_org` (optional) | Organization header forwarded to native OpenAI endpoints. |

## Set catalog metadata

| Key | Applies to | Description |
| --- | --- | --- |
| `base_url` | both | Overrides the API base (include `/v1`). Defaults to `https://api.openai.com/v1` for `openai`. |
| `api_key` | `openai-compatible` | Per-entry API key override for BYOK tenants or alternate deployments. |
| `openai_organization` | both | Overrides the `OpenAI-Organization` header. |

Declare `modalities` (`text`, `embedding`, `image`, `audio`) and capability flags just like other providers.

## Deploy sample entries

```yaml
model_catalog:
  - alias: gpt-4o
    provider: openai
    provider_model: gpt-4o
    model_type: llm
    modalities: [text]
    supports_tools: true
  - alias: private-qwen
    provider: openai-compatible
    provider_model: Qwen/Qwen3-Coder-30B-A3B-Instruct
    model_type: llm
    modalities: [text]
    endpoint: https://gateway.internal/api/v1
    metadata:
      api_key: sk-router-123
      openai_organization: tenant-lab
```

## Verify behavior
The native adapter uses the official `openai-go` client to guarantee parity, `/v1/models` health checks rely on OpenAI’s list endpoint, and compatible entries assume the upstream accepts `Authorization: Bearer <key>` and the standard JSON schema.

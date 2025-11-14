# OpenAI & OpenAI-Compatible Providers

The platform now ships two first-class adapters for services that expose the OpenAI API surface:

| Provider name | Description |
|---------------|-------------|
| `openai` | Calls api.openai.com using your OpenAI key/organization. Supports chat + streaming, embeddings, images, and model listing/health checks. |
| `openai-compatible` | Targets any endpoint that implements the OpenAI REST contract (e.g., self-hosted gateways). You provide the base URL + API key per catalog entry. |

## Global Configuration (`providers.*`)

| Key | Description |
|-----|-------------|
| `providers.openai_key` | Default API key for both native and compatible adapters (can still be overridden per model). |

## Catalog Metadata Keys

Use `model_catalog[].metadata` to customize connections:

| Key | Applies to | Description |
|-----|------------|-------------|
| `base_url` | `openai`, `openai-compatible` | Overrides the API base URL (e.g., `https://api.openai.com/v1` or `https://router.example.com/v1`). If omitted for `openai`, the official base is used. |
| `api_key` | `openai-compatible` | Per-model API key override (falls back to `providers.openai_key`). |
| `openai_organization` | both | Optional `OpenAI-Organization` header value. |

`model_catalog[].endpoint` (or the structured `openai`, `openai_compatible` override blocks) can also be used as a shorthand for `base_url` if you prefer keeping metadata minimal. Always include the `/v1` suffix—the adapter appends resource paths like `/chat/completions` relative to whatever you provide.

### Example: Native OpenAI

```yaml
- alias: gpt-4o
  provider: openai
  provider_model: gpt-4o
  modalities: [text]
```

### Example: Compatible Gateway

```yaml
- alias: my-private-model
  provider: openai-compatible
  provider_model: gpt-4o
  endpoint: https://gateway.internal/api/v1
  metadata:
    api_key: sk-router-123
```

### Example: Self-hosted Qwen deployment

```yaml
- alias: qwen3-coder-30b
  provider: openai-compatible
  provider_model: Qwen/Qwen3-Coder-30B-A3B-Instruct
  modalities: [text]
  endpoint: http://137.220.56.131:8000/v1
  openai_compatible:
    base_url: http://137.220.56.131:8000/v1
    api_key: public-demo
```

## Features

- Chat + SSE streaming (`/v1/chat/completions`).
- Embeddings (`/v1/embeddings`).
- Images (`/v1/images/generations`).
- `openai` provider can list models and is used for health checks.

## Notes

- The adapter reuses the official `openai-go` client so request/response parity matches the native SDK.
- Compatible endpoints must honor the same authentication header (`Authorization: Bearer <key>`) and JSON schema.

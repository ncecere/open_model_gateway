# Azure OpenAI Provider

Use the `azure` provider when routing OpenAI-compatible requests through Azure OpenAI deployments for chat, streaming, embeddings, and images.

## Know the basics
Keep these facts in mind before configuring entries.

| Field | Value |
| --- | --- |
| Provider key | `azure` |
| Coverage | Chat, SSE streaming, embeddings, images |
| Health | Circuit breaker pings each deployment using the configured key; failures mark the catalog entry unhealthy |
| Routing | Requests become `https://{resource}.openai.azure.com/openai/deployments/{deployment}/{operation}?api-version=...` |

## Configure global defaults
Set top-level config under `providers.*` for shared credentials.

| Key | Description |
| --- | --- |
| `providers.azure_openai_endpoint` | Base endpoint such as `https://contoso.openai.azure.com`. |
| `providers.azure_openai_key` | Default API key used when catalog entries omit `metadata.api_key`. |
| `providers.azure_openai_version` | API version applied to every request unless overridden (e.g., `2024-08-01-preview`). |

## Set catalog metadata
Define these keys within each `model_catalog[].metadata` block or via `provider_overrides.azure`.

| Key | Description |
| --- | --- |
| `deployment` | Required Azure deployment name matching the Azure portal. |
| `endpoint` | Overrides the global endpoint for this entry. |
| `api_key` | Per-model API key override for BYOK scenarios. |
| `api_version` | Per-entry API version when preview models differ from the default. |
| `region` | Free-form label surfaced in `/admin/providers` and `/v1/models`. |
| `subscription` | Optional subscription identifier for auditing. |
| `failover_group` | Logical grouping so the router can weight redundant deployments. |
| `price_image_cents` | Image-route cost override forwarded to the usage ledger. |

Always declare `modalities` (`text`, `embedding`, `image`) so the router only exposes supported routes.

## Deploy sample entries

```yaml
model_catalog:
  - alias: gpt-4o-mini-azure
    provider: azure
    provider_model: gpt-4o-mini
    model_type: llm
    modalities: [text, embedding, image]
    supports_tools: true
    price_input: 0.005
    price_output: 0.015
    metadata:
      deployment: gpt-4o-mini-eastus
      endpoint: https://contoso.openai.azure.com
      api_version: 2024-08-01-preview
      region: eastus
      failover_group: primary
  - alias: text-embedding-3-large-azure
    provider: azure
    provider_model: text-embedding-3-large
    model_type: embedding
    modalities: [embedding]
    price_input: 0.0002
    metadata:
      deployment: embed3-large-prod
      api_key: ${AZURE_EMBED_KEY}
      subscription: prod-subscription-id
      region: westus
```

## Verify behavior
Ensure `/v1/models` lists Azure-backed aliases (Azure itself does not implement the endpoint), confirm SSE streaming emits OpenAI-formatted chunks, and keep `providers.azure_openai_version` aligned with your deployments so Azure does not reject requests.

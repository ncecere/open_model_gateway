# Azure OpenAI Provider

The Azure adapter exposes chat completions (sync + SSE), embeddings, and image generation. Model listing is not available from the Azure REST API, so `/v1/models` falls back to static catalog data.

## Required Fields

Each `model_catalog[]` entry must define:

| Field | Purpose |
|-------|---------|
| `alias` | Public name clients request (e.g., `gpt-5-mini`). |
| `provider` | Must be `azure`. |
| `provider_model` | Azure deployment model (e.g., `gpt-4o-mini`). |
| `deployment` | Azure deployment name (required). |
| `modalities` | Include `text` for chat, `embedding` for embeddings, `image` for DALL·E style generation. |

## Metadata Keys

| Key | Description |
|-----|-------------|
| `endpoint` | Overrides `providers.azure_openai_endpoint` for this entry. |
| `api_key` | Overrides `providers.azure_openai_key`. |
| `api_version` | Defaults to `providers.azure_openai_version`. |
| `region` | Informational label used by routing and health checks. |
| `subscription` | Optional subscription identifier for ops dashboards. |
| `failover_group` | Arbitrary grouping key so the router can spread requests. |
| `price_image_cents` | Per-image override in cents; feeds the usage ledger. |

## Example

```yaml
model_catalog:
  - alias: gpt-5-mini
    provider: azure
    provider_model: gpt-4o-mini
    deployment: gpt-4o-mini-us
    modalities: ["text", "embedding", "image"]
    metadata:
      endpoint: https://my-openai-resource.openai.azure.com/
      region: eastus
      failover_group: primary
```

See `GET /admin/providers` (admin portal) or `docs/architecture/providers/adding.md` for the latest registry summary and onboarding checklist.

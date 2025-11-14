# Vertex AI Provider

Use the `vertex` provider to route chat/streaming requests to Gemini models on Google Vertex AI or to text embedding models such as `text-embedding-005`. The adapter speaks the Vertex REST API directly and requires a service-account credential with access to the target project/location.

## Supported Modalities

| Capability | Notes |
|------------|-------|
| Chat / Streaming | Backed by `:generateContent` and `:streamGenerateContent` endpoints. System prompts are mapped to `systemInstruction`, user/assistant messages are converted into Vertex `contents`. |
| Embeddings | Uses `:predict` for text embedding models (e.g., `text-embedding-005`). |

## Required Configuration

Global defaults live under `providers.*` in `router.yaml` or environment variables:

| Config Key | Description |
|------------|-------------|
| `providers.gcp_project_id` | Default GCP project used when a catalog entry does not provide its own `gcp_project_id`. |
| `providers.gcp_json_credentials` | Raw (or base64-encoded) service-account JSON with access to Vertex AI. |

### Catalog Metadata / Overrides

You can override per-model settings through `model_catalog[].metadata` or the structured `provider_overrides.vertex` block (the Admin UI writes the latter). Both feed the same builder; using the overrides block simply keeps credentials scoped to the provider instead of mixing them into `metadata`.

| Key | Description |
|-----|-------------|
| `gcp_project_id` | Overrides the project for this catalog entry. Falls back to `providers.gcp_project_id`. |
| `gcp_credentials_json` | Overrides the credential JSON for this entry. Prefer pasting the raw service-account JSON via a YAML block scalar; the builder automatically detects base64 if you still need it. Falls back to `providers.gcp_json_credentials`. |
| `gcp_credentials_format` | Optional hint (`json` or `base64`). Only required when supplying base64 manually. |
| `vertex_location` | Region/location for the Vertex endpoint (defaults to the catalog `region` or `us-central1`). |
| `vertex_publisher` | Defaults to `google`. Override when using custom publishers. |
| `vertex_endpoint` | Optional full model endpoint (without `:generateContent`). Useful for private endpoints or proxies. |

### Example Catalog Snippet

```yaml
model_catalog:
  - alias: gemini-1.5-pro
    provider: vertex
    provider_model: gemini-1.5-pro
    region: us-central1
    modalities: [text]
    vertex:
      gcp_project_id: ict-genai
      vertex_location: us-east1
      gcp_credentials_json: |-
        {
          "type": "service_account",
          ...
        }
```

For embeddings:

```yaml
  - alias: text-embedding-005
    provider: vertex
    provider_model: text-embedding-005
    modalities: [embeddings]
```

## Authentication

Provide a service-account JSON (either directly or base64-encoded) with the `cloud-platform` scope allowed. The adapter exchanges it for OAuth2 access tokens and signs every request automatically.

## Health Checks

`routerd` issues a lightweight `GET` against the resolved model endpoint during health checks. Ensure the service account has permission to describe the model resource.

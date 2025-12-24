# Vertex AI Provider

Use the `vertex` adapter to reach Gemini chat/streaming models and Vertex embedding models with service-account credentials.

## Know the basics

| Field | Value |
| --- | --- |
| Provider key | `vertex` |
| Coverage | Chat, SSE streaming, embeddings |
| Auth | Exchanges service-account JSON for OAuth tokens per request |
| Endpoint | `projects/{project}/locations/{location}/publishers/{publisher}/models/{provider_model}` |

## Configure globals

| Key | Description |
| --- | --- |
| `providers.gcp_project_id` | Default project when metadata omits one. |
| `providers.gcp_json_credentials` | Base64 or raw JSON used for token minting. |

## Set catalog metadata
Use `metadata` or `provider_overrides.vertex`.

| Key | Description |
| --- | --- |
| `gcp_project_id` | Overrides the project for this entry. |
| `gcp_credentials_json` | Per-entry credential JSON to enforce least privilege. |
| `gcp_credentials_format` | Hints whether the JSON is raw or base64. |
| `vertex_location` | Region (e.g., `us-central1`, `europe-west3`). |
| `vertex_publisher` | Defaults to `google`; override for custom publishers. |
| `vertex_endpoint` | Full override when calling private endpoints or proxies. |

Set `modalities` (`text`, `embedding`) and keep `provider_model` aligned with Gemini or `text-embedding-005` identifiers.

## Deploy sample entries

```yaml
model_catalog:
  - alias: gemini-1.5-pro
    provider: vertex
    provider_model: gemini-1.5-pro
    model_type: llm
    modalities: [text]
    metadata:
      gcp_project_id: ai-platform-prod
      vertex_location: us-east1
      gcp_credentials_json: |-
        {
          "type": "service_account",
          "project_id": "ai-platform-prod",
          "private_key": "-----BEGIN PRIVATE KEY-----...",
          "client_email": "vertex-router@ai-platform-prod.iam.gserviceaccount.com"
        }
  - alias: text-embedding-005
    provider: vertex
    provider_model: text-embedding-005
    model_type: embedding
    modalities: [embedding]
    metadata:
      vertex_location: europe-west1
      gcp_project_id: analytics-shared
```

## Verify behavior
Ensure the service account has `aiplatform.endpoints.predict` (or `cloud-platform`) permissions, confirm `/admin/providers` shows healthy status per entry, and remember that Vertex responses lack billing data—set accurate pricing in the catalog for usage logging.

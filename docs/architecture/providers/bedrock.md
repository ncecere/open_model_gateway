# AWS Bedrock Provider

The Bedrock adapter covers three capability families:

- **Chat (sync + SSE)** via Anthropic Claude when `bedrock_chat_format=anthropic_messages`.
- **Embeddings** via Titan (`bedrock_embedding_format=titan_text`).
- **Images** via Titan image generation when `bedrock_image_task_type` is supplied.

## Required Fields

| Field | Purpose |
|-------|---------|
| `alias` | Public name (`claude-3-sonnet`). |
| `provider` | Must be `bedrock`. |
| `provider_model` | Bedrock model ID, e.g., `anthropic.claude-3-sonnet-20240229-v1:0`. |
| `region` | Optional override; defaults to `providers.aws_region`. |
| `modalities` | Include the capabilities you intend to expose (`text`, `embedding`, `image`). |

## Metadata Keys

| Key | Description |
|-----|-------------|
| `bedrock_chat_format` | `anthropic_messages` enables Claude chat + streaming. |
| `anthropic_version` | Defaults to `bedrock-2023-05-31`. |
| `bedrock_default_max_tokens` | Fallback `max_tokens` for chat requests. |
| `bedrock_embedding_format` | `titan_text` for Titan embeddings. |
| `bedrock_embed_dims` | Integer dimension override. |
| `bedrock_embed_normalize` | Boolean string enabling Titan normalization. |
| `bedrock_image_task_type` | e.g., `TEXT_IMAGE` to unlock Titan image support. |
| `bedrock_image_cfg_scale` | Float (string) controlling CFG scale. |
| `bedrock_image_quality` | `standard`/`premium`. |
| `bedrock_image_style` | Titan style hint. |
| `bedrock_image_seed` | Deterministic seed override. |
| `aws_access_key_id` / `aws_secret_access_key` / `aws_session_token` | Per-entry AWS credentials; fall back to `providers.*`. |
| `aws_profile` | Optional shared config profile. |

## Example Entry

```yaml
model_catalog:
  - alias: claude-3-sonnet
    provider: bedrock
    provider_model: anthropic.claude-3-sonnet-20240229-v1:0
    region: us-west-2
    modalities: ["text"]
    metadata:
      bedrock_chat_format: anthropic_messages
      bedrock_default_max_tokens: "4096"
```

Check `GET /admin/providers` for the latest capability summary or `docs/architecture/providers/adding.md` for onboarding guidance.

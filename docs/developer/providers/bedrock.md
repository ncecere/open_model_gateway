# AWS Bedrock Provider

The `bedrock` provider routes chat, streaming, embeddings, and Titan image generation through Amazon Bedrock using metadata-driven adapters.

## Know the basics

| Field | Value |
| --- | --- |
| Provider key | `bedrock` |
| Coverage | Claude chat (sync + SSE), Titan embeddings, Titan image generation |
| Auth | Uses AWS credentials from global config or per-entry metadata |
| Routing | Calls `bedrock-runtime.{region}.amazonaws.com/model/{provider_model}` with capability-specific payloads |

## Configure global defaults

| Key | Description |
| --- | --- |
| `providers.aws_region` | Default AWS region when metadata omits `region`. |
| `providers.aws_access_key_id` / `providers.aws_secret_access_key` / `providers.aws_session_token` | Shared credentials for Bedrock requests. |
| `providers.aws_profile` | Shared AWS profile when relying on local config/STS. |

## Set catalog metadata
Use `model_catalog[].metadata` to fine-tune each deployment.

| Key | Description |
| --- | --- |
| `bedrock_chat_format` | `anthropic_messages` unlocks Claude chat + streaming. |
| `anthropic_version` | Overrides the Bedrock Claude API version (`bedrock-2023-05-31` default). |
| `bedrock_default_max_tokens` | Fallback `max_tokens` when callers omit it. |
| `bedrock_embedding_format` | Usually `titan_text` for Titan embeddings. |
| `bedrock_embed_dims` / `bedrock_embed_normalize` | Control embedding dimension and normalization. |
| `bedrock_image_task_type` | e.g., `TEXT_IMAGE` to enable Titan image routes. |
| `bedrock_image_cfg_scale` / `bedrock_image_quality` / `bedrock_image_style` / `bedrock_image_seed` | Titan image tuning knobs. |
| `region` | Overrides the AWS region for this entry. |
| `aws_access_key_id` / `aws_secret_access_key` / `aws_session_token` / `aws_profile` | Per-entry credentials for BYOK tenants or cross-account routing. |

Declare `modalities` per entry (`text`, `embedding`, `image`) so unsupported paths stay disabled.

## Deploy sample entry

```yaml
model_catalog:
  - alias: claude-3-sonnet
    provider: bedrock
    provider_model: anthropic.claude-3-sonnet-20240229-v1:0
    model_type: llm
    region: us-west-2
    modalities: [text]
    supports_tools: true
    price_input: 0.006
    price_output: 0.015
    currency: USD
    metadata:
      bedrock_chat_format: anthropic_messages
      bedrock_default_max_tokens: "4096"
  - alias: titan-embed
    provider: bedrock
    provider_model: amazon.titan-embed-text-v2:0
    model_type: embedding
    modalities: [embedding]
    price_input: 0.0001
    metadata:
      bedrock_embedding_format: titan_text
      bedrock_embed_dims: "1536"
```

## Verify behavior
Check `/admin/providers` to confirm the adapter reports chat, stream, embedding, or image capabilities, ensure AWS credentials resolve (bad keys surface `invalid_signature` errors), and keep metadata aligned so the evaluator can mark unhealthy deployments before routing traffic.

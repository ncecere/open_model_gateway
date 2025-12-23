# Model Catalog: Adding Models

The model catalog defines which models are available to tenants and how those
models map to provider deployments. Every request routed by the gateway uses
the catalog to determine provider, pricing, and capabilities.

## Add a Model in the Admin UI

1. Open **Admin -> Models**.
2. Click **Add model**.
3. Fill in the core fields:
   - **Alias**: the name clients will request (example: `gpt-4o-mini`).
   - **Provider**: OpenAI, Azure, Bedrock, Vertex, OpenAI-compatible, etc.
   - **Provider model**: upstream model identifier.
   - **Model type**: LLM, embedding, image, moderation, audio, batch.
   - **Deployment**: provider deployment or routing label.
4. Pricing and constraints:
   - **Price input/output** and **currency**.
   - **Context window** and **max output tokens** (LLMs).
   - **Modalities** (text, image, audio, embedding).
   - **Supports tools** (tool calling, functions).
5. Provider-specific metadata (examples below).
6. Save. The model becomes available for routing immediately.

## Validate the Model

1. Confirm it appears in `/v1/models`.
2. Send a test request to the model alias.
3. Check **Admin -> Usage** for recorded spend.

## Required Fields (Minimum)

- `alias`
- `provider`
- `provider_model`
- `model_type`
- `deployment`
- `price_input`
- `price_output`
- `currency`

## Common Optional Fields

- `context_window`, `max_output_tokens`
- `modalities` (text, image, audio, embedding)
- `supports_tools`
- provider metadata (API versions, regions, image settings, etc.)

## Provider Metadata Examples

Use the provider-specific examples in `docs/admin/model-catalog-examples.md`.
These examples include:

- OpenAI and OpenAI-compatible base URL settings.
- Azure API versions and deployment names.
- Bedrock image task types.
- Vertex region and service account configuration.

## Example Minimal Entry (YAML)

```yaml
- alias: "gpt-4o-mini"
  provider: "openai"
  provider_model: "gpt-4o-mini"
  model_type: "llm"
  deployment: "gpt-4o-mini"
  context_window: 128000
  max_output_tokens: 16384
  modalities: ["text"]
  supports_tools: true
  price_input: 0.0005
  price_output: 0.0015
  currency: "USD"
```

If you add the same values in the UI form, the resulting catalog entry is identical.

## Model Types and Matching Endpoints

- **LLM**: `/v1/chat/completions`
- **Embeddings**: `/v1/embeddings`
- **Images**: `/v1/images/generations` (and edits/variations if supported)
- **Moderation**: `/v1/moderations`
- **Audio**: `/v1/audio/speech`, `/v1/audio/transcriptions`

If the model type does not match the endpoint, the gateway will reject the call.

## Troubleshooting

- **Model not appearing**: check alias uniqueness and provider config.
- **Image endpoint errors**: confirm `model_type=image` and metadata.
- **Tool calls failing**: enable `supports_tools`.
- **Unexpected spend**: verify pricing and currency values.

# Model Catalog Example Library

Use this reference when adding models through the Admin UI (“Add model”) or by editing the `model_catalog` block inside `router.yaml`. Every example below shows the minimum fields required plus the metadata/override keys that enable provider-specific capabilities.

> **Tip:** Populate secrets (`api_key`, `openrouter_api_key`, etc.) via environment variables in production. Examples use inline strings for readability only.

## How to Apply These Examples
1. Pick the provider + model type you need.
2. Copy the YAML snippet into `model_catalog` **or** mirror the fields inside the Admin UI dialog.
3. Adjust `alias`, `deployment`, pricing, and any metadata to match your account.
4. Reload the router (or save in the UI) to activate the alias.

---

## Multimodal Capabilities & Overrides
Each catalog entry already lists its `modalities` (e.g. `"text"`, `"image"`, `"audio"`). The router now publishes those capabilities back through `/v1/models` under the `capabilities` field (boolean flags for `image_input`, `audio_input`, and `video_input`) and enforces them at runtime:

- If a chat request contains an image/audio/video block but the selected alias does not support that modality, the gateway immediately returns `400` instead of forwarding an invalid payload to the provider.
- When multiple routes back an alias, only the ones that can satisfy the requested modalities are tried.

You can explicitly override the inferred capabilities by adding metadata flags to any catalog entry:

| Metadata Key        | Description                                      |
| ------------------- | ------------------------------------------------ |
| `cap_image_input`   | Set to `"true"` or `"false"` to force-enable/disable image inputs.
| `cap_audio_input`   | Same for audio inputs (speech-to-text, etc.).    |
| `cap_video_input`   | Reserve for providers that accept video frames.  |

Use these overrides when a provider adapter supports additional modalities (e.g. OpenAI vision) or when you want to block multimodal requests for a specific deployment even though the upstream model technically allows them.

---

## OpenAI (LLM)
```yaml
- alias: "gpt-4o-mini"
  provider: "openai"
  provider_model: "gpt-4o-mini"
  model_type: "llm"
  context_window: 128000
  max_output_tokens: 16384
  modalities: ["text"]
  supports_tools: true
  price_input: 0.0005
  price_output: 0.0015
  currency: "USD"
  deployment: "gpt-4o-mini"
  metadata:
    openai_organization: "org_123"
```

### OpenAI Embeddings
```yaml
- alias: "text-embedding-3-small"
  provider: "openai"
  provider_model: "text-embedding-3-small"
  model_type: "embedding"
  context_window: 8192
  modalities: ["embedding"]
  supports_tools: false
  price_input: 0.00002
  price_output: 0
  currency: "USD"
  deployment: "text-embedding-3-small"
```

### OpenAI Images
```yaml
- alias: "gpt-image-1"
  provider: "openai"
  provider_model: "gpt-image-1"
  model_type: "image"
  modalities: ["image"]
  supports_tools: false
  price_input: 0.04
  price_output: 0.08
  currency: "USD"
  deployment: "gpt-image-1"
  pricing_tiers:
    image:
      - unit: per_image
        price_per_unit: 0.01
        metadata:
          quality: "low"
          resolution: "512x512"
      - unit: per_image
        price_per_unit: 0.04
        metadata:
          quality: "standard"
          resolution: "1024x1024"
      - unit: per_image
        price_per_unit: 0.17
        metadata:
          quality: "hd"
          resolution: "2048x2048"
    image_edit:
      - unit: per_image
        price_per_unit: 0.08
        metadata:
          quality: "standard"
    image_variation:
      - unit: per_image
        price_per_unit: 0.12
        metadata:
          quality: "standard"
```

### OpenAI Audio (Speech-to-Text)
```yaml
- alias: "gpt-4o-mini-transcribe"
  provider: "openai"
  provider_model: "gpt-4o-mini-transcribe"
  model_type: "audio_transcription"
  modalities: ["audio"]
  supports_tools: false
  price_input: 0
  price_output: 0.006
  currency: "USD"
  deployment: "gpt-4o-mini-transcribe"
  metadata:
    audio_formats: "json,verbose_json,srt,vtt"
    audio_streaming: "true"
```

### OpenAI Audio (Text-to-Speech)
```yaml
- alias: "gpt-4o-mini-tts"
  provider: "openai"
  provider_model: "gpt-4o-mini-tts"
  model_type: "audio_speech"
  modalities: ["audio"]
  supports_tools: false
  price_input: 0
  price_output: 0.015
  currency: "USD"
  deployment: "gpt-4o-mini-tts"
  metadata:
    audio_voice: "alloy"
    audio_format: "mp3"
```

### OpenAI Moderation
```yaml
- alias: "omni-moderation-latest"
  provider: "openai"
  provider_model: "omni-moderation-latest"
  model_type: "moderation"
  modalities: ["text"]
  supports_tools: false
  price_input: 0.0001
  price_output: 0
  currency: "USD"
  deployment: "omni-moderation-latest"
```

---

## Azure OpenAI (LLM)
```yaml
- alias: "gpt-4o-azure"
  provider: "azure"
  provider_model: "gpt-4o"
  model_type: "llm"
  context_window: 128000
  max_output_tokens: 4096
  modalities: ["text"]
  supports_tools: true
  price_input: 0.005
  price_output: 0.015
  currency: "USD"
  deployment: "gpt-4o-eastus"
  endpoint: "https://my-azure.openai.azure.com"
  api_version: "2024-07-01-preview"
  metadata:
    azure_resource_group: "rg-openai-prod"
```

---

## OpenAI-Compatible Gateway
```yaml
- alias: "partner-sonnet"
  provider: "openai-compatible"
  provider_model: "claude-3-sonnet"
  model_type: "llm"
  context_window: 200000
  max_output_tokens: 4096
  modalities: ["text"]
  supports_tools: true
  price_input: 0.003
  price_output: 0.015
  currency: "USD"
  deployment: "claude-3-sonnet"
  endpoint: "https://partner-gateway.example.com/v1"
  api_key: "${PARTNER_GATEWAY_KEY}"
  metadata:
    base_url: "https://partner-gateway.example.com/v1"
    description: "Proxy to partner cluster"
```

### OpenAI-Compatible Images (Per-Megapixel)
If `quality` is omitted, the gateway assumes `standard` when selecting image tiers.

```yaml
- alias: "partner-flux-mp"
  provider: "openai-compatible"
  provider_model: "flux-1-schnell"
  model_type: "image"
  modalities: ["image"]
  supports_tools: false
  currency: "USD"
  pricing_tiers:
    image:
      - unit: per_megapixel
        price_per_unit: 0.012
        metadata:
          quality: "standard"
  deployment: "flux-1-schnell"
  endpoint: "https://partner-gateway.example.com/v1"
  api_key: "${PARTNER_GATEWAY_KEY}"
```

---

## Amazon Bedrock
### Claude 3 Sonnet (LLM)
```yaml
- alias: "claude-3-sonnet-bedrock"
  provider: "bedrock"
  provider_model: "anthropic.claude-3-sonnet-20240229-v1:0"
  model_type: "llm"
  context_window: 200000
  max_output_tokens: 4096
  modalities: ["text"]
  supports_tools: true
  price_input: 0.006
  price_output: 0.015
  currency: "USD"
  region: "us-west-2"
  metadata:
    bedrock_chat_format: "anthropic_messages"
```

### Titan Image Generator
```yaml
- alias: "titan-image"
  provider: "bedrock"
  provider_model: "amazon.titan-image-generator-v1"
  model_type: "image"
  modalities: ["image"]
  supports_tools: false
  price_input: 0
  price_output: 0.02
  currency: "USD"
  region: "us-west-2"
  metadata:
    bedrock_image_task_type: "TEXT_IMAGE"
    bedrock_image_quality: "standard"
```

---

## Vertex AI (Imagen)
```yaml
- alias: "vertex-imagen-2"
  provider: "vertex"
  provider_model: "imagen-2"
  model_type: "image"
  context_window: 0
  max_output_tokens: 0
  modalities: ["image"]
  supports_tools: false
  price_input: 0
  price_output: 0.06
  currency: "USD"
  deployment: "projects/my-gcp-project/locations/us-central1"
  metadata:
    vertex_location: "us-central1"
    vertex_publisher: "google"
    gcp_project_id: "my-gcp-project"
```

---

## Anthropic (Native API)
```yaml
- alias: "claude-3-opus"
  provider: "anthropic"
  provider_model: "claude-3-opus-20240229"
  model_type: "llm"
  context_window: 200000
  max_output_tokens: 4096
  modalities: ["text"]
  supports_tools: true
  price_input: 0.015
  price_output: 0.075
  currency: "USD"
  deployment: "claude-3-opus"
  metadata:
    anthropic_version: "2023-06-01"
```

---

## vLLM / TGI
```yaml
- alias: "llama-3-vllm"
  provider: "vllm"
  provider_model: "meta-llama/Meta-Llama-3-8B-Instruct"
  model_type: "llm"
  context_window: 8192
  max_output_tokens: 2048
  modalities: ["text"]
  supports_tools: true
  price_input: 0.15
  price_output: 0.45
  currency: "USD"
  deployment: "llama-3-vllm"
  endpoint: "http://localhost:8000/v1"
  metadata:
    vllm_mode: "openai"

- alias: "llama-3-tgi"
  provider: "vllm"
  provider_model: "meta-llama/Meta-Llama-3-8B-Instruct"
  model_type: "llm"
  context_window: 8192
  max_output_tokens: 2048
  modalities: ["text"]
  supports_tools: false
  price_input: 0.15
  price_output: 0.45
  currency: "USD"
  deployment: "llama-3-tgi"
  endpoint: "http://localhost:8080"
  metadata:
    vllm_mode: "tgi"
    auth_header: "Authorization"
```

---

## OpenRouter
```yaml
- alias: "openrouter-qwen72b"
  provider: "openrouter"
  provider_model: "qwen/qwen2.5-72b-instruct"
  model_type: "llm"
  context_window: 32768
  max_output_tokens: 4096
  modalities: ["text"]
  supports_tools: true
  price_input: 0.0004
  price_output: 0.0008
  currency: "USD"
  deployment: "qwen2.5-72b-instruct"
  metadata:
    openrouter_referer: "https://your-app.example.com"
    openrouter_app_name: "Open Model Gateway"
```

---

## Groq
```yaml
- alias: "groq-llama3-70b"
  provider: "groq"
  provider_model: "llama-3.3-70b-versatile"
  model_type: "llm"
  context_window: 131072
  max_output_tokens: 32768
  modalities: ["text"]
  supports_tools: true
  price_input: 0.59
  price_output: 0.79
  currency: "USD"
  deployment: "llama-3.3-70b-versatile"
  metadata:
    groq_region: "us-east-1"
```

---

## Moderation + Audio Summary
| Model Type | Suggested Provider | Example Alias |
| --- | --- | --- |
| LLM | Azure, OpenAI, Bedrock, OpenRouter | `gpt-4o-azure`, `claude-3-sonnet-bedrock` |
| Embedding | OpenAI (`text-embedding-3-small`) | `text-embedding-3-small` |
| Image | OpenAI, Bedrock Titan/SD, Vertex Imagen | `gpt-image-1`, `titan-image`, `vertex-imagen-2` |
| Audio Transcription | OpenAI (`gpt-4o-mini-transcribe`) | `gpt-4o-mini-transcribe` |
| Audio Speech | OpenAI (`gpt-4o-mini-tts`) | `gpt-4o-mini-tts` |
| Moderation | OpenAI (`omni-moderation-latest`) | `omni-moderation-latest` |

Keep this sheet handy when onboarding new models. Update it whenever providers add new capabilities so admins can follow a proven template.

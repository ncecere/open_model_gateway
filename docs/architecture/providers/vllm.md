# vLLM / TGI Provider

The `vllm` provider lets the gateway route chat requests to:

- vLLM's OpenAI-compatible API (`/v1/chat/completions`, `/v1/embeddings`)
- Hugging Face Text Generation Inference (`/generate`, `/generate_stream`)

Choose the mode via `providers.vllm.mode` or per-entry `metadata.vllm_mode`.

## Config

```yaml
providers:
  vllm:
    base_url: "http://localhost:8000/v1"
    mode: "openai"            # openai or tgi
    api_key: ""
    auth_header: "Authorization"
```

## Catalog examples

### OpenAI-compatible vLLM

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
  endpoint: "http://localhost:8000/v1"
  metadata:
    vllm_mode: "openai"
```

### TGI (/generate)

```yaml
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
  endpoint: "http://localhost:8080"
  metadata:
    vllm_mode: "tgi"
    auth_header: "Authorization"
```

## Notes

- `vllm` mode defaults to `openai`.
- TGI mode supports chat and streaming only; embeddings/images are not exposed.
- When `auth_header` is `Authorization` and `api_key` has no prefix, the gateway sends `Bearer <api_key>`.

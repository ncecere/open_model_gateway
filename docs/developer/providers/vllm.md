# vLLM / TGI Provider

The `vllm` adapter targets either vLLM's OpenAI-compatible API or Hugging Face Text Generation Inference (TGI) endpoints so operators can self-host models behind the same `/v1/*` routes.

## Know the basics

| Field | Value |
| --- | --- |
| Provider key | `vllm` |
| Modes | `openai` for `/v1/chat/completions`, `tgi` for `/generate` + `/generate_stream` |
| Coverage | Chat + SSE streaming (embeddings only when running the OpenAI-compatible API) |
| Auth | Optional API key sent via `Authorization` (default) or custom header |

## Configure globals

| Key | Description |
| --- | --- |
| `providers.vllm.base_url` | Default endpoint (include `/v1` for OpenAI mode). |
| `providers.vllm.mode` | `openai` or `tgi`; entries can override via metadata. |
| `providers.vllm.api_key` | Shared API key forwarded to the upstream. |
| `providers.vllm.auth_header` | Header name for the API key (`Authorization` default). |

## Set catalog metadata

| Key | Description |
| --- | --- |
| `vllm_mode` | Overrides the global mode per entry. |
| `auth_header` | Overrides the header used for credentials. |
| `endpoint` / `base_url` | Entry-specific URL when hosts differ. |

Declare `modalities: [text]`, set `supports_tools` according to your upstream, and configure context/max token metadata as desired.

## Deploy sample entries

```yaml
model_catalog:
  - alias: llama-3-vllm
    provider: vllm
    provider_model: meta-llama/Meta-Llama-3-8B-Instruct
    model_type: llm
    context_window: 8192
    max_output_tokens: 2048
    modalities: [text]
    supports_tools: true
    price_input: 0.15
    price_output: 0.45
    endpoint: http://localhost:8000/v1
    metadata:
      vllm_mode: openai
  - alias: llama-3-tgi
    provider: vllm
    provider_model: meta-llama/Meta-Llama-3-8B-Instruct
    model_type: llm
    context_window: 8192
    max_output_tokens: 2048
    modalities: [text]
    supports_tools: false
    price_input: 0.15
    price_output: 0.45
    endpoint: http://localhost:8080
    metadata:
      vllm_mode: tgi
      auth_header: Authorization
```

## Verify behavior
Keep `providers.vllm.mode` aligned with the upstream API, ensure embeddings routes are only exposed when the OpenAI-compatible API supports them, and confirm `/admin/providers` shows the correct capabilities before enabling tenant traffic.

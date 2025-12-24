# Request reference
Use these endpoint guides to exercise `/v1/*` APIs with curl.

## Prepare environment
Set the shared variables before running any example.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-demo.my-secret"
```

## Pick an endpoint
Jump into the file that matches your workflow.

| Doc | Focus |
| --- | --- |
| [models.md](models.md) | List available models and provider health. |
| [chat.md](chat.md) | Call chat completions or the Responses API, including SSE. |
| [embeddings.md](embeddings.md) | Generate single or batched embedding vectors. |
| [moderations.md](moderations.md) | Submit text or JSON payloads for safety reviews. |
| [images.md](images.md) | Run generations, edits, or variations with file references. |
| [audio.md](audio.md) | Handle transcriptions, translations, and speech synthesis. |
| [files.md](files.md) | Upload, list, download, and delete supporting files. |
| [batches.md](batches.md) | Orchestrate NDJSON batch jobs and inspect outputs. |

## Track headers
Monitor every response to validate budgets and throttling.

| Header | Description |
| --- | --- |
| `X-Budget-Limit`, `X-Budget-Remaining`, `X-Budget-Reset` | Budget policy applied to the caller; values are USD cents and ISO timestamps. |
| `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` | Per-key RPM/TPM/parallel counters for client throttling. |
| `X-Budget-Warning` | Present when usage crosses the configured warning threshold. |
| `X-Provider-Request-ID` | Upstream request identifier for paging providers. |
| `X-Request-ID` | Router request ID for correlating logs and traces. |

## Helpful extras
Copy workflows from `Code_Examples/curl/*.sh` or substitute your tenant-specific model aliases.

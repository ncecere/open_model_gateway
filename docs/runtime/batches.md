# Batch API Reference Snapshot

This document summarizes the upstream OpenAI/Azure OpenAI Batch API behavior so backend/frontend agents can implement compatible semantics. Source citations point to the current authoritative docs:

- Azure OpenAI Batch how-to and reference payloads<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
- OpenAI Python SDK-generated schema (`openai-python` repo)<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup><sup>,</sup><sup>[3](https://github.com/openai/openai-python/blob/main/src/openai/types/batch.py)</sup>

## Supported Endpoints & Request Constraints

| Requirement | Notes |
| --- | --- |
| Allowed endpoints | `/v1/responses`, `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, `/v1/moderations` today.<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup> |
| Completion window | Only `"24h"` is currently valid; future API versions may add longer windows.<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup> |
| Input file | JSONL, uploaded with `purpose=batch`, max 200 MB and up to 100k requests depending on provider quota.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup><sup>,</sup><sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup> Each line must contain `{ "custom_id": "...", "method": "POST", "url": "/v1/...", "body": { ... } }`. |
| Metadata limits | Up to 16 key/value pairs; keys ≤64 chars, values ≤512 chars.<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup> |
| Output expiry | Clients can optionally request a TTL by passing `output_expires_after` (seconds).<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup> |

## Lifecycle & Status Mapping

| Upstream status | Meaning | Required timestamps |
| --- | --- | --- |
| `validating` | File accepted, schema/quotas being validated. | `created_at` set, others null. |
| `in_progress` | Worker is executing requests. | `in_progress_at`. |
| `finalizing` | Work finished, output/error files being flushed. | `finalizing_at`. |
| `completed` | All requests processed successfully (errors file may still exist but can be empty). | `completed_at`, `expires_at`. |
| `failed` | Batch aborted due to validation/runtime error. | `failed_at`, `errors` list populated. |
| `cancelling` | User requested cancellation while work still running; expect transition to `cancelled`. | `cancelling_at`. |
| `cancelled` | Outstanding work stopped; partial results available. | `cancelled_at`. |
| `expired` | Output file TTL elapsed or completion window exceeded. | `expired_at`. |

Ensure we persist every timestamp column described in the Azure Batch table (`created_at`, `in_progress_at`, `finalizing_at`, `completed_at`, `failed_at`, `cancelling_at`, `cancelled_at`, `expires_at`, `expired_at`).<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>

### Counts & Errors

- `request_counts` includes `total`, `completed`, `failed`.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
- `errors` is an object `{ "object": "list", "data": [ { "code", "message", "param?", "line?" } ] }` populated when validation fails or when the provider reported fatal errors.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup><sup>,</sup><sup>[3](https://github.com/openai/openai-python/blob/main/src/openai/types/batch.py)</sup>

## Pagination Semantics

- `GET /v1/batches` accepts `after` + `limit` query params; `limit` ∈ [1,100] with default 20.<sup>[2](https://github.com/openai/openai-python/blob/main/src/openai/resources/batches.py)</sup>
- Responses are cursor-based and include `{ object: "list", data: [...], has_more: bool, first_id: string, last_id: string }` per the Azure REST examples.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
- Clients expect stable ordering (most recent first in Azure docs) and `has_more=false` when the final page is returned.

## Result & Error Files

- Output and error artifacts are retrieved through `/v1/files/{file_id}/content` and contain NDJSON rows.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
- Each line mirrors the following schema shown in the Azure reference sample:<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>

```json
{
  "id": "batch_req_123",
  "custom_id": "task-0",
  "response": {
    "status_code": 200,
    "request_id": "req_abc",
    "body": { "id": "chatcmpl-...", "object": "chat.completion", ... }
  },
  "error": null
}
```

- Failed lines place the payload under `"error"` with the same `{status_code, request_id, body}` shape; `response` becomes `null`. This is what the official docs show for troubleshooting scenarios.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
- The gateway should keep writing NDJSON until the worker flushes to our file store. Respect the upstream `output_expires_after` TTL when setting file expirations.

## Implementation Notes for Open Model Gateway

1. **State mapping:** Our current `queued` → `in_progress` → `completed/failed` flow must be translated to the upstream names (`validating`, etc.) and record timestamps on every transition.
2. **Max concurrency:** Honor the per-batch `max_concurrency` server-side, but clamp to our configured ceilings before accepting the job.
3. **Validation feedback:** Populate the `errors` list when JSONL parsing fails (e.g., `invalid_json_line`, `empty_file`) so clients receive the same codes shown in Azure’s troubleshooting guide.<sup>[1](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/batch)</sup>
4. **Pagination:** Switch list queries to cursor semantics so SDKs that rely on `after` don’t break.
5. **Result schema:** When writing NDJSON we must include upstream IDs, HTTP codes, and provider request IDs so the downloaded files drop-in replace OpenAI’s artifacts.

Keeping this sheet updated as the spec evolves prevents backend/frontend drift and makes it clear which behavior is contractually required before coding.

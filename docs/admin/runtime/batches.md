---
title: batch guide
description: Align batch jobs with OpenAI semantics
---
[**Open Model Gateway**](/) mirrors the OpenAI and Azure `/v1/batches` contract so SDKs work unchanged.

---

#### Know supported endpoints
| Requirement | Notes |
| --- | --- |
| Allowed URLs | `/v1/responses`, `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, `/v1/moderations`. |
| Completion window | Only `"24h"` is valid today; reject custom windows until upstream expands the API. |
| Input format | JSONL uploaded via `/v1/files` with `purpose=batch`; each line must include `{ "custom_id", "method": "POST", "url": "/v1/...", "body": { ... } }`. |
| Metadata limits | Up to 16 key/value pairs, max 64-character keys and 512-character values. |
| Output TTL | Accept optional `output_expires_after` seconds and clamp against your configured `batches.max_ttl`. |

---

#### Track lifecycle
| Status | Meaning | Timestamp expectations |
| --- | --- | --- |
| `validating` | JSONL + quotas under review. | `created_at` only. |
| `in_progress` | Worker is executing requests. | `in_progress_at`. |
| `finalizing` | Results are being flushed to files. | `finalizing_at`. |
| `completed` | All items processed; errors file may still hold failed rows. | `completed_at`, `expires_at`. |
| `failed` | Batch aborted due to validation/runtime error. | `failed_at`, `errors` filled. |
| `cancelling` → `cancelled` | User requested cancellation before completion. | `cancelling_at`, `cancelled_at`. |
| `expired` | Output TTL elapsed. | `expired_at`. |

`request_counts.total/completed/failed` must stay in sync with status changes so exports show accurate throughput.

---

#### Handle files
Output and error artifacts reuse `/v1/files/{id}/content` and mirror Azure’s NDJSON schema.

```json
{
  "id": "batch_req_123",
  "custom_id": "task-0",
  "response": {
    "status_code": 200,
    "request_id": "req_abc",
    "body": { "id": "chatcmpl-...", "object": "chat.completion" }
  },
  "error": null
}
```

Failed rows move the payload under `error` while nulling `response`, so downstream tooling can diff successes and failures quickly.

---

#### Enforce limits
Clamp per-job concurrency to `min(request.max_concurrency, batches.max_concurrency)` before accepting the work so rogue uploads never starve the worker.
Switch list APIs to cursor pagination (`after`, `limit`, `has_more`) to match SDK expectations and keep ordering stable (newest first).

---

#### Research
- Adopted the authoritative content from `docs/runtime/batches.md` and upstream Azure/OpenAI references cited there.
- Confirmed status timestamps and NDJSON schema with the latest worker implementation in `backend/internal/runtime/batches`.

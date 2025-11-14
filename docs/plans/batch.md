# Batch API Work Plan

## Goals
- Implement the OpenAI Batch Jobs surface so tenants can submit asynchronous workloads that fan out across existing `/v1/*` endpoints (chat, embeddings, images, future audio/files integrations).
- Provide a job orchestration layer in the gateway (queue + worker) because upstream providers have uneven support for batches; we need to mimic OpenAI semantics regardless of provider.
- Ensure jobs reference tenant-owned files (from the Files plan) and emit per-request usage/budget data.

## API Surface
Mirror OpenAI’s current batch API:

| Endpoint | Behavior |
|----------|----------|
| `POST /v1/batches` | Submit a batch JSON describing input file IDs, endpoint to target (`chat.completions`, `embeddings`, etc.), and completion webhook. Returns batch object (`id`, `status`, `created_at`, `request_counts`). |
| `GET /v1/batches` | List recent batches (pagination). |
| `GET /v1/batches/:id` | Fetch batch status + stats (in_progress/completed/cancelled/failed). |
| `POST /v1/batches/:id/cancel` | Request cancellation (best effort). |
| `GET /v1/batches/:id/output` | Download results file (mirrors Files API; stored in S3/local). |

## Architecture

1. **Metadata Table** (Postgres): `batches(id, tenant_id, status, endpoint, input_file_id, completion_window, max_concurrency, result_file_id, error_file_id, created_at, completed_at, cancelled_at, stats_json)`.
2. **Queue + Workers**:
   - Use Redis streams or Postgres job queue (`batches_jobs` table) to track job items per batch.
   - Worker goroutine(s) dequeue requests, call the specified `/v1/*` handler internally (reuse router factory), capture response/error, and append to output file.
   - Enforce per-batch concurrency and global worker pool limits.
3. **Files integration**:
   - Batch input references file IDs uploaded via `/v1/files`. On submission, load each file, parse NDJSON/JSONL, and materialize job items.
   - Batch outputs (success and failure logs) written as files and exposed via `/v1/files/:id/content`.

## Config & Limits

Add `batches.*` config block:

```yaml
batches:
  max_requests: 5000
  max_concurrency: 50
  default_ttl: 168h
  storage:
    results_bucket: ...   # reuse files storage config
```

Tie into budget enforcement: each job counts toward tenant usage; if budget exhausted mid-batch, remaining jobs transition to `failed` with budget error.

## Job Lifecycle

1. **Submit**: Validate `endpoint` (must be one of the supported public APIs), check file existence/ownership, enqueue job items.
2. **Process**: Worker pulls job, obtains auth context (tenant, API key, overrides), calls handler (with fake HTTP context) so existing limits/idempotency apply. Capture full response + status.
3. **Persist**: Write per-item results to NDJSON stored as a file. Maintain `stats_json` with counts (total, succeeded, failed, cancelled).
4. **Complete**: When all items finished or cancelled, mark batch `completed`/`failed`. Optionally POST to a tenant-supplied webhook (signed) with status summary.
5. **Cleanup**: Respect TTL (default 7d, configurable) for batch metadata and output files (reuse sweeper from Files plan).

## Error Handling & Validation

- Enforce request/batch size limits (max items, max file size). Reject unsupported endpoints immediately.
- Provide detailed status transitions: `queued`, `running`, `completed`, `failed`, `cancelled`, `expired`.
- Allow cancellation to stop new work, but in-flight jobs finish (mark partial results accordingly).

## Docs & UI

- Update `API.md` with batch endpoints, JSON examples, and status enums.
- Add `docs/architecture/batches.md` describing worker design, config, and failure scenarios.
- Future UI work: add Batch tab for tenants to monitor jobs.

## Work Breakdown

1. Schema migrations for `batches` + `batch_items` (and optional webhook table).
2. Config (`batches.*`) + storage integration for output files.
3. Queue & worker implementation (Redis streams or Postgres polling + goroutines).
4. HTTP handlers for `/v1/batches*` with OpenAI-compatible JSON.
5. Tests: unit tests for submission/validation, worker processing, cancellation; integration tests using small sample files.
6. Docs updates (PRD, API.md, new architecture doc).


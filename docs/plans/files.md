# Files API Work Plan

## Goals
- Mirror the full OpenAI Files surface (`/v1/files` CRUD, `/v1/files/:id/content`, `/v1/uploads`) so tenants can manage documents used by downstream jobs (fine-tuning, batches, etc.) even if upstream providers don’t support these endpoints.
- Add a storage subsystem that can write file bodies to S3 (prod) or local disk (dev) with metadata tracked in Postgres.
- Provide configurable at-rest encryption, TTL (default 7 days, max 30 days), and max upload size limits (separate from the existing 20 MB body cap).

## API Surface

| Endpoint | Behavior |
|----------|----------|
| `GET /v1/files` | List files scoped to the authenticated tenant. |
| `POST /v1/files` | Upload file (multipart/form-data). Supports configurable max size (e.g., 200 MB). |
| `GET /v1/files/:id` | Return metadata (id, filename, bytes, purpose, created_at, expires_at, status). |
| `DELETE /v1/files/:id` | Soft delete (mark deleted, remove object from storage). |
| `GET /v1/files/:id/content` | Stream raw file bytes back to the tenant (with `Content-Type` from metadata). |
| `POST /v1/uploads` | Optional chunked upload endpoint (match OpenAI “upload sessions”) if needed for larger files. |

## Storage Architecture

1. **Metadata Table** (Postgres): `files(id uuid, tenant_id uuid, filename, purpose, bytes bigint, content_type, storage_key, storage_backend, checksum, encrypted bool, expires_at timestamptz, created_at, deleted_at)`.
2. **Backends**:
   - **S3-compatible**: Use AWS SDK (or MinIO for dev). Bucket/prefix configurable. Server-side encryption (SSE-S3/KMS) or client-side encryption using a configured key.
   - **Local disk**: Dev convenience (write under `./data/files`), still encrypt-at-rest using envelope encryption (AES-GCM with key from config).
3. **Encryption**: Introduce `files.encryption_key` (or use KMS) to encrypt before writing to storage. Store IV + auth tag in metadata.

## TTL & Retention

- New config block `files`:
  ```yaml
  files:
    max_size_mb: 200
    default_ttl: 168h    # 7 days
    max_ttl: 720h        # 30 days
    storage: s3          # or local
    s3:
      bucket: ...
      prefix: ...
      region: ...
    encryption_key: base64:aes256key==
  ```
- Upon upload, set `expires_at = now + min(requested_ttl, max_ttl)`; default to `default_ttl`. Background job sweeps expired files (delete metadata + object).

## Gateway Behavior (since providers may not support files)

- Treat files as a tenant-owned asset: the router handles uploads/downloads entirely, regardless of provider capabilities.
- When jobs (batch, fine-tuning) need files, they reference the stored metadata (ID → storage key). When calling providers that support file IDs, the router uploads the file on-demand or streams it as part of the request, depending on provider requirements.
- Provide `purpose` field validation (e.g., `fine-tune`, `batch`, `assistants`) to mirror OpenAI and allow future policy gating.

## Request Handling

- **Upload**: Accept multipart (per OpenAI), enforce max size using `files.max_size_mb`. Validate file type if necessary (configurable allowlist). Compute checksum (SHA256) for dedupe/integrity.
- **Download**: Stream via Fiber with appropriate headers, ensure tenant authorization.
- **Delete**: Mark metadata deleted, remove storage object asynchronously (retry/backoff).
- **List**: Paginate (limit/after) similar to OpenAI responses.

## Admin / Ops

- Add documentation in `API.md` and provider docs describing the Files feature, quotas, TTL.
- Extend admin UI later to surface file listings/backups (optional).
- Metrics: track upload counts/bytes, storage errors, TTL sweeps.

## Work Breakdown

1. **Config & Schema**: Add `files.*` config block, new SQL migration for `files` table.
2. **Storage layer**: Implement `internal/storage/files` with pluggable backends (S3/local) and encryption helpers.
3. **HTTP routes**: Add `/v1/files` handlers under `internal/httpserver/public/files.go`, following OpenAI response schema.
4. **Background sweeper**: Cron/goroutine that deletes expired files and cleans storage.
5. **Docs**: Update PRD + `API.md`; add new doc explaining config, TTL, curl examples.


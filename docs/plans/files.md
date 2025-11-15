# Files API Work Plan

## Goals
- Mirror the full OpenAI Files surface (`/v1/files` CRUD, `/v1/files/:id/content`, `/v1/uploads`) so tenants can manage documents used by downstream jobs (fine-tuning, batches, etc.) even if upstream providers don’t support these endpoints.
- Add a storage subsystem that can write file bodies to S3 (prod) or local disk (dev) with metadata tracked in Postgres.
- Provide configurable at-rest encryption, TTL (default 7 days, max 30 days), and max upload size limits (separate from the existing 20 MB body cap).

## API Surface

| Endpoint | Behavior |
|----------|----------|
| `GET /v1/files` | List files scoped to the authenticated tenant. Supports `limit` (1-100), cursor-based `after`, and optional `purpose` filter. Response mirrors OpenAI’s `{object:\"list\", data:[...], has_more:false}` envelope. |
| `POST /v1/files` | Upload file (multipart/form-data). Supports configurable max size (e.g., 200 MB). |
| `GET /v1/files/:id` | Return metadata (id, filename, bytes, purpose, created_at, expires_at, status). We also expose `status_details` and `deleted` fields. |
| `DELETE /v1/files/:id` | Soft delete (mark deleted, remove object from storage) and return `{id, object:\"file\", deleted:true}`. |
| `GET /v1/files/:id/content` | Stream raw file bytes back to the tenant (with `Content-Type` from metadata). |
| `POST /v1/uploads` | OpenAI’s “upload session” endpoint. Returns an upload session ID, chunk `part_size`, and `url`. Clients upload chunks via the provided pre-signed URL, then finalize with `/v1/uploads/{upload_id}/complete`. (We can initially respond with 501 until the session flow is ready.) |

### OpenAI Spec Summary

- **FileObject fields**: `id`, `object` (`"file"`), `bytes`, `created_at` (unix seconds), `filename`, `purpose`, `status` (`"uploading"`, `"uploaded"`, `"processed"`, `"error"`, `"deleted"`), `status_details` (string), `deletion_strategy` (optional), `document_type` (optional future), and for lists `deleted` (bool) when returned by `DELETE`.
- **List response**: `{"object":"list","data":[FileObject...],"has_more":false,"first_id":...,"last_id":...}`. `limit` defaults to 100 (max 100), `after` is a cursor referencing the last returned ID.
- **Upload request**: multipart with fields `purpose` (e.g., `fine-tune`, `batch`, `assistants`, `assistants_output`, `vision`, `moderation`, `responses`, `fine-tune-results`) and `file`. Optional `expires_in` (seconds) controls TTL but must not exceed `max_ttl`. OpenAI returns a FileObject with `status="uploaded"`.
- **Download**: `GET /v1/files/{id}/content` returns raw bytes with `Content-Type` header; 404/403 when missing or unauthorized.
- **Delete**: Soft delete, returning `{id, object:"file", deleted:true}` while the file remains queryable in audit logs.
- **Chunked uploads**: `/v1/uploads` returns `{id, object:"upload", created_at, expires_at, part_size, url}`; clients `PUT` parts to the provided pre-signed URL and call `/v1/uploads/{id}/parts` + `/complete`. We can stub `POST /v1/uploads` initially with 501 but need the types defined for future compatibility.
- **Error semantics**: invalid purpose → 400, file too large → 400, TTL > max → 400, expired/deleted → 404, unauthorized tenant → 403.

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

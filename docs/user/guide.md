# User Guide

This guide explains how API consumers interact with Open Model Gateway—both through the user portal and via OpenAI-compatible HTTP endpoints.

## Getting Access

1. Ask an administrator to invite you (either to a shared tenant or to create a personal account).
2. Log in at `https://<gateway-host>/` (use **Continue with SSO** if your org enabled OIDC; otherwise use the local email/password form).
3. Go to **API Keys** and create an API key for the tenant you want to use. Copy the secret immediately—keys are only shown once. The dialog lets you specify optional per-key budgets and RPM/TPM/parallel overrides; the UI displays the maximum allowed values based on your personal defaults or the selected tenant so you know the ceiling before issuing a key.

Personal tenants are labelled **Personal** inside the UI and scoped to your user account only.

If your organization configures OIDC role requirements, make sure your Authentik/IdP account has one of the allowed roles; otherwise the router will reject the login even after SSO succeeds.

## User Portal Overview

| Section | Purpose |
| --- | --- |
| **Dashboard** | Usage snapshots, spend, recent activity. |
| **Usage** | Detailed spend graphs with tenant/model selectors, timezone overrides, and a “Custom range” picker to match specific billing windows. |
| **Tenants** | Lists the tenants you belong to, their budgets/rate limits, and (for owners/admins) membership management tools. |
| **API Keys** | Generate/revoke API keys per tenant. |
| **Files** | Upload JSONL/docs, view per-file status, paginate through history, and download raw content without leaving the browser. |
| **Batches** | Monitor NDJSON batch jobs and download output/error NDJSON from the UI. The table defaults to your Personal tenant and only lists tenants you belong to. |

The Batches view includes:

- Scope selector (defaults to your personal tenant).
- Status + progress table.
- “Finished” column showing completion timestamps.
- Buttons to download `output` or `errors` without leaving the portal.
- Cursor-friendly pagination (the UI calls `GET /v1/batches?limit=...&after=...`, and the API responds with `has_more`, `first_id`, `last_id`).

### Usage Comparison API

- `GET /user/usage/compare` mirrors the admin comparison endpoint but automatically scopes data to your personal + membership tenants.
- Optional query params:
  - `tenant_ids` – comma-separated UUIDs to focus the series (defaults to all of your tenants when omitted and no model aliases are specified).
  - `model_aliases` – comma-separated model aliases to compare usage for the models you actually consumed (still filtered to your tenant scope).
  - `period` / `timezone` – same options as the dashboard (`7d`, `30d`, `UTC`, `America/New_York`, etc.).
  - `start` + `end` – optional RFC3339 timestamps for custom ranges (both required, max 180 days). When supplied they override `period` so you can request exact billing windows.
- Response shape:

```json
{
  "period": "30d",
  "timezone": "America/New_York",
  "series": [
    {
      "kind": "tenant",
      "id": "<tenant-uuid>",
      "label": "Personal",
      "totals": {"requests": 1200, "tokens": 180000, "cost_cents": 423, "cost_usd": 4.23},
      "points": [{"date": "2025-11-01T00:00:00-04:00", "requests": 40, ...}]
    },
    {
      "kind": "model",
      "id": "gpt-4o-mini",
      "provider": "openai",
      "totals": {"requests": 600, ...},
      "points": [...]
    }
  ]
}
```

Front-end overlays can call this endpoint to draw multi-tenant or tenant+model charts without batching separate `/usage` requests.

### Model Catalog Helper

- `GET /user/models` returns the subset of model catalog entries the current user can target. Useful for populating model selectors in the user portal without exposing admin-only metadata.
- Response: `{ "models": [{ "alias": "gpt-4o-mini", "provider": "openai", "enabled": true }, ...] }`.

### Managing Tenants & Invites

- Every tenant you belong to appears as a card plus a tabular list. Click **Manage** to view budgets, remaining spend, and membership details.
- Owners and admins can invite teammates by supplying an email + role (owner, admin, viewer, or user) and optionally an initial password for local auth. Invites immediately create the membership so the new user can log in with their personal tenant.
- Existing memberships can be refreshed or removed from the same dialog (admins may edit non-owner roles; only owners can grant/remove the owner role).
- Membership tables show who invited whom, the assigned role, and whether an entry corresponds to your own account.

## OpenAI-Compatible APIs

All requests use the standard OpenAI headers (`Authorization: Bearer <api-key>`, `Content-Type: application/json`). Supported endpoints:

| Path | Notes |
| --- | --- |
| `POST /v1/chat/completions` | Streaming + non-streaming chat. |
| `POST /v1/embeddings` | Text embeddings. |
| `POST /v1/images/generations` | Image generation (models must expose image capabilities). |
| `POST /v1/images/edits` | Supply images + optional mask for edit/extension (OpenAI/OpenAI-compatible adapters today). |
| `POST /v1/images/variations` | Remix a single image (`n` ≤ 10). Same provider constraints as edits. |
| `GET /v1/models` | Lists the catalog. |
| `POST /v1/files` / `GET /v1/files` / `DELETE /v1/files/:id` | File upload, listing, download. Supports `limit` (1–100), cursor-based `after`, optional `purpose=batch|fine-tune|...` filters, and OpenAI-style `{has_more, first_id, last_id}` metadata. |
| `POST /v1/audio/transcriptions` / `/translations` | Audio transcription/translation (subject to provider support). |
| `POST /v1/audio/speech` | Text-to-speech (returns binary audio; use `-o` when using curl). |
| `POST /v1/batches` | NDJSON batch ingestion. Supports `limit` (1–100) + `after` cursors on `GET /v1/batches` and returns OpenAI-style `errors`, `cancelling_at`, and `expired_at` fields. Metadata is limited to 16 key/value pairs (keys ≤ 64 chars, values ≤ 512 chars). |

### Chat Example

```bash
curl http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-4o-azure",
        "messages": [
          {"role": "system", "content": "You are a concise assistant."},
          {"role": "user", "content": "Explain Open Model Gateway in one sentence."}
        ],
        "stream": false
      }'
```

For streaming responses, set `"stream": true` and read the SSE frames exactly like OpenAI’s API.

### Files API Examples

```bash
curl http://localhost:8090/v1/files \
  -H "Authorization: Bearer $API_KEY" \
  -F "file=@prompt.jsonl" \
  -F "purpose=batch" \
  -F "expires_in=604800"
```

Notes:
- `expires_in` is expressed in seconds (e.g., `604800` for seven days). The backend clamps the value to `files.max_ttl` and defaults to `files.default_ttl` if omitted.
- Responses include `status` (`uploading`, `uploaded`, `processed`, `error`, or `deleted`) plus optional `status_details` to mirror OpenAI’s FileObject shape.
- `DELETE /v1/files/:id` responds with `{id, object:"file", deleted:true}`.

List files with cursor pagination and purpose filtering:

```bash
curl "http://localhost:8090/v1/files?limit=20&after=$LAST_ID&purpose=batch" \
  -H "Authorization: Bearer $API_KEY" | jq
```

Sample response:

```json
{
  "object": "list",
  "data": [
    {
      "id": "file_abc123",
      "object": "file",
      "filename": "prompt.jsonl",
      "purpose": "batch",
      "bytes": 1024,
      "created_at": 1739908702,
      "expires_at": 1740513502,
      "status": "uploaded",
      "status_details": null
    }
  ],
  "has_more": true,
  "first_id": "file_abc123",
  "last_id": "file_abc123"
}
```

Use the returned `last_id` as the next `after` cursor while `has_more` is `true`. Download file contents through `GET /v1/files/{id}/content`—the router streams the persisted blob with the original `Content-Type`.

### Image Edit Example

```bash
curl http://localhost:8090/v1/images/edits \
  -H "Authorization: Bearer $API_KEY" \
  -F "model=gpt-image-1" \
  -F "prompt=Extend the skyline with neon reflections" \
  -F "image=@./base.png" \
  -F "mask=@./mask.png" \
  -F "n=2" \
  | jq -r '.data[0].b64_json' | base64 --decode > edit.png
```

### Image Variation Example

```bash
curl http://localhost:8090/v1/images/variations \
  -H "Authorization: Bearer $API_KEY" \
  -F "model=gpt-image-1" \
  -F "image=@./base.png" \
  -F "response_format=b64_json" \
  -F "n=2" \
  | jq -r '.data[0].b64_json' | base64 --decode > variation.png
```

OpenAI + OpenAI-compatible adapters, Vertex Imagen/Nano Banana, and Bedrock Stable Diffusion aliases (where `bedrock_image_task_type` targets a diffusion model) implement edits and variations. Azure image routes and Bedrock Titan entries still return `image_operation_unsupported`, so fall back to a supported alias when necessary.

### Batch Example

1. Upload an NDJSON file (one request per line) using the previous endpoint.
2. Submit the batch:

```bash
curl http://localhost:8090/v1/batches \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "input_file_id": "uuid-from-upload",
        "endpoint": "/v1/chat/completions",
        "completion_window": "24h",
        "metadata": {"label": "ad hoc run"}
      }'
```

3. Poll `/v1/batches/:id` or open the **Batches** tab in the portal to monitor progress.
4. Download output/error NDJSON via `curl` or the UI buttons.

> The list API supports cursor pagination via `limit` (1–100) and `after=batch_id` so long-running jobs can stream results. Each batch response mirrors OpenAI’s schema, including an `errors` list when validation/runtime issues occur and the `cancelling_at`/`expired_at` timestamps, so SDKs can understand the lifecycle without custom glue.

### Audio Example

```bash
curl http://localhost:8090/v1/audio/transcriptions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: multipart/form-data" \
  -F "file=@meeting.mp3" \
  -F "model=whisper-1"
```

Uploads are restricted by `audio.max_upload_mb`. Only models that advertise audio capabilities will be routed.

### Audio Speech Example

```bash
curl http://localhost:8090/v1/audio/speech \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -o speech.mp3 \
  -d '{
        "model": "gpt-4o-mini-tts",
        "voice": "alloy",
        "format": "mp3",
        "input": "Welcome to the Open Model Gateway"
      }'
```

The response is binary audio (mp3 by default). Streaming speech (`"stream": true`) is not supported yet, so omit that flag for now.

## Tenant Budgets & Rate Limits

- Budgets are enforced per tenant. When a request would exceed the remaining budget you’ll receive a `402` response with `budget_exceeded`.
- Rate limits (TPM/RPM/parallel) are enforced using Redis. Errors follow OpenAI’s schema (`rate_limit_error`).
- Operators can override limits per tenant or per API key; check the **Tenants** or **API Keys** tabs to see current values.

## Troubleshooting

| Issue | Resolution |
| --- | --- |
| `401 unauthorized` | Ensure you’re using the current API key and sending the `Authorization` header. |
| `404 model_not_found` | The alias isn’t enabled for your tenant. Ask an admin to assign the model. |
| `429 rate_limit_error` | Slow down or request higher limits from the admin team. |
| Batch output download fails | Refresh the page and try again; if it persists, share the batch ID with support. |
| File upload rejected | Check `files.max_size_mb` and TTL settings shown in this guide. |
| `image_operation_unsupported` | The alias only supports generations. Switch to an OpenAI/OpenAI-compatible, Vertex Imagen, or Bedrock Stable Diffusion model for edits/variations. |

## Support

- Administrators can audit tenant events under the admin portal (**Settings → Audit Log**).
- Provide request IDs (from the `X-Request-Id` response header) when opening tickets; the backend logs use the same IDs.

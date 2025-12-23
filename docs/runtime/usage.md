# Usage Exports & Billing Webhooks

Open Model Gateway can generate async usage exports (CSV/Parquet) and push monthly summaries to billing webhooks. Both surfaces exist for admin (`/admin/*`) and tenant-scoped user (`/user/*`) APIs.

## Authorization

- Admin endpoints accept either admin session tokens (browser login) or Admin API tokens created in **Settings -> Admin tokens**.
- User endpoints require a user portal access token scoped to the tenant.

All requests use `Authorization: Bearer <token>`.

## Usage Exports

Exports are created asynchronously. Create an export, poll until `status=ready`, then download the file.

### Create export (admin)

`POST /admin/usage-exports`

Body:

```json
{
  "tenant_ids": ["uuid-1", "uuid-2"],
  "tenant_id": "uuid-3",
  "period": "30d",
  "start": "2025-01-01",
  "end": "2025-01-31",
  "granularity": "daily",
  "format": "csv",
  "timezone": "America/New_York"
}
```

Notes:
- Provide either `start` + `end` or `period`. `period` defaults to `30d`.
- `tenant_ids` is an array and `tenant_id` is an optional single-id convenience field. When both are provided they are combined.
- Non-super admins must provide at least one tenant ID. Super admins may omit tenant IDs to export all tenants.
- `granularity`: `daily` (default) or `monthly`.
- `format`: `csv` (default) or `parquet`.
- `timezone`: IANA zone name; defaults to `reporting.timezone`.
- Maximum export range is 90 days.

### Create export (user)

`POST /user/usage-exports`

Body:

```json
{
  "tenant_id": "uuid-1",
  "period": "7d",
  "granularity": "daily",
  "format": "csv",
  "timezone": "UTC"
}
```

Notes:
- `tenant_id` is optional; omit it to export the caller's personal tenant.
- Users can only export tenants they belong to.

### List exports

`GET /admin/usage-exports?limit=25&offset=0`  
`GET /user/usage-exports?limit=25&offset=0`

### Poll export

`GET /admin/usage-exports/{id}` or `GET /user/usage-exports/{id}`

Responses include `status`, timestamps, and (for admin endpoints) `download_url` once ready.

### Download export

`GET /admin/usage-exports/{id}/content` or `GET /user/usage-exports/{id}/content`

Returns the CSV/Parquet file as an attachment. For admin exports you may also use the `download_url` field when present.

### Export rows

Each export row includes:
- `bucket_start`
- `tenant_id`, `tenant_name`
- `model_alias`, `provider`
- `requests`, `input_tokens`, `output_tokens`, `total_tokens`
- `cost_cents`, `cost_usd_micros`, `cost_usd`

### Tenant IDs

Use one of the list APIs to fetch tenant IDs:

```bash
curl -sS http://localhost:8090/admin/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
| jq -r '.tenants[] | "\(.id)\t\(.name)"'
```

User-scoped:

```bash
curl -sS http://localhost:8090/user/tenants \
  -H "Authorization: Bearer $USER_TOKEN" \
| jq -r '.tenants[] | "\(.id)\t\(.name)"'
```

## Billing Webhooks

Billing webhooks deliver a usage summary payload to a configured endpoint. Requests are HMAC signed so recipients can verify integrity.

### Configure a webhook

`POST /admin/billing-webhooks` or `POST /user/billing-webhooks`

```json
{
  "tenant_id": "uuid-1",
  "name": "NetSuite",
  "url": "https://billing.example.com/usage",
  "secret": "super-secret",
  "enabled": true
}
```

Notes:
- `tenant_id` is required for user endpoints.
- `GET /admin/billing-webhooks?tenant_id=...` filters to one tenant; omit the query param to list all tenants.

### Update or delete

`PUT /admin/billing-webhooks/{id}` / `DELETE /admin/billing-webhooks/{id}`  
`PUT /user/billing-webhooks/{id}` / `DELETE /user/billing-webhooks/{id}`

### Dispatch a webhook

`POST /admin/billing-webhooks/{id}/dispatch` or `POST /user/billing-webhooks/{id}/dispatch`

```json
{
  "period": "2025-01",
  "start": "2025-01-01",
  "end": "2025-01-31",
  "timezone": "America/New_York"
}
```

Notes:
- Provide either `start` + `end` or `period`. `period` defaults to `30d`.
- `timezone` uses IANA names and defaults to `reporting.timezone`.

### Signature headers

Webhook requests include:
- `X-OMG-Signature-Version: v1`
- `X-OMG-Timestamp: 2025-01-31T23:59:59Z`
- `X-OMG-Signature: sha256=<hex hmac>`

The HMAC is computed as:

```
HMAC_SHA256(secret, body)
```

### Payload shape

```json
{
  "tenant_id": "uuid",
  "period_start": "2025-01-01T00:00:00Z",
  "period_end": "2025-01-31T23:59:59Z",
  "generated_at": "2025-02-01T00:00:00Z",
  "spend": {
    "cost_cents": 1234,
    "cost_usd_micros": 1234000,
    "cost_usd": 12.34,
    "currency": "USD"
  },
  "token_counts": {
    "requests": 1234,
    "input_tokens": 450000,
    "output_tokens": 150000,
    "total_tokens": 600000
  },
  "top_models": [
    {
      "model_alias": "gpt-4o",
      "provider": "openai",
      "requests": 1200,
      "tokens": 500000,
      "cost_cents": 1150,
      "cost_usd_micros": 1150000,
      "cost_usd": 11.5
    }
  ]
}
```

`top_models` is limited to the top 5 models by cost.

### Events & retries

- List events: `GET /admin/billing-webhooks/{id}/events` or `GET /user/billing-webhooks/{id}/events`
- Retry event: `POST /admin/billing-webhooks/events/{event_id}/retry` (admin only)
- Pagination: `limit` and `offset` query parameters (default 25).

Dispatcher retries honor `budgets.alert.webhook.*` settings.

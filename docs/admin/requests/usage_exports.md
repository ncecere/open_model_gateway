# Usage exports and billing hooks
Generate CSV/Parquet exports or manage billing webhooks for finance automation.

## Prepare environment
Authenticate once to reuse the admin bearer token.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" -H "Content-Type: application/json" -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Create an export
POST `/admin/usage-exports` with the time window, format, and optional filters.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/usage-exports" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "period_start": "2025-12-01T00:00:00Z",
        "period_end": "2025-12-31T23:59:59Z",
        "format": "csv",
        "tenant_ids": ["tenant_123"],
        "timezone": "America/Los_Angeles"
      }' | jq '{id, status}'
```

## Poll export status
List exports or fetch a single record to check if processing finished.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/usage-exports/{export_id}" \
  -H "Authorization: Bearer $TOKEN" | jq '{id, status, download_url}'
```

## Download export files
Use `/admin/usage-exports/{id}/content` once status equals `succeeded`.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/usage-exports/{export_id}/content" \
  -H "Authorization: Bearer $TOKEN" --output usage.csv
```

## Configure billing webhooks
POST webhook targets that receive budget or usage summaries.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/billing-webhooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "name": "netsuite",
        "url": "https://billing.example.com/hooks/usage",
        "secret": "hmac-secret",
        "retries": 3
      }' | jq '{id, name, url}'
```

## Verify events
Inspect webhook delivery logs via `/admin/billing-webhooks/{id}` or monitor `X-Request-ID` headers to correlate emitter traces.

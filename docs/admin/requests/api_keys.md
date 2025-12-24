# API keys
Issue and maintain tenant API keys with scoped budgets and rate limits.

## Prepare environment
Reuse the admin base + credential exports.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" -H "Content-Type: application/json" -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Create a key
Target the tenant-specific collection and capture the one-time secret from the response.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/api-keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "name": "demo-app",
        "budget_usd": 200,
        "warning_threshold": 0.8,
        "rate_limit": {
          "requests_per_minute": 300,
          "tokens_per_minute": 250000,
          "parallel_requests": 25
        }
      }' | jq '{id, name, prefix, secret}'
```

## Rotate a key
POST to the rotation endpoint to reissue the secret while preserving budgets and rate limits.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/api-keys/{key_id}/rotate" \
  -H "Authorization: Bearer $TOKEN" | jq '{id, prefix, secret}'
```

## Revoke a key
DELETE the resource to immediately block traffic.

```bash
curl -sS -X DELETE "$GATEWAY_ADMIN_BASE_URL/admin/api-keys/{key_id}" \
  -H "Authorization: Bearer $TOKEN" | jq '{id, deleted}'
```

## Inspect header telemetry
User calls that present the new key inherit `X-Budget-*` and `X-RateLimit-*` settings so you can verify enforcement without opening the UI.

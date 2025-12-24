# Budgets and alerts
Tune tenant and key spend limits plus alert routes via `/admin/budgets/*` helpers.

## Prepare environment
Authenticate once and reuse the bearer token for every request.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" -H "Content-Type: application/json" -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Set a tenant budget
Put overrides that include USD limit, refresh cadence, and alert targets.

```bash
curl -sS -X PUT "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/budget" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "limit_usd": 5000,
        "warning_threshold": 0.85,
        "refresh_schedule": "calendar_month",
        "alerts": {
          "emails": ["alerts@example.com"],
          "webhooks": [
            {
              "url": "https://ops.example.com/budget",
              "secret": "hmac-secret"
            }
          ]
        }
      }' | jq '{tenant_id, limit_usd, warning_threshold}'
```

## Reset a tenant budget
DELETE the override to fall back to platform defaults defined in `budgets.*` config.

```bash
curl -sS -X DELETE "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/budget" \
  -H "Authorization: Bearer $TOKEN" | jq '{tenant_id, deleted}'
```

## Update a key budget
PATCH `/admin/api-keys/{id}` with the new `budget_usd`, warning threshold, or rate-limit overrides when a workload changes.

```bash
curl -sS -X PATCH "$GATEWAY_ADMIN_BASE_URL/admin/api-keys/{key_id}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "budget_usd": 750,
        "warning_threshold": 0.7
      }' | jq '{id, budget_usd, warning_threshold}'
```

## Monitor headers
User requests inherit these settings immediately, so watch `X-Budget-*` after adjustments to confirm the policy took effect.

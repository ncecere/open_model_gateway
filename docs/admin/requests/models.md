# Models
Attach or detach catalog models for a tenant via `/admin/tenants/{id}/models`.

## Prepare environment
Authenticate once and reuse the bearer token.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" -H "Content-Type: application/json" -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Attach a model
Provide the `model_id` plus enablement flag to make the alias available to the tenant.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/models" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "model_id": "gpt-oss-20b",
        "enabled": true
      }' | jq '{tenant_id, model_id, enabled}'
```

## Detach or suspend
Send `enabled:false` or DELETE the relationship depending on whether you want to keep historical context.

```bash
curl -sS -X PATCH "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/models/{model_id}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled":false}' | jq '{tenant_id, model_id, enabled}'
```

## Query assignments
List current attachments to verify which aliases are tenant-assignable.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}/models" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {model_id, enabled, routing_weight}'
```

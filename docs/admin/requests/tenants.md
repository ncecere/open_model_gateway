# Tenants
Provision and manage tenants through `/admin/tenants` endpoints.

## Prepare environment
Set both the public and admin bases so curl can locate the router.

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
```

## Authenticate
Exchange credentials for an access token and reuse it for subsequent calls.

```bash
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Create a tenant
POST minimal fields plus metadata describing who initiated the request.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/tenants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "name": "sdk-demo",
        "status": "active",
        "metadata": {"created_by": "curl"}
      }' | jq '{id, name, status}'
```

## List tenants
Filter by status or search term with query parameters and stitch results into your internal CMDB.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/tenants?status=active" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {id, name, status, budget: .budget_summary}'
```

## Update a tenant
PATCH status or metadata as business rules evolve.

```bash
curl -sS -X PATCH "$GATEWAY_ADMIN_BASE_URL/admin/tenants/{tenant_id}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"suspended","metadata":{"reason":"payment"}}' | jq '{id, status}'
```

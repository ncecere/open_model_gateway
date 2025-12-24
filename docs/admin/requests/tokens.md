# Admin tokens
Obtain bearer tokens for automation via session login or long-lived admin API keys.

## Prepare environment
Set base URLs and credentials before requesting tokens.

```bash
export GATEWAY_ADMIN_BASE_URL="http://localhost:8090"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="change-me"
```

## Login for a short-lived token
Call `/admin/auth/login` and store the `access_token` for CI jobs.

```bash
TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"'$ADMIN_EMAIL'","password":"'$ADMIN_PASSWORD'"}' | jq -r '.access_token')
```

## Issue an admin API token
Create a scoped key tied to your admin account for background automation.

```bash
ADMIN_TOKEN=$(curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/admin-keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "name": "ops-export",
        "scope": "system",
        "expires_in_seconds": 2592000
      }' | jq -r '.token')
```

## Use the admin token
Call any `/admin/*` endpoint with the token while respecting its scope and expiry date.

```bash
curl -sS "$GATEWAY_ADMIN_BASE_URL/admin/usage-exports" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data[0:5]'
```

## Rotate tokens
Delete or rotate tokens after audits and record issuance details in your internal inventory.

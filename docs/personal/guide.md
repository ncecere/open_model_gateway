# Personal use guide

This guide is for individuals who want to use Open Model Gateway without managing a shared tenant. Personal tenants are created automatically and let you experiment with models while keeping your data isolated.

## Access your personal tenant

1. Sign in to the user portal at `/` using either local credentials or SSO.
2. Open **Tenants** to confirm the **Personal** tenant exists and shows your membership.
3. If you belong to additional shared tenants, use the selector to switch context.

![TODO: User portal tenants list with Personal highlighted](../assets/screenshots/user-portal-personal-tenant.png)

## Create a personal API key

1. Go to **API Keys**.
2. Stay on the **Personal** tab and click **Create**.
3. Copy the secret once and store it in your password manager or secret vault.

![TODO: Personal API key create dialog](../assets/screenshots/user-personal-key-create.png)
![TODO: Issued secret confirmation banner](../assets/screenshots/user-personal-key-secret.png)

## Test a request

```bash
export GATEWAY_BASE_URL="http://localhost:8090/v1"
export OPENAI_API_KEY="sk-your-personal-secret"

curl -sS "$GATEWAY_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
        "model": "gpt-4o-mini",
        "messages": [{"role": "user", "content": "Summarize current usage."}]
      }' | jq '.choices[0].message.content'
```

## Monitor usage and budgets

- **Usage** shows spend by day and model for your personal tenant.
- The budget cards show remaining spend and reset times.
- Response headers (`X-Budget-*`, `X-RateLimit-*`) confirm enforcement for each request.

![TODO: User portal usage dashboard](../assets/screenshots/user-portal-usage.png)

## Rotate or revoke keys

1. Open **API Keys -> Personal**.
2. Use **Rotate** to issue a new secret without losing budget history.
3. Use **Revoke** when you no longer need the key.

![TODO: Personal API key table with rotate/revoke actions](../assets/screenshots/user-personal-key-actions.png)

## Need shared access?

If you need access to a shared tenant or higher quotas, contact your tenant owner or platform admin. They can invite you to a tenant and provision shared keys.

## Related references

- `docs/user/guide.md` for broader user portal workflows.
- `docs/reference/requests/README.md` for endpoint examples.
- `docs/reference/troubleshooting.md` for error details.

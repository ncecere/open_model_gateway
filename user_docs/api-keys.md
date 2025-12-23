# API Keys and Admin Tokens

There are two key types:

- **API keys**: used for `/v1/*` model operations (chat, embeddings, images, etc.).
- **Admin tokens**: used for `/admin/*` automation (tenants, models, exports).

This guide explains how to issue, use, and rotate both.

## Create an API Key (Tenant or Personal)

### From the Admin Portal (tenant keys)

1. Open **Admin -> Keys**.
2. Choose a tenant in the scope selector.
3. Click **Create key**.
4. Fill the form:
   - **Name**: human label for audit logs.
   - **Budget override** (optional): USD cap for this key.
   - **Warning threshold** (optional): 0.0–1.0 fraction of the budget.
   - **RPM / TPM / Parallel** (optional): rate limit overrides.
5. Save and copy the secret. It is displayed once.

### From the User Portal (personal or tenant keys)

1. Open **User -> API Keys**.
2. Select **Personal** or a tenant scope.
3. Click **Create key** and fill the same fields.
4. Copy the secret immediately.

## Key Fields Explained

- **Budget override**: hard stop once spend reaches this amount.
- **Warning threshold**: triggers alerts before shutdown.
- **RPM**: requests per minute limit.
- **TPM**: tokens per minute limit.
- **Parallel**: max concurrent requests.
- **Scope**: determines which tenant receives usage and budget accounting.

Key limits cannot exceed tenant-level caps. Tenant caps cannot exceed global defaults.

## Use an API Key

Chat example:

```bash
curl http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

Embeddings example:

```bash
curl http://localhost:8090/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "Vectorize this."
  }'
```

Images example:

```bash
curl http://localhost:8090/v1/images/generations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-1",
    "prompt": "Line art of a hummingbird",
    "size": "1024x1024"
  }'
```

## Create an Admin Token

1. Open **Admin -> Settings -> Admin tokens**.
2. Choose scope:
   - **admin**: tied to your admin account.
   - **system**: privileged token for shared operational tooling.
3. Set expiration (required).
4. Save and copy the token once.

Use the admin token:

```bash
curl http://localhost:8090/admin/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Rotation and Revocation

- **Rotate API key**: issues a new secret and revokes the old one.
- **Revoke API key**: stops all usage immediately.
- **Admin tokens**: create a new token with a new expiration; revoke old tokens.

## Common Errors

- **401 Unauthorized**: token missing or invalid.
- **403 Forbidden**: wrong token type for the endpoint.
- **429 Too Many Requests**: rate limits exceeded.
- **402 Budget exceeded**: budget limit reached.

## Rules of Thumb

- Use tenant keys for production apps.
- Use personal keys for developer testing.
- Use admin tokens only for automation or operator tooling.

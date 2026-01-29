# Standard User Guide

This guide is for developers and analysts who consume the OpenAI-compatible public API surface (`/v1/*`). It explains how to obtain a key, choose models, send requests, and interpret the headers the gateway adds for budgets and rate limits.

## Getting Access

1. Ask your tenant owner for a membership invitation. Provide the email that matches your SSO identity (if enforced).
2. Sign in to the user portal at `/`. Personal tenants are auto-created for individual experimentation.
3. Request an API key:
   - **Personal**: go to **API Keys -> Personal -> Create**.
   - **Shared tenant**: coordinate with the tenant owner; they can issue shared or per-user keys from the admin portal.
4. Store the key securely (password manager, secret manager). Keys follow the OpenAI pattern (`sk-<prefix>.<secret>`).

![TODO: User portal login screen](../assets/screenshots/user-login.png)
![TODO: User portal API keys page](../assets/screenshots/user-api-keys.png)

## Making Requests

- Use any OpenAI client/SDK (curl, Python, TypeScript, etc.). Point the base URL at your gateway deployment (e.g., `http://gateway.example.com/v1`).
- Add the header `Authorization: Bearer <your_api_key>` and `Content-Type: application/json` for POST endpoints.
- Supported endpoints include:
  - `GET /v1/models`
  - `POST /v1/chat/completions`
  - `POST /v1/embeddings`
  - `POST /v1/moderations`
  - `POST /v1/images/generations`
  - `POST /v1/audio/transcriptions`, `POST /v1/audio/translations`, `POST /v1/audio/speech`
  - `POST /v1/responses`
  - `/v1/files` CRUD
  - `/v1/batches`
- See `../reference/requests/README.md` and `Code_Examples/curl` for copy/paste snippets of every endpoint.

## Reading Responses

- `X-Budget-Remaining`, `X-Budget-Reset`, `X-Budget-Limit` - track how much budget is left for your key and when it refreshes.
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` - use for adaptive throttling/backoff.
- `X-Provider-Request-ID`, `X-Request-ID` - capture for debugging; share them when opening tickets.
- Errors follow OpenAI's schema (`{"error": {"type": "...", "code": "...", "message": "..."}}`) so SDK error handling works unchanged.

## Personal use quickstart

If you only need personal experimentation, follow the dedicated walkthrough in `docs/personal/guide.md` for personal keys, usage dashboards, and key rotation.

## Staying Within Budgets and Limits

- Throttle long-running jobs using the rate-limit headers or the `Retry-After` value on `429` responses.
- Monitor spend in the **Usage** tab or ask your tenant owner for delegated dashboards.
- Pause workloads proactively when alert emails or webhooks warn that the budget threshold is near exhaustion.

![TODO: User portal usage dashboard](../assets/screenshots/user-usage-dashboard.png)

## Troubleshooting Quick Reference

| Error | What it means | What to do |
| --- | --- | --- |
| `401 unauthorized` | Key revoked or expired | Request a new key from the tenant owner |
| `402 budget_exceeded` | Key/tenant budget spent | Wait for refresh or request a higher budget |
| `404 model_not_found` | Alias not attached to your tenant | Ask the tenant owner to enable the model |
| `429 rate_limit_exceeded` | RPM/TPM reached | Honour the `Retry-After` header value, then add exponential backoff; the gateway itself retries upstream 429s with jitter |
| `provider_unavailable` | Upstream provider outage | Retry later or use a different model alias |

Escalate to your tenant owner with the request ID and timestamp if the issue persists. They can coordinate with platform administrators for deeper investigation.

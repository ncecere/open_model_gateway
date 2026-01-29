---
title: usage flows
description: Export spend and trigger billing webhooks
---
[**Open Model Gateway**](/) exposes admin and tenant APIs for exports plus signed billing webhooks.

---

#### Authenticate requests
Admins send bearer tokens issued by `/admin/auth/login` or Admin API tokens, while tenant users rely on `/user/auth/login` or refreshed user tokens.
Always pass `Authorization: Bearer <token>` to every `/admin/*` or `/user/*` usage route.

---

#### Create exports
| Endpoint | Scope | Body highlights |
| --- | --- | --- |
| `POST /admin/usage-exports` | Admin | Optional `tenant_ids[]`, `tenant_id`, `period` (`30d` default), `start` + `end`, `granularity` (`daily`/`monthly`), `format` (`csv`/`parquet`), and `timezone`. |
| `POST /user/usage-exports` | Tenant | `tenant_id` optional (personal tenant implied), same period/granularity/format/timezone fields, 90-day max window. |

```json
{
  "tenant_ids": ["uuid-1"],
  "period": "30d",
  "granularity": "daily",
  "format": "csv",
  "timezone": "America/New_York"
}
```

---

#### Manage exports
List exports with `GET /admin/usage-exports?limit=25&offset=0` or `GET /user/usage-exports?...` to inspect recent jobs.
Poll `GET .../{id}` until `status=ready`, then download via `GET .../{id}/content` or the admin `download_url` link.

---

#### Inspect export rows
Each CSV/Parquet row contains `bucket_start`, tenant identifiers, `model_alias`, `provider`, request counts, individual token buckets, and `cost_cents` plus `cost_usd_micros`.
Use `/admin/tenants` or `/user/tenants` to look up tenant IDs before scoping exports.

---

#### Configure billing webhooks
Create hooks through `POST /admin/billing-webhooks` or `POST /user/billing-webhooks`; user calls require `tenant_id` and inherit tenant limits.

```json
{
  "tenant_id": "uuid-1",
  "name": "NetSuite",
  "url": "https://billing.example.com/usage",
  "secret": "super-secret",
  "enabled": true
}
```

Update, delete, or filter hooks using the matching `PUT`/`DELETE` endpoints and `?tenant_id=` query parameter if needed.

---

#### Dispatch summaries
Trigger a payload with `POST /admin/billing-webhooks/{id}/dispatch` (or `/user/...`) by passing either `period` (`2025-01`) or `start` + `end` plus an optional `timezone`.
Registered webhooks include `X-OMG-Signature-Version: v1`, `X-OMG-Timestamp: <RFC 3339>`, and `X-OMG-Signature: sha256=<hex>` headers where the signature equals `HMAC_SHA256(secret, body)`.
The same HMAC signing applies to **budget alert webhooks** and **provider alert webhooks** when `budgets.alert.webhook.secret` is configured.
Failed deliveries are retried up to `max_retries` times (default 3) with exponential backoff and 0-25% random jitter. Each attempt is logged with URL, status code, latency, and attempt number.

---

#### Monitor events
List deliveries via `GET /admin/billing-webhooks/{id}/events` or `GET /user/.../events`, then retry failed admin events with `POST /admin/billing-webhooks/events/{event_id}/retry`.
Retries obey the global webhook timeout and `max_retries` set under `budgets.alert.webhook.*`.

---

#### Research
- Lifted payloads, field names, and flow details from the original usage runtime guide.
- Confirmed webhook signature headers and retry settings against the limiter + alerting releases listed in `AGENTS.md`.

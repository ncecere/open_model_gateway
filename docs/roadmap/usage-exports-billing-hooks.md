# Usage Exports & Billing Hooks

## Summary
Finance and RevOps teams need self-serve access to normalized token/spend data without touching the production database. We will expose tenant-scoped exports (CSV/Parquet) plus optional webhook pushes so downstream billing systems (Chargebee, NetSuite, Snowflake, etc.) can ingest Open Model Gateway usage automatically.

## Implementation Overview

### Export Delivery
1. **Export Service** – new `internal/services/exports` package that queries the aggregated usage tables and writes CSV/Parquet blobs to the existing Files backend (purpose=`usage_export`).
2. **API Surface** –
   - `POST /user/usage-exports` and `/admin/usage-exports` for scoped vs. global exports.
   - `GET /user/usage-exports/:id` to poll status and download the generated file.
   - Query params: tenant IDs, date range (max 90 days), granularity (daily/monthly), currency override.
3. **Scheduling** – optional cron that auto-generates monthly exports per tenant and emails a signed download link.

### Billing Webhooks
1. Extend the notification service with a `billing_summary` webhook type. Payload: `{tenant_id, period_start, period_end, spend, token_counts, top_models}`.
2. Allow tenants/admins to configure webhook endpoints + HMAC secrets via Settings.
3. Support manual re-delivery and failure tracking like the existing budget alerts.

### Storage & Retention
- Reuse Files TTL for exports (default 30 days) and allow admins to push to S3/GCS via the existing blob abstraction for long-term archiving.

## Example Workflows
- **Monthly close**: Finance calls `POST /admin/usage-exports` with `period=2025-12` and downloads a CSV summarizing each tenant’s spend/tokens.
- **Tenant self-service**: A customer admin hits `POST /user/usage-exports` scoped to their tenant, feeds the CSV into their internal chargeback tooling.
- **Automated billing**: A webhook posts to NetSuite whenever a tenant exceeds $500 spend in the month; the downstream job issues an invoice automatically.

## Components & Touchpoints
- **Backend**: new export service, controller handlers, webhook payload structs, background job (cron or queue).
- **Database**: materialized views or temp tables to speed up long-range exports; `usage_exports` table to track status/metadata.
- **Files service**: new purpose + TTL defaults.
- **Frontend**: admin + user portal panels showing export history, download buttons, and webhook configuration.
- **Docs/Samples**: update `docs/runtime/usage.md` (TBD) and add curl examples for triggering exports/webhooks.

## Risks & Considerations
- Large exports may exceed request timeouts → perform work asynchronously and poll status.
- Ensure tenant scoping is enforced so users cannot export other tenants’ data.
- Hook payloads should be idempotent and signed (HMAC) to avoid duplicate billing entries.

## Next Steps
1. Design DB schema (`usage_exports`, `billing_webhooks`).
2. Implement async export job + polling APIs.
3. Add webhook dispatcher + retries.
4. Build portal UI and documentation.

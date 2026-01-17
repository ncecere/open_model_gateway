# Usage Exports & Billing Hooks

> Status: Core APIs and background processing are implemented. UI surfaces and scheduled exports are still pending.

## Summary
Finance and RevOps teams need self-serve access to normalized token/spend data without touching the production database. We will expose tenant-scoped exports (CSV/Parquet) plus optional webhook pushes so downstream billing systems (Chargebee, NetSuite, Snowflake, etc.) can ingest Open Model Gateway usage automatically.

## Implementation Overview

### Export Delivery
1. **Export Service** - new `internal/services/exports` package that queries the aggregated usage tables and writes CSV/Parquet blobs to the existing Files backend (purpose=`usage_export`).
2. **API Surface** -
3. **Scheduling** - optional cron that auto-generates monthly exports per tenant and emails a signed download link.


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
- **Frontend**: admin + user portal panels showing export history, download buttons, and webhook configuration (pending).
- **Docs/Samples**: see `docs/admin/runtime/usage.md` for current payloads and curl examples.

## Risks & Considerations
- Large exports may exceed request timeouts; perform work asynchronously and poll status.
- Ensure tenant scoping is enforced so users cannot export other tenants’ data.
- Hook payloads should be idempotent and signed (HMAC) to avoid duplicate billing entries.

## Next Steps
1. Build portal UI for exports/webhooks.
2. Add scheduled monthly exports.
3. Expand webhook payloads or add per-model billing breakdowns if required by finance.

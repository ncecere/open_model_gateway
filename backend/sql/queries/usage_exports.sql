-- name: CreateUsageExport :one
INSERT INTO usage_exports (
    scope,
    status,
    format,
    granularity,
    timezone,
    period_start,
    period_end,
    tenant_ids,
    requested_by,
    file_tenant_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetUsageExport :one
SELECT *
FROM usage_exports
WHERE id = $1;

-- name: ListUsageExportsAdmin :many
SELECT *
FROM usage_exports
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUsageExportsByRequester :many
SELECT *
FROM usage_exports
WHERE requested_by = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ClaimUsageExport :one
UPDATE usage_exports
SET status = 'processing',
    started_at = NOW(),
    updated_at = NOW()
WHERE id = (
    SELECT id
    FROM usage_exports
    WHERE status = 'queued'
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: CompleteUsageExport :one
UPDATE usage_exports
SET status = 'ready',
    file_id = $2,
    row_count = $3,
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailUsageExport :one
UPDATE usage_exports
SET status = 'failed',
    error = $2,
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ExportUsageRows :many
SELECT
    timezone($5::text, date_trunc($4::text, u.ts AT TIME ZONE $5::text))::timestamptz AS bucket,
    u.tenant_id,
    t.name AS tenant_name,
    u.model_alias,
    u.provider,
    COALESCE(SUM(u.requests), 0)::bigint AS requests,
    COALESCE(SUM(u.input_tokens), 0)::bigint AS input_tokens,
    COALESCE(SUM(u.output_tokens), 0)::bigint AS output_tokens,
    COALESCE(SUM(u.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(u.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.ts >= $1
  AND u.ts < $2
  AND ($3::uuid[] IS NULL OR u.tenant_id = ANY($3::uuid[]))
GROUP BY bucket, u.tenant_id, t.name, u.model_alias, u.provider
ORDER BY bucket, t.name, u.model_alias;

-- name: AggregateUsageByModelForTenant :many
SELECT
    model_alias,
    provider,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE tenant_id = $1
  AND ts >= $2
  AND ts < $3
GROUP BY model_alias, provider
ORDER BY cost_usd_micros DESC, requests DESC
LIMIT $4;

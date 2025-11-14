-- name: InsertUsageRecord :one
INSERT INTO usage_records (
    tenant_id,
    api_key_id,
    ts,
    model_alias,
    provider,
    input_tokens,
    output_tokens,
    requests,
    cost_cents,
    cost_usd_micros
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: SumUsageForTenant :one
SELECT
    COALESCE(SUM(input_tokens), 0) AS total_input_tokens,
    COALESCE(SUM(output_tokens), 0) AS total_output_tokens,
    COALESCE(SUM(requests), 0) AS total_requests,
    COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records
WHERE tenant_id = $1
  AND ts >= $2
  AND ts < $3;

-- name: SumUsage :one
SELECT
    COALESCE(SUM(requests), 0)::bigint AS total_requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records
WHERE ($1::uuid IS NULL OR tenant_id = $1)
  AND ts >= $2
  AND ts < $3;

-- name: AggregateUsageDaily :many
SELECT
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE ($1::uuid IS NULL OR tenant_id = $1)
  AND ts >= $2
  AND ts < $3
GROUP BY day
ORDER BY day;

-- name: ListUserOwnedTenants :many
SELECT DISTINCT
    tm.tenant_id,
    t.name,
    tm.role,
    t.status
FROM tenant_memberships tm
JOIN tenants t ON t.id = tm.tenant_id
WHERE tm.user_id = $1
  AND t.kind <> 'personal'
  AND EXISTS (
    SELECT 1
    FROM api_keys k
    WHERE k.owner_user_id = $1
      AND k.tenant_id = tm.tenant_id
  )
ORDER BY t.name;

-- name: SumUsageForUserTenant :one
SELECT
    COALESCE(SUM(u.requests), 0)::bigint AS total_requests,
    COALESCE(SUM(u.input_tokens + u.output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(u.cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(u.cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records u
JOIN api_keys k ON u.api_key_id = k.id
WHERE k.owner_user_id = $1
  AND u.tenant_id = $2
  AND u.ts >= $3
  AND u.ts < $4;

-- name: AggregateUsageDailyForUserTenant :many
SELECT
    timezone($5::text, date_trunc('day', u.ts AT TIME ZONE $5::text))::timestamptz AS day,
    COALESCE(SUM(u.requests), 0)::bigint AS requests,
    COALESCE(SUM(u.input_tokens + u.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(u.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(u.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records u
JOIN api_keys k ON u.api_key_id = k.id
WHERE k.owner_user_id = $1
  AND u.tenant_id = $2
  AND u.ts >= $3
  AND u.ts < $4
GROUP BY day
ORDER BY day;

-- name: ListUsageRecords :many
SELECT *
FROM usage_records
WHERE tenant_id = $1
  AND ts >= $2
  AND ts < $3
ORDER BY ts DESC
LIMIT $4 OFFSET $5;

-- name: AggregateUsageByTenant :many
SELECT
    u.tenant_id,
    t.name,
    COALESCE(SUM(u.requests), 0)::bigint AS requests,
    COALESCE(SUM(u.input_tokens + u.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(u.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(u.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.ts >= $1
  AND u.ts < $2
GROUP BY u.tenant_id, t.name
ORDER BY cost_cents DESC, requests DESC
LIMIT $3;

-- name: AggregateUsageByModel :many
SELECT
    model_alias,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE ts >= $1
  AND ts < $2
GROUP BY model_alias
ORDER BY cost_cents DESC, requests DESC
LIMIT $3;

-- name: AggregateUsageByUser :many
SELECT
    k.owner_user_id AS user_id,
    u.email,
    u.name,
    COALESCE(SUM(r.requests), 0)::bigint AS requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records r
JOIN api_keys k ON r.api_key_id = k.id
JOIN users u ON u.id = k.owner_user_id
WHERE r.ts >= $1
  AND r.ts < $2
GROUP BY k.owner_user_id, u.email, u.name
ORDER BY cost_cents DESC, requests DESC
LIMIT $3;

-- name: AggregateTenantUsageDaily :many
SELECT
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE tenant_id = $1
  AND ts >= $2
  AND ts < $3
GROUP BY day
ORDER BY day;

-- name: AggregateModelUsageDaily :many
SELECT
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE model_alias = $1
  AND ts >= $2
  AND ts < $3
GROUP BY day
ORDER BY day;

-- name: AggregateUsageDailyByUsers :many
SELECT
    k.owner_user_id AS user_id,
    timezone($4::text, date_trunc('day', r.ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(r.requests), 0)::bigint AS requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records r
JOIN api_keys k ON r.api_key_id = k.id
WHERE k.owner_user_id = ANY($1::uuid[])
  AND r.ts >= $2
  AND r.ts < $3
GROUP BY k.owner_user_id, day
ORDER BY k.owner_user_id, day;

-- name: SumUsageForAPIKey :one
SELECT
    COALESCE(SUM(requests), 0)::bigint AS total_requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records
WHERE api_key_id = $1
  AND ts >= $2
  AND ts < $3;

-- name: AggregateAPIKeyUsageDaily :many
SELECT
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE api_key_id = $1
  AND ts >= $2
  AND ts < $3
GROUP BY day
ORDER BY day;

-- name: AggregateUsageDailyByTenants :many
SELECT
    tenant_id,
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE tenant_id = ANY($1::uuid[])
  AND ts >= $2
  AND ts < $3
GROUP BY tenant_id, day
ORDER BY tenant_id, day;

-- name: SumUsageByUsers :many
SELECT
    k.owner_user_id AS user_id,
    COALESCE(SUM(r.requests), 0)::bigint AS total_requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records r
JOIN api_keys k ON r.api_key_id = k.id
WHERE k.owner_user_id = ANY($1::uuid[])
  AND r.ts >= $2
  AND r.ts < $3
GROUP BY k.owner_user_id;

-- name: SumUsageByTenants :many
SELECT
    tenant_id,
    COALESCE(SUM(requests), 0)::bigint AS total_requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records
WHERE tenant_id = ANY($1::uuid[])
  AND ts >= $2
  AND ts < $3
GROUP BY tenant_id;

-- name: AggregateUsageDailyByModels :many
SELECT
    model_alias,
    timezone($4::text, date_trunc('day', ts AT TIME ZONE $4::text))::timestamptz AS day,
    COALESCE(SUM(requests), 0)::bigint AS requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records
WHERE model_alias = ANY($1::text[])
  AND ts >= $2
  AND ts < $3
  AND (cardinality($5::uuid[]) = 0 OR tenant_id = ANY($5::uuid[]))
GROUP BY model_alias, day
ORDER BY model_alias, day;

-- name: AggregateTenantUsageDailyByAPIKeys :many
SELECT
    timezone($4::text, date_trunc('day', r.ts AT TIME ZONE $4::text))::timestamptz AS day,
    r.api_key_id,
    COALESCE(k.name, '') AS api_key_name,
    COALESCE(k.prefix, '') AS api_key_prefix,
    COALESCE(SUM(r.requests), 0)::bigint AS requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records r
LEFT JOIN api_keys k ON k.id = r.api_key_id
WHERE r.tenant_id = $1
  AND r.ts >= $2
  AND r.ts < $3
GROUP BY day, r.api_key_id, k.name, k.prefix
ORDER BY day, api_key_name;

-- name: AggregateUserUsageDailyByTenants :many
SELECT
    timezone($4::text, date_trunc('day', r.ts AT TIME ZONE $4::text))::timestamptz AS day,
    r.tenant_id,
    COALESCE(t.name, '') AS tenant_name,
    COALESCE(SUM(r.requests), 0)::bigint AS requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records r
JOIN api_keys k ON k.id = r.api_key_id
LEFT JOIN tenants t ON t.id = r.tenant_id
WHERE k.owner_user_id = $1
  AND r.ts >= $2
  AND r.ts < $3
GROUP BY day, r.tenant_id, t.name
ORDER BY day, tenant_name;

-- name: AggregateModelUsageDailyByTenants :many
SELECT
    timezone($4::text, date_trunc('day', r.ts AT TIME ZONE $4::text))::timestamptz AS day,
    r.tenant_id,
    COALESCE(t.name, '') AS tenant_name,
    COALESCE(SUM(r.requests), 0)::bigint AS requests,
    COALESCE(SUM(r.input_tokens + r.output_tokens), 0)::bigint AS tokens,
    COALESCE(SUM(r.cost_cents), 0)::bigint AS cost_cents,
    COALESCE(SUM(r.cost_usd_micros), 0)::bigint AS cost_usd_micros
FROM usage_records r
LEFT JOIN tenants t ON t.id = r.tenant_id
WHERE r.model_alias = $1
  AND r.ts >= $2
  AND r.ts < $3
GROUP BY day, r.tenant_id, t.name
ORDER BY day, tenant_name;

-- name: SumUsageByModels :many
SELECT
    model_alias,
    COALESCE(SUM(requests), 0)::bigint AS total_requests,
    COALESCE(SUM(input_tokens + output_tokens), 0)::bigint AS total_tokens,
    COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
    COALESCE(SUM(cost_usd_micros), 0)::bigint AS total_cost_usd_micros
FROM usage_records
WHERE model_alias = ANY($1::text[])
  AND ts >= $2
  AND ts < $3
  AND (cardinality($4::uuid[]) = 0 OR tenant_id = ANY($4::uuid[]))
GROUP BY model_alias;

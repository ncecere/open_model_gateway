-- name: ListTenantRateLimits :many
SELECT *
FROM tenant_rate_limits;

-- name: GetTenantRateLimit :one
SELECT *
FROM tenant_rate_limits
WHERE tenant_id = $1;

-- name: UpsertTenantRateLimit :one
INSERT INTO tenant_rate_limits (
    tenant_id,
    requests_per_minute,
    tokens_per_minute,
    parallel_requests
) VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id) DO UPDATE
SET requests_per_minute = EXCLUDED.requests_per_minute,
    tokens_per_minute = EXCLUDED.tokens_per_minute,
    parallel_requests = EXCLUDED.parallel_requests,
    updated_at = NOW()
RETURNING *;

-- name: DeleteTenantRateLimit :exec
DELETE FROM tenant_rate_limits
WHERE tenant_id = $1;

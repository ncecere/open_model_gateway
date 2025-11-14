-- name: GetRateLimitDefaults :one
SELECT *
FROM rate_limit_defaults
WHERE id IS TRUE;

-- name: UpsertRateLimitDefaults :one
INSERT INTO rate_limit_defaults (
    id,
    requests_per_minute,
    tokens_per_minute,
    parallel_requests_key,
    parallel_requests_tenant
)
VALUES (TRUE, $1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
    SET requests_per_minute = EXCLUDED.requests_per_minute,
        tokens_per_minute = EXCLUDED.tokens_per_minute,
        parallel_requests_key = EXCLUDED.parallel_requests_key,
        parallel_requests_tenant = EXCLUDED.parallel_requests_tenant,
        updated_at = NOW()
RETURNING *;

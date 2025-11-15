-- name: ListAPIKeyRateLimits :many
SELECT ak.prefix AS prefix,
       r.api_key_id,
       r.requests_per_minute,
       r.tokens_per_minute,
       r.parallel_requests
FROM api_key_rate_limits r
JOIN api_keys ak ON ak.id = r.api_key_id;

-- name: GetAPIKeyRateLimit :one
SELECT api_key_id,
       requests_per_minute,
       tokens_per_minute,
       parallel_requests
FROM api_key_rate_limits
WHERE api_key_id = $1;

-- name: UpsertAPIKeyRateLimit :one
INSERT INTO api_key_rate_limits (
    api_key_id,
    requests_per_minute,
    tokens_per_minute,
    parallel_requests
) VALUES ($1, $2, $3, $4)
ON CONFLICT (api_key_id) DO UPDATE
SET requests_per_minute = EXCLUDED.requests_per_minute,
    tokens_per_minute = EXCLUDED.tokens_per_minute,
    parallel_requests = EXCLUDED.parallel_requests,
    updated_at = NOW()
RETURNING api_key_id,
          requests_per_minute,
          tokens_per_minute,
          parallel_requests;

-- name: DeleteAPIKeyRateLimit :execrows
DELETE FROM api_key_rate_limits
WHERE api_key_id = $1;

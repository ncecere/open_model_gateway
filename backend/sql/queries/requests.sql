-- name: InsertRequestRecord :one
INSERT INTO requests (
    tenant_id,
    api_key_id,
    ts,
    model_alias,
    provider,
    latency_ms,
    status,
    error_code,
    input_tokens,
    output_tokens,
    cost_cents,
    cost_usd_micros,
    idempotency_key,
    trace_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetRequestByIdempotencyKey :one
SELECT *
FROM requests
WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: ListRequests :many
SELECT *
FROM requests
WHERE tenant_id = $1
  AND ts >= $2
  AND ts < $3
ORDER BY ts DESC
LIMIT $4 OFFSET $5;

-- name: ListRecentRequestsByAPIKeys :many
SELECT *
FROM requests
WHERE api_key_id = ANY($1::uuid[])
ORDER BY ts DESC
LIMIT $2;

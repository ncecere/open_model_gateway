-- name: InsertResponseCache :exec
INSERT INTO responses_cache (id, tenant_id, model, input, output, instructions, metadata, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetResponseCache :one
SELECT id, tenant_id, model, input, output, instructions, metadata, created_at
FROM responses_cache
WHERE id = $1 AND expires_at > now();

-- name: DeleteExpiredResponseCache :exec
DELETE FROM responses_cache WHERE expires_at <= now();

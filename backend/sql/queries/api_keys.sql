-- name: CreateAPIKey :one
INSERT INTO api_keys (
    tenant_id,
    prefix,
    secret_hash,
    name,
    scopes_json,
    quota_json,
    kind,
    owner_user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAPIKeyByPrefix :one
SELECT *
FROM api_keys
WHERE prefix = $1;

-- name: GetAPIKeyByID :one
SELECT *
FROM api_keys
WHERE id = $1;

-- name: ListAPIKeysByTenant :many
SELECT *
FROM api_keys
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: ListPersonalAPIKeysByUser :many
SELECT *
FROM api_keys
WHERE owner_user_id = $1
ORDER BY created_at DESC;

-- name: ListAPIKeysByOwnerAndTenant :many
SELECT *
FROM api_keys
WHERE owner_user_id = $1
  AND tenant_id = $2
ORDER BY created_at DESC;

-- name: ListAPIKeysByIDs :many
SELECT *
FROM api_keys
WHERE id = ANY($1::uuid[]);

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL
RETURNING *;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1;

-- name: UpdateAPIKeyTenant :one
UPDATE api_keys
SET tenant_id = $2
WHERE id = $1
RETURNING *;

-- name: ListAllAPIKeys :many
SELECT
    k.id,
    k.tenant_id,
    t.name AS tenant_name,
    t.kind AS tenant_kind,
    k.prefix,
    k.name,
    k.scopes_json,
    k.quota_json,
    k.kind,
    k.owner_user_id,
    k.created_at,
    k.revoked_at,
    k.last_used_at,
    u.email AS owner_email,
    u.name AS owner_name
FROM api_keys k
JOIN tenants t ON t.id = k.tenant_id
LEFT JOIN users u ON u.id = k.owner_user_id
ORDER BY k.created_at DESC;

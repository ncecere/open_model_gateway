-- name: CreateAdminAPIKey :one
INSERT INTO admin_api_keys (
    name,
    prefix,
    secret_hash,
    scope,
    owner_user_id,
    created_by_user_id,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAdminAPIKeyByPrefix :one
SELECT *
FROM admin_api_keys
WHERE prefix = $1;

-- name: GetAdminAPIKeyByID :one
SELECT *
FROM admin_api_keys
WHERE id = $1;

-- name: ListAdminAPIKeys :many
SELECT
    k.id,
    k.name,
    k.prefix,
    k.secret_hash,
    k.scope,
    k.owner_user_id,
    k.created_by_user_id,
    k.expires_at,
    k.revoked_at,
    k.last_used_at,
    k.created_at,
    owner.email AS owner_email,
    owner.name AS owner_name,
    creator.email AS creator_email,
    creator.name AS creator_name
FROM admin_api_keys k
LEFT JOIN users owner ON owner.id = k.owner_user_id
JOIN users creator ON creator.id = k.created_by_user_id
ORDER BY k.created_at DESC;

-- name: ListAdminAPIKeysByOwner :many
SELECT
    k.id,
    k.name,
    k.prefix,
    k.secret_hash,
    k.scope,
    k.owner_user_id,
    k.created_by_user_id,
    k.expires_at,
    k.revoked_at,
    k.last_used_at,
    k.created_at,
    owner.email AS owner_email,
    owner.name AS owner_name,
    creator.email AS creator_email,
    creator.name AS creator_name
FROM admin_api_keys k
LEFT JOIN users owner ON owner.id = k.owner_user_id
JOIN users creator ON creator.id = k.created_by_user_id
WHERE k.owner_user_id = $1
ORDER BY k.created_at DESC;

-- name: RevokeAdminAPIKey :one
UPDATE admin_api_keys
SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL
RETURNING *;

-- name: UpdateAdminAPIKeyLastUsed :exec
UPDATE admin_api_keys
SET last_used_at = NOW()
WHERE id = $1;

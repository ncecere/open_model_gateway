-- name: CreateUser :one
INSERT INTO users (email, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUsersByIDs :many
SELECT id, email, name, theme_preference
FROM users
WHERE id = ANY($1::uuid[]);

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: SetUserSuperAdmin :exec
UPDATE users
SET is_super_admin = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserPersonalTenant :one
UPDATE users
SET personal_tenant_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users
SET
    name = COALESCE(sqlc.narg(name), name),
    theme_preference = COALESCE(sqlc.narg(theme_preference), theme_preference),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

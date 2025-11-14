-- name: CreateTenant :one
INSERT INTO tenants (name, status, kind)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTenantByID :one
SELECT *
FROM tenants
WHERE id = $1;

-- name: GetTenantByName :one
SELECT *
FROM tenants
WHERE name = $1;

-- name: ListTenants :many
SELECT *
FROM tenants
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTenantsByIDs :many
SELECT id, name, status, kind
FROM tenants
WHERE id = ANY($1::uuid[]);

-- name: ListPersonalTenantIDs :many
SELECT id
FROM tenants
WHERE kind = 'personal';

-- name: ListPersonalTenants :many
SELECT
    t.id,
    t.name,
    t.status,
    t.created_at,
    u.id AS user_id,
    u.email AS user_email,
    u.name AS user_name,
    u.created_at AS user_created_at,
    COALESCE(mc.membership_count, 0) AS membership_count
FROM tenants t
JOIN users u ON u.personal_tenant_id = t.id
LEFT JOIN (
    SELECT user_id, COUNT(DISTINCT tenant_id) AS membership_count
    FROM tenant_memberships
    GROUP BY user_id
) mc ON mc.user_id = u.id
WHERE t.kind = 'personal'
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateTenantStatus :one
UPDATE tenants
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdateTenantName :one
UPDATE tenants
SET name = $2
WHERE id = $1
RETURNING *;

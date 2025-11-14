-- name: AddTenantMembership :one
INSERT INTO tenant_memberships (tenant_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateTenantMembershipRole :one
UPDATE tenant_memberships
SET role = $3
WHERE tenant_id = $1 AND user_id = $2
RETURNING *;

-- name: RemoveTenantMembership :exec
DELETE FROM tenant_memberships
WHERE tenant_id = $1 AND user_id = $2;

-- name: ListTenantMembers :many
SELECT
    tm.id,
    tm.tenant_id,
    tm.user_id,
    tm.role,
    tm.created_at,
    u.email AS user_email,
    u.name AS user_name,
    u.created_at AS user_created_at
FROM tenant_memberships tm
JOIN users u ON u.id = tm.user_id
WHERE tm.tenant_id = $1
ORDER BY u.email;

-- name: ListUserTenants :many
SELECT
    tm.id,
    tm.tenant_id,
    tm.user_id,
    tm.role,
    tm.created_at,
    t.name AS tenant_name,
    t.status AS tenant_status,
    t.created_at AS tenant_created_at
FROM tenant_memberships tm
JOIN tenants t ON t.id = tm.tenant_id
WHERE tm.user_id = $1
ORDER BY t.name;

-- name: GetTenantMembership :one
SELECT *
FROM tenant_memberships
WHERE tenant_id = $1 AND user_id = $2;

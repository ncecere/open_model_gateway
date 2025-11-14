-- name: InsertAuditLog :one
INSERT INTO admin_audit_logs (
    user_id,
    action,
    resource_type,
    resource_id,
    metadata
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAuditLogs :many
SELECT id, user_id, action, resource_type, resource_id, metadata, created_at
FROM admin_audit_logs
WHERE (
    sqlc.narg(user_id_filter)::uuid IS NULL
    OR user_id = sqlc.narg(user_id_filter)::uuid
)
  AND (
    sqlc.narg(action_filter)::text IS NULL
    OR action = sqlc.narg(action_filter)::text
)
  AND (
    sqlc.narg(resource_filter)::text IS NULL
    OR resource_type = sqlc.narg(resource_filter)::text
)
ORDER BY created_at DESC
LIMIT sqlc.arg(list_limit) OFFSET sqlc.arg(list_offset);

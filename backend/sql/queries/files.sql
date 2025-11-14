-- name: CreateFile :one
INSERT INTO files (
    tenant_id,
    filename,
    purpose,
    content_type,
    bytes,
    storage_backend,
    storage_key,
    checksum,
    encrypted,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetFile :one
SELECT *
FROM files
WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: DeleteFile :exec
UPDATE files
SET deleted_at = NOW()
WHERE tenant_id = $1 AND id = $2;

-- name: ListFiles :many
SELECT *
FROM files
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListFilesAdmin :many
SELECT f.*, t.name AS tenant_name, COUNT(*) OVER() AS total_count
FROM files f
JOIN tenants t ON t.id = f.tenant_id
WHERE (sqlc.narg(tenant_id)::uuid IS NULL OR f.tenant_id = sqlc.narg(tenant_id))
  AND (sqlc.narg(purpose)::text IS NULL OR f.purpose = sqlc.narg(purpose)::text)
  AND (
    sqlc.narg(search)::text IS NULL
    OR f.filename ILIKE '%' || sqlc.narg(search)::text || '%'
    OR t.name ILIKE '%' || sqlc.narg(search)::text || '%'
    OR f.id::text ILIKE '%' || sqlc.narg(search)::text || '%'
  )
  AND (
    sqlc.arg(state)::text = 'all'
    OR (sqlc.arg(state)::text = 'deleted' AND f.deleted_at IS NOT NULL)
    OR (sqlc.arg(state)::text = 'active' AND f.deleted_at IS NULL)
  )
ORDER BY f.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetFileByID :one
SELECT *
FROM files
WHERE id = $1;

-- name: ListExpiredFiles :many
SELECT *
FROM files
WHERE deleted_at IS NULL AND expires_at <= $1
LIMIT $2;

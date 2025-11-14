-- name: UpsertRoute :one
INSERT INTO routes (
    tenant_id,
    alias,
    provider,
    provider_model,
    weight,
    params_json,
    region,
    sticky_key,
    enabled
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (tenant_id, alias, provider, provider_model)
DO UPDATE SET
    weight = EXCLUDED.weight,
    params_json = EXCLUDED.params_json,
    region = EXCLUDED.region,
    sticky_key = EXCLUDED.sticky_key,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
RETURNING *;

-- name: ListRoutesByTenant :many
SELECT *
FROM routes
WHERE tenant_id = $1
ORDER BY alias, created_at DESC;

-- name: ListRoutesByAlias :many
SELECT *
FROM routes
WHERE tenant_id = $1 AND alias = $2 AND enabled = true
ORDER BY weight DESC, created_at DESC;

-- name: DeleteRoute :exec
DELETE FROM routes
WHERE id = $1;

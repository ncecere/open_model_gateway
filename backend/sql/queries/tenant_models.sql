-- name: ListTenantModels :many
SELECT alias
FROM tenant_models
WHERE tenant_id = $1
ORDER BY alias;

-- name: ListAllTenantModels :many
SELECT tenant_id, alias
FROM tenant_models;

-- name: InsertTenantModel :exec
INSERT INTO tenant_models (tenant_id, alias)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteTenantModels :exec
DELETE FROM tenant_models
WHERE tenant_id = $1;

-- name: DeleteTenantModel :exec
DELETE FROM tenant_models
WHERE tenant_id = $1 AND alias = $2;

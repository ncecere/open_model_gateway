-- name: GetTenantGuardrailPolicy :one
SELECT *
FROM guardrail_policies
WHERE tenant_id = $1;

-- name: UpsertTenantGuardrailPolicy :one
INSERT INTO guardrail_policies (tenant_id, config_json)
VALUES ($1, $2)
ON CONFLICT (tenant_id)
DO UPDATE SET config_json = EXCLUDED.config_json, updated_at = NOW()
RETURNING *;

-- name: DeleteTenantGuardrailPolicy :exec
DELETE FROM guardrail_policies
WHERE tenant_id = $1;

-- name: GetAPIKeyGuardrailPolicy :one
SELECT *
FROM guardrail_policies
WHERE api_key_id = $1;

-- name: UpsertAPIKeyGuardrailPolicy :one
INSERT INTO guardrail_policies (api_key_id, config_json)
VALUES ($1, $2)
ON CONFLICT (api_key_id)
DO UPDATE SET config_json = EXCLUDED.config_json, updated_at = NOW()
RETURNING *;

-- name: DeleteAPIKeyGuardrailPolicy :exec
DELETE FROM guardrail_policies
WHERE api_key_id = $1;

-- name: InsertGuardrailEvent :one
INSERT INTO guardrail_events (
    tenant_id,
    api_key_id,
    model_alias,
    action,
    category,
    details
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListGuardrailEvents :many
SELECT
    ge.*,
    t.name AS tenant_name,
    k.name AS api_key_name,
    COUNT(*) OVER() AS total_rows
FROM guardrail_events ge
LEFT JOIN tenants t ON ge.tenant_id = t.id
LEFT JOIN api_keys k ON ge.api_key_id = k.id
WHERE ($1::uuid IS NULL OR ge.tenant_id = $1)
  AND ($2::uuid IS NULL OR ge.api_key_id = $2)
  AND (($3)::text IS NULL OR $3 = '' OR ge.action = $3)
  AND (($4)::text IS NULL OR $4 = '' OR ge.details->>'stage' = $4)
  AND (($5)::text IS NULL OR $5 = '' OR ge.category = $5)
  AND (($6)::timestamptz IS NULL OR ge.created_at >= $6)
  AND (($7)::timestamptz IS NULL OR ge.created_at <= $7)
ORDER BY ge.created_at DESC
LIMIT $8 OFFSET $9;

-- name: UpsertModelCatalogEntry :one
INSERT INTO model_catalog (
    alias,
    provider,
    provider_model,
    model_type,
    context_window,
    max_output_tokens,
    modalities_json,
    supports_tools,
    price_input,
    price_output,
    currency,
    enabled,
    deployment,
    endpoint,
    api_key,
    api_version,
    region,
    metadata_json,
    weight,
    provider_config_json
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
ON CONFLICT (alias)
DO UPDATE SET
    provider = EXCLUDED.provider,
    provider_model = EXCLUDED.provider_model,
    model_type = EXCLUDED.model_type,
    context_window = EXCLUDED.context_window,
    max_output_tokens = EXCLUDED.max_output_tokens,
    modalities_json = EXCLUDED.modalities_json,
    supports_tools = EXCLUDED.supports_tools,
    price_input = EXCLUDED.price_input,
    price_output = EXCLUDED.price_output,
    currency = EXCLUDED.currency,
    enabled = EXCLUDED.enabled,
    deployment = EXCLUDED.deployment,
    endpoint = EXCLUDED.endpoint,
    api_key = EXCLUDED.api_key,
    api_version = EXCLUDED.api_version,
    region = EXCLUDED.region,
    metadata_json = EXCLUDED.metadata_json,
    weight = EXCLUDED.weight,
    provider_config_json = EXCLUDED.provider_config_json,
    updated_at = NOW()
RETURNING *;

-- name: GetModelByAlias :one
SELECT *
FROM model_catalog
WHERE alias = $1;

-- name: ListModelCatalog :many
SELECT *
FROM model_catalog
ORDER BY alias;

-- name: ListEnabledModels :many
SELECT *
FROM model_catalog
WHERE enabled = true
ORDER BY alias;

-- name: ListModelCatalogByAliases :many
SELECT *
FROM model_catalog
WHERE alias = ANY($1::text[]);

-- name: DeleteModelCatalogEntry :exec
DELETE FROM model_catalog
WHERE alias = $1;

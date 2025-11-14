-- name: ListDefaultModels :many
SELECT alias
FROM default_models
ORDER BY alias;

-- name: UpsertDefaultModel :exec
INSERT INTO default_models (alias)
VALUES ($1)
ON CONFLICT (alias) DO NOTHING;

-- name: DeleteDefaultModel :exec
DELETE FROM default_models
WHERE alias = $1;

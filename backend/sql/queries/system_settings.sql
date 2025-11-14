-- name: GetSystemSetting :one
SELECT key, value, updated_at, updated_by
FROM system_settings
WHERE key = $1;

-- name: UpsertSystemSetting :one
INSERT INTO system_settings (key, value, updated_at, updated_by)
VALUES ($1, $2, now(), $3)
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value, updated_at = now(), updated_by = EXCLUDED.updated_by
RETURNING key, value, updated_at, updated_by;

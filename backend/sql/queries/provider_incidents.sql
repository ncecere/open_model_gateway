-- name: UpsertProviderIncident :one
INSERT INTO provider_incidents (
    provider,
    model_alias,
    incident_type,
    status,
    window_seconds,
    opened_at,
    resolved_at,
    window_started_at,
    window_ended_at,
    sample_error,
    request_count,
    error_count,
    timeout_count,
    latency_p95_ms,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) ON CONFLICT (provider, model_alias, incident_type) WHERE resolved_at IS NULL
DO UPDATE SET
    status = EXCLUDED.status,
    window_seconds = EXCLUDED.window_seconds,
    opened_at = LEAST(provider_incidents.opened_at, EXCLUDED.opened_at),
    resolved_at = EXCLUDED.resolved_at,
    window_started_at = EXCLUDED.window_started_at,
    window_ended_at = EXCLUDED.window_ended_at,
    sample_error = COALESCE(EXCLUDED.sample_error, provider_incidents.sample_error),
    request_count = EXCLUDED.request_count,
    error_count = EXCLUDED.error_count,
    timeout_count = EXCLUDED.timeout_count,
    latency_p95_ms = EXCLUDED.latency_p95_ms,
    metadata = EXCLUDED.metadata
RETURNING *;

-- name: ResolveProviderIncident :one
UPDATE provider_incidents
SET status = 'resolved',
    resolved_at = NOW(),
    metadata = COALESCE($4, metadata)
WHERE provider = $1 AND model_alias = $2 AND incident_type = $3 AND resolved_at IS NULL
RETURNING *;

-- name: PurgeProviderIncidents :exec
DELETE FROM provider_incidents
WHERE opened_at < NOW() - ($1 * INTERVAL '1 day');

-- name: ListProviderIncidents :many
SELECT id, provider, model_alias, incident_type, status, window_seconds, opened_at, resolved_at, window_started_at, window_ended_at, sample_error, request_count, error_count, timeout_count, latency_p95_ms, metadata
FROM provider_incidents
WHERE (sqlc.arg(provider)::text = '' OR provider = sqlc.arg(provider)::text)
  AND (sqlc.arg(model_alias)::text = '' OR model_alias = sqlc.arg(model_alias)::text)
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY opened_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProviderIncidents :one
SELECT count(*) FROM provider_incidents
WHERE (sqlc.arg(provider)::text = '' OR provider = sqlc.arg(provider)::text)
  AND (sqlc.arg(model_alias)::text = '' OR model_alias = sqlc.arg(model_alias)::text)
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text);

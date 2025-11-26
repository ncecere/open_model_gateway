-- +goose Up
CREATE TABLE IF NOT EXISTS provider_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    model_alias TEXT NOT NULL,
    incident_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    window_seconds INT NOT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    window_started_at TIMESTAMPTZ,
    window_ended_at TIMESTAMPTZ,
    sample_error TEXT,
    request_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    timeout_count INT NOT NULL DEFAULT 0,
    latency_p95_ms INT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS provider_incidents_provider_model_idx ON provider_incidents(provider, model_alias);
CREATE UNIQUE INDEX IF NOT EXISTS provider_incidents_open_idx ON provider_incidents(provider, model_alias, incident_type) WHERE resolved_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS provider_incidents_open_idx;
DROP INDEX IF EXISTS provider_incidents_provider_model_idx;
DROP TABLE IF EXISTS provider_incidents;

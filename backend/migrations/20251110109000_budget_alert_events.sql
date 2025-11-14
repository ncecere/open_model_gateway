-- +goose Up
CREATE TABLE IF NOT EXISTS budget_alert_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    level TEXT NOT NULL,
    channels JSONB NOT NULL DEFAULT '{}'::jsonb,
    payload JSONB NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS budget_alert_events_tenant_id_idx ON budget_alert_events(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS budget_alert_events;

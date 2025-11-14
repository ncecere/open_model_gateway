CREATE TABLE IF NOT EXISTS budget_alert_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    level TEXT NOT NULL,
    channels JSONB NOT NULL,
    payload JSONB NOT NULL,
    success BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

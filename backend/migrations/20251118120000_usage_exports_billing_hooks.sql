-- +goose Up
CREATE TABLE IF NOT EXISTS usage_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    format TEXT NOT NULL DEFAULT 'csv',
    granularity TEXT NOT NULL DEFAULT 'daily',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    tenant_ids UUID[],
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    file_id UUID REFERENCES files(id) ON DELETE SET NULL,
    file_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    row_count INTEGER,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS usage_exports_status_idx ON usage_exports(status, created_at DESC);
CREATE INDEX IF NOT EXISTS usage_exports_requested_by_idx ON usage_exports(requested_by);

CREATE TABLE IF NOT EXISTS billing_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT,
    url TEXT NOT NULL,
    secret TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS billing_webhooks_tenant_idx ON billing_webhooks(tenant_id);

CREATE TABLE IF NOT EXISTS billing_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES billing_webhooks(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    status_code INTEGER,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS billing_webhook_events_tenant_idx ON billing_webhook_events(tenant_id);
CREATE INDEX IF NOT EXISTS billing_webhook_events_webhook_idx ON billing_webhook_events(webhook_id);

-- +goose Down
DROP TABLE IF EXISTS billing_webhook_events;
DROP TABLE IF EXISTS billing_webhooks;
DROP TABLE IF EXISTS usage_exports;

-- +goose Up
CREATE TABLE guardrail_policies (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE CASCADE,
    config_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT guardrail_policy_scope_check CHECK (
        (tenant_id IS NOT NULL AND api_key_id IS NULL)
        OR (tenant_id IS NULL AND api_key_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX guardrail_policies_tenant_unique ON guardrail_policies(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX guardrail_policies_api_key_unique ON guardrail_policies(api_key_id) WHERE api_key_id IS NOT NULL;

CREATE TABLE guardrail_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    model_alias TEXT,
    action TEXT NOT NULL,
    category TEXT,
    details JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX guardrail_events_tenant_idx ON guardrail_events(tenant_id, created_at DESC);
CREATE INDEX guardrail_events_api_key_idx ON guardrail_events(api_key_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS guardrail_events;
DROP TABLE IF EXISTS guardrail_policies;

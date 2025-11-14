-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $func$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$func$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS tenant_budget_overrides (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    budget_cents BIGINT NOT NULL CHECK (budget_cents >= 0),
    warning_threshold NUMERIC NOT NULL CHECK (warning_threshold > 0 AND warning_threshold <= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tenant_budget_overrides_updated_at
    BEFORE UPDATE ON tenant_budget_overrides
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS tenant_budget_overrides_updated_at ON tenant_budget_overrides;
DROP TABLE IF EXISTS tenant_budget_overrides;

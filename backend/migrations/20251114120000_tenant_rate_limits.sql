-- +goose Up
CREATE TABLE IF NOT EXISTS tenant_rate_limits (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    requests_per_minute INTEGER NOT NULL CHECK (requests_per_minute >= 0),
    tokens_per_minute INTEGER NOT NULL CHECK (tokens_per_minute >= 0),
    parallel_requests INTEGER NOT NULL CHECK (parallel_requests >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER tenant_rate_limits_updated_at
    BEFORE UPDATE ON tenant_rate_limits
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS tenant_rate_limits_updated_at ON tenant_rate_limits;
DROP TABLE IF EXISTS tenant_rate_limits;

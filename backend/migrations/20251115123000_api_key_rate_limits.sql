-- +goose Up
CREATE TABLE IF NOT EXISTS api_key_rate_limits (
    api_key_id UUID PRIMARY KEY REFERENCES api_keys(id) ON DELETE CASCADE,
    requests_per_minute INTEGER NOT NULL CHECK (requests_per_minute >= 0),
    tokens_per_minute INTEGER NOT NULL CHECK (tokens_per_minute >= 0),
    parallel_requests INTEGER NOT NULL CHECK (parallel_requests >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER api_key_rate_limits_updated_at
    BEFORE UPDATE ON api_key_rate_limits
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS api_key_rate_limits_updated_at ON api_key_rate_limits;
DROP TABLE IF EXISTS api_key_rate_limits;

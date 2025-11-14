-- +goose Up
CREATE TABLE IF NOT EXISTS rate_limit_defaults (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    requests_per_minute INTEGER NOT NULL CHECK (requests_per_minute >= 0),
    tokens_per_minute INTEGER NOT NULL CHECK (tokens_per_minute >= 0),
    parallel_requests_key INTEGER NOT NULL CHECK (parallel_requests_key >= 0),
    parallel_requests_tenant INTEGER NOT NULL CHECK (parallel_requests_tenant >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER rate_limit_defaults_updated_at
    BEFORE UPDATE ON rate_limit_defaults
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS rate_limit_defaults_updated_at ON rate_limit_defaults;
DROP TABLE IF EXISTS rate_limit_defaults;

-- +goose Up
-- Add indexes for frequently queried columns that were missing.

-- usage_records by api_key_id (used in key stats, usage-by-key queries)
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id
    ON usage_records (api_key_id);

-- tenant_memberships by user_id (used in "list tenants for user" queries)
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user_id
    ON tenant_memberships (user_id);

-- requests by api_key_id (used in request-by-key lookups)
CREATE INDEX IF NOT EXISTS idx_requests_api_key_id
    ON requests (api_key_id);

-- +goose Down
DROP INDEX IF EXISTS idx_usage_records_api_key_id;
DROP INDEX IF EXISTS idx_tenant_memberships_user_id;
DROP INDEX IF EXISTS idx_requests_api_key_id;

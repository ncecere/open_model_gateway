-- +goose Up
CREATE TYPE admin_api_key_scope AS ENUM ('admin', 'system');

CREATE TABLE admin_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    prefix TEXT NOT NULL UNIQUE,
    secret_hash TEXT NOT NULL,
    scope admin_api_key_scope NOT NULL DEFAULT 'admin',
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_api_keys_owner_check CHECK (
        (scope = 'admin' AND owner_user_id IS NOT NULL) OR
        (scope = 'system' AND owner_user_id IS NULL)
    )
);

CREATE INDEX idx_admin_api_keys_owner ON admin_api_keys(owner_user_id);
CREATE INDEX idx_admin_api_keys_creator ON admin_api_keys(created_by_user_id);

-- +goose Down
DROP TABLE admin_api_keys;
DROP TYPE admin_api_key_scope;

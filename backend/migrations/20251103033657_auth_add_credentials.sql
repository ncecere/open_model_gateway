-- +goose Up
ALTER TABLE users
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN last_login_at TIMESTAMPTZ;

CREATE TABLE user_credentials (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       TEXT NOT NULL,
    issuer         TEXT NOT NULL DEFAULT '',
    subject        TEXT NOT NULL,
    password_hash  TEXT,
    metadata       JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, issuer, subject)
);

CREATE UNIQUE INDEX idx_user_credentials_user_provider
    ON user_credentials (user_id, provider, issuer);

-- +goose Down
DROP INDEX IF EXISTS idx_user_credentials_user_provider;
DROP TABLE IF EXISTS user_credentials;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS updated_at;

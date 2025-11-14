-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS theme_preference TEXT NOT NULL DEFAULT 'system';

ALTER TABLE users
    ADD CONSTRAINT users_theme_preference_check
    CHECK (theme_preference IN ('system', 'light', 'dark'));

-- +goose Down
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_theme_preference_check;

ALTER TABLE users
    DROP COLUMN IF EXISTS theme_preference;

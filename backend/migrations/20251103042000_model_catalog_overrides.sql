-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN deployment TEXT NOT NULL DEFAULT '',
    ADD COLUMN endpoint TEXT NOT NULL DEFAULT '',
    ADD COLUMN api_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN api_version TEXT NOT NULL DEFAULT '',
    ADD COLUMN region TEXT NOT NULL DEFAULT '',
    ADD COLUMN metadata_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    ADD COLUMN weight INT NOT NULL DEFAULT 100;

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS weight,
    DROP COLUMN IF EXISTS metadata_json,
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS api_version,
    DROP COLUMN IF EXISTS api_key,
    DROP COLUMN IF EXISTS endpoint,
    DROP COLUMN IF EXISTS deployment;

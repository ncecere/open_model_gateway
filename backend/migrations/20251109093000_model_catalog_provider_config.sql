-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN provider_config_json JSONB NOT NULL DEFAULT '{}'::JSONB;

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN provider_config_json;

-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN model_type TEXT NOT NULL DEFAULT 'llm';

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS model_type;

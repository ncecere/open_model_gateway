-- +goose Up
ALTER TABLE batches
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE batches DROP COLUMN IF EXISTS metadata;

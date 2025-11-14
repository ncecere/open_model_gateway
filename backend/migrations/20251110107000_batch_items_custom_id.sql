-- +goose Up
ALTER TABLE batch_items ADD COLUMN IF NOT EXISTS custom_id TEXT;

-- +goose Down
ALTER TABLE batch_items DROP COLUMN IF EXISTS custom_id;

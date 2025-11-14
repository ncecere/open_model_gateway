-- +goose Up
ALTER TABLE batch_items ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
ALTER TABLE batch_items ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE batch_items DROP COLUMN IF EXISTS completed_at;
ALTER TABLE batch_items DROP COLUMN IF EXISTS started_at;

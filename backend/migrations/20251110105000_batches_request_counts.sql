-- +goose Up
ALTER TABLE batches ADD COLUMN IF NOT EXISTS request_count_total INT NOT NULL DEFAULT 0;
ALTER TABLE batches ADD COLUMN IF NOT EXISTS request_count_completed INT NOT NULL DEFAULT 0;
ALTER TABLE batches ADD COLUMN IF NOT EXISTS request_count_failed INT NOT NULL DEFAULT 0;
ALTER TABLE batches ADD COLUMN IF NOT EXISTS request_count_cancelled INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE batches DROP COLUMN IF EXISTS request_count_cancelled;
ALTER TABLE batches DROP COLUMN IF EXISTS request_count_failed;
ALTER TABLE batches DROP COLUMN IF EXISTS request_count_completed;
ALTER TABLE batches DROP COLUMN IF EXISTS request_count_total;

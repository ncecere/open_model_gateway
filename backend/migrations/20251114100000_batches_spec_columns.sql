-- +goose Up
ALTER TABLE batches
    ADD COLUMN errors JSONB,
    ADD COLUMN cancelling_at TIMESTAMPTZ,
    ADD COLUMN expired_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS batches_created_cursor_idx
    ON batches (tenant_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS batches_created_global_idx
    ON batches (created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS batches_created_global_idx;
DROP INDEX IF EXISTS batches_created_cursor_idx;

ALTER TABLE batches
    DROP COLUMN IF EXISTS expired_at,
    DROP COLUMN IF EXISTS cancelling_at,
    DROP COLUMN IF EXISTS errors;

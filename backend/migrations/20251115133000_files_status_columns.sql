-- +goose Up
ALTER TABLE files
    ADD COLUMN status TEXT NOT NULL DEFAULT 'uploaded',
    ADD COLUMN status_details TEXT,
    ADD COLUMN status_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS files_tenant_created_idx ON files (tenant_id, created_at DESC, id);
CREATE INDEX IF NOT EXISTS files_purpose_idx ON files (purpose);

-- +goose Down
DROP INDEX IF EXISTS files_purpose_idx;
DROP INDEX IF EXISTS files_tenant_created_idx;

ALTER TABLE files
    DROP COLUMN IF EXISTS status_updated_at,
    DROP COLUMN IF EXISTS status_details,
    DROP COLUMN IF EXISTS status;

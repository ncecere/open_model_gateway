-- +goose Up
ALTER TABLE batches ADD COLUMN IF NOT EXISTS api_key_id UUID;

UPDATE batches AS b
SET api_key_id = (
    SELECT id FROM api_keys
    WHERE tenant_id = b.tenant_id
    ORDER BY created_at
    LIMIT 1
)
WHERE b.api_key_id IS NULL;

ALTER TABLE batches DROP CONSTRAINT IF EXISTS batches_api_key_id_fkey;
ALTER TABLE batches
    ADD CONSTRAINT batches_api_key_id_fkey FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;
ALTER TABLE batches
    ALTER COLUMN api_key_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS batches_api_key_idx ON batches (api_key_id);

-- +goose Down
ALTER TABLE batches DROP CONSTRAINT IF EXISTS batches_api_key_id_fkey;
DROP INDEX IF EXISTS batches_api_key_idx;
ALTER TABLE batches DROP COLUMN IF EXISTS api_key_id;

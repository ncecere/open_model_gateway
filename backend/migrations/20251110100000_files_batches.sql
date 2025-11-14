-- +goose Up
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    filename TEXT NOT NULL,
    purpose TEXT NOT NULL,
    content_type TEXT NOT NULL,
    bytes BIGINT NOT NULL,
    storage_backend TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    checksum TEXT,
    encrypted BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS files_tenant_idx ON files (tenant_id);
CREATE INDEX IF NOT EXISTS files_expires_at_idx ON files (expires_at);

CREATE TABLE IF NOT EXISTS batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    api_key_id UUID NOT NULL REFERENCES api_keys(id),
    status TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    input_file_id UUID REFERENCES files(id),
    result_file_id UUID REFERENCES files(id),
    error_file_id UUID REFERENCES files(id),
    completion_window TEXT,
    max_concurrency INT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_count_total INT NOT NULL DEFAULT 0,
    request_count_completed INT NOT NULL DEFAULT 0,
    request_count_failed INT NOT NULL DEFAULT 0,
    request_count_cancelled INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    in_progress_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    finalizing_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS batches_tenant_idx ON batches (tenant_id);
CREATE INDEX IF NOT EXISTS batches_status_idx ON batches (status);
CREATE INDEX IF NOT EXISTS batches_api_key_idx ON batches (api_key_id);

CREATE TABLE IF NOT EXISTS batch_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    item_index BIGINT NOT NULL,
    status TEXT NOT NULL,
    custom_id TEXT,
    input JSONB NOT NULL,
    response JSONB,
    error JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS batch_items_idx_unique ON batch_items (batch_id, item_index);
CREATE INDEX IF NOT EXISTS batch_items_status_idx ON batch_items (status);
CREATE INDEX IF NOT EXISTS batch_items_batch_idx ON batch_items (batch_id);

-- +goose Down
DROP TABLE IF EXISTS batch_items;
DROP TABLE IF EXISTS batches;
DROP TABLE IF EXISTS files;

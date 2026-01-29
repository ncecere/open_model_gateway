-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN managed_by TEXT NOT NULL DEFAULT 'ui';

-- Tag entries that were seeded by config on next startup.
-- The runtime catalog persister will set managed_by='config' for YAML-sourced entries.

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS managed_by;

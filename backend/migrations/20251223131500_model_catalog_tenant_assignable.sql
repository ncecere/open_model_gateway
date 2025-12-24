-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN tenant_assignable BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS tenant_assignable;

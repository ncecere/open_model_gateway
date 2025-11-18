-- +goose Up
ALTER TABLE tenant_models
    DROP CONSTRAINT IF EXISTS tenant_models_alias_fkey;

ALTER TABLE tenant_models
    ADD CONSTRAINT tenant_models_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias)
        ON DELETE CASCADE;

-- +goose Down
ALTER TABLE tenant_models
    DROP CONSTRAINT IF EXISTS tenant_models_alias_fkey;

ALTER TABLE tenant_models
    ADD CONSTRAINT tenant_models_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias);

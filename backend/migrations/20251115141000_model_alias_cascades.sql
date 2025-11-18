-- +goose Up
ALTER TABLE default_models
    DROP CONSTRAINT IF EXISTS default_models_alias_fkey;

ALTER TABLE default_models
    ADD CONSTRAINT default_models_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias)
        ON DELETE CASCADE;

ALTER TABLE routes
    DROP CONSTRAINT IF EXISTS routes_alias_fkey;

ALTER TABLE routes
    ADD CONSTRAINT routes_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias)
        ON DELETE CASCADE;

-- +goose Down
ALTER TABLE routes
    DROP CONSTRAINT IF EXISTS routes_alias_fkey;

ALTER TABLE routes
    ADD CONSTRAINT routes_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias);

ALTER TABLE default_models
    DROP CONSTRAINT IF EXISTS default_models_alias_fkey;

ALTER TABLE default_models
    ADD CONSTRAINT default_models_alias_fkey
        FOREIGN KEY (alias)
        REFERENCES model_catalog(alias);

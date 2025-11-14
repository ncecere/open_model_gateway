-- +goose Up
CREATE TABLE IF NOT EXISTS tenant_models (
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alias      TEXT NOT NULL REFERENCES model_catalog(alias),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, alias)
);

-- +goose Down
DROP TABLE IF EXISTS tenant_models;

-- +goose Up
ALTER TABLE tenant_memberships
    ADD COLUMN budget_usd NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (budget_usd >= 0),
    ADD COLUMN warning_threshold NUMERIC NOT NULL DEFAULT 0 CHECK (warning_threshold >= 0 AND warning_threshold <= 1),
    ADD COLUMN token_cap BIGINT NOT NULL DEFAULT 0 CHECK (token_cap >= 0);

-- +goose Down
ALTER TABLE tenant_memberships
    DROP COLUMN IF EXISTS token_cap,
    DROP COLUMN IF EXISTS warning_threshold,
    DROP COLUMN IF EXISTS budget_usd;

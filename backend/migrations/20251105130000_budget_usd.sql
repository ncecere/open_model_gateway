-- +goose Up
ALTER TABLE tenant_budget_overrides
    ADD COLUMN budget_usd NUMERIC(14,2);

UPDATE tenant_budget_overrides
SET budget_usd = ROUND(budget_cents::numeric / 100, 2);

ALTER TABLE tenant_budget_overrides
    DROP COLUMN budget_cents;

ALTER TABLE tenant_budget_overrides
    ALTER COLUMN budget_usd SET NOT NULL;

ALTER TABLE tenant_budget_overrides
    ADD CONSTRAINT tenant_budget_overrides_budget_usd_check CHECK (budget_usd >= 0);

-- +goose Down
ALTER TABLE tenant_budget_overrides
    ADD COLUMN budget_cents BIGINT;

UPDATE tenant_budget_overrides
SET budget_cents = COALESCE(ROUND(budget_usd * 100), 0);

ALTER TABLE tenant_budget_overrides
    DROP CONSTRAINT IF EXISTS tenant_budget_overrides_budget_usd_check;

ALTER TABLE tenant_budget_overrides
    DROP COLUMN budget_usd;

ALTER TABLE tenant_budget_overrides
    ALTER COLUMN budget_cents SET NOT NULL;

ALTER TABLE tenant_budget_overrides
    ADD CONSTRAINT tenant_budget_overrides_budget_cents_check CHECK (budget_cents >= 0);

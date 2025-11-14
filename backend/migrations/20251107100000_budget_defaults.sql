-- +goose Up
CREATE TABLE IF NOT EXISTS budget_defaults (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    default_usd NUMERIC(12,2) NOT NULL,
    warning_threshold NUMERIC(5,4) NOT NULL,
    refresh_schedule TEXT NOT NULL,
    alert_emails TEXT[],
    alert_webhooks TEXT[],
    alert_cooldown_seconds INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION budget_defaults_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER budget_defaults_updated_at
BEFORE UPDATE ON budget_defaults
FOR EACH ROW EXECUTE PROCEDURE budget_defaults_set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS budget_defaults_updated_at ON budget_defaults;
DROP FUNCTION IF EXISTS budget_defaults_set_updated_at();
DROP TABLE IF EXISTS budget_defaults;

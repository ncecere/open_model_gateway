-- +goose Up
ALTER TABLE tenant_budget_overrides
    ADD COLUMN refresh_schedule TEXT,
    ADD COLUMN alert_emails TEXT[],
    ADD COLUMN alert_webhooks TEXT[],
    ADD COLUMN alert_cooldown_seconds INTEGER,
    ADD COLUMN last_alert_at TIMESTAMPTZ,
    ADD COLUMN last_alert_level TEXT;

UPDATE tenant_budget_overrides
SET refresh_schedule = 'calendar_month'
WHERE refresh_schedule IS NULL;

ALTER TABLE tenant_budget_overrides
    ALTER COLUMN refresh_schedule SET NOT NULL;

UPDATE tenant_budget_overrides
SET alert_cooldown_seconds = 3600
WHERE alert_cooldown_seconds IS NULL;

ALTER TABLE tenant_budget_overrides
    ALTER COLUMN alert_cooldown_seconds SET NOT NULL,
    ALTER COLUMN alert_cooldown_seconds SET DEFAULT 3600;

-- +goose Down
ALTER TABLE tenant_budget_overrides
    DROP COLUMN last_alert_level,
    DROP COLUMN last_alert_at,
    DROP COLUMN alert_cooldown_seconds,
    DROP COLUMN alert_webhooks,
    DROP COLUMN alert_emails,
    DROP COLUMN refresh_schedule;

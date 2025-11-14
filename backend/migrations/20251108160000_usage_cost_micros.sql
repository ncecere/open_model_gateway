-- +goose Up
ALTER TABLE usage_records
    ADD COLUMN IF NOT EXISTS cost_usd_micros BIGINT NOT NULL DEFAULT 0;

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS cost_usd_micros BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE usage_records
    DROP COLUMN IF EXISTS cost_usd_micros;

ALTER TABLE requests
    DROP COLUMN IF EXISTS cost_usd_micros;

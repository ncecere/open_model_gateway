-- +goose Up
ALTER TABLE budget_defaults
    ADD COLUMN IF NOT EXISTS created_by_user_id UUID,
    ADD COLUMN IF NOT EXISTS updated_by_user_id UUID;

ALTER TABLE budget_defaults
    ADD CONSTRAINT budget_defaults_created_by_user_fk
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE budget_defaults
    ADD CONSTRAINT budget_defaults_updated_by_user_fk
        FOREIGN KEY (updated_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

-- initialize updated_by using created_by when possible
UPDATE budget_defaults
SET updated_by_user_id = COALESCE(updated_by_user_id, created_by_user_id);

-- +goose Down
ALTER TABLE budget_defaults
    DROP CONSTRAINT IF EXISTS budget_defaults_updated_by_user_fk,
    DROP CONSTRAINT IF EXISTS budget_defaults_created_by_user_fk,
    DROP COLUMN IF EXISTS updated_by_user_id,
    DROP COLUMN IF EXISTS created_by_user_id;

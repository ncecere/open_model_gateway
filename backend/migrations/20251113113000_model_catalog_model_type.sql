-- +goose Up
-- Idempotent in case the column shipped with initial schema.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'model_catalog'
          AND column_name = 'model_type'
    ) THEN
        ALTER TABLE model_catalog
            ADD COLUMN model_type TEXT NOT NULL DEFAULT 'llm';
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS model_type;

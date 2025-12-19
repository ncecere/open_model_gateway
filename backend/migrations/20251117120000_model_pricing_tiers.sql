-- +goose Up
ALTER TABLE model_catalog
    ADD COLUMN IF NOT EXISTS pricing_tiers_json JSONB NOT NULL DEFAULT '{}'::JSONB;

UPDATE model_catalog
SET pricing_tiers_json = jsonb_strip_nulls(
    jsonb_build_object(
        'input', CASE
            WHEN price_input IS NULL OR price_input = 0 THEN '[]'::JSONB
            ELSE jsonb_build_array(
                jsonb_build_object(
                    'unit', 'tokens_per_million',
                    'price_per_unit', price_input,
                    'metadata', '{}'::JSONB
                )
            )
        END,
        'output', CASE
            WHEN price_output IS NULL OR price_output = 0 THEN '[]'::JSONB
            ELSE jsonb_build_array(
                jsonb_build_object(
                    'unit', 'tokens_per_million',
                    'price_per_unit', price_output,
                    'metadata', '{}'::JSONB
                )
            )
        END
    )
)
WHERE pricing_tiers_json = '{}'::JSONB;

-- +goose Down
ALTER TABLE model_catalog
    DROP COLUMN IF EXISTS pricing_tiers_json;

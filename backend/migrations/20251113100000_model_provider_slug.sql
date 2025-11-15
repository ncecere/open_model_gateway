-- +goose Up
UPDATE model_catalog
SET provider = 'openai-compatible'
WHERE provider = 'openai_compatible';

-- +goose Down
UPDATE model_catalog
SET provider = 'openai_compatible'
WHERE provider = 'openai-compatible';

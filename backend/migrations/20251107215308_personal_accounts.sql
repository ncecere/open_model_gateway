-- +goose Up
CREATE TYPE tenant_kind AS ENUM ('organization', 'personal');
ALTER TABLE tenants
    ADD COLUMN kind tenant_kind NOT NULL DEFAULT 'organization';

ALTER TABLE users
    ADD COLUMN personal_tenant_id UUID UNIQUE REFERENCES tenants(id) ON DELETE SET NULL;

CREATE TYPE api_key_kind AS ENUM ('service', 'personal');
ALTER TABLE api_keys
    ADD COLUMN kind api_key_kind NOT NULL DEFAULT 'service';
ALTER TABLE api_keys
    ADD COLUMN owner_user_id UUID REFERENCES users(id);
ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_owner_check
        CHECK ((kind = 'personal' AND owner_user_id IS NOT NULL) OR (kind = 'service'));

CREATE TABLE IF NOT EXISTS default_models (
    alias TEXT PRIMARY KEY REFERENCES model_catalog(alias),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backfill personal tenants for existing users
INSERT INTO tenants (name, status, kind)
SELECT CONCAT('personal:', u.id::text), 'active', 'personal'
FROM users u
WHERE u.personal_tenant_id IS NULL;

UPDATE users u
SET personal_tenant_id = t.id
FROM tenants t
WHERE u.personal_tenant_id IS NULL
  AND t.kind = 'personal'
  AND t.name = CONCAT('personal:', u.id::text);

INSERT INTO tenant_memberships (id, tenant_id, user_id, role)
SELECT gen_random_uuid(), u.personal_tenant_id, u.id, 'owner'
FROM users u
WHERE u.personal_tenant_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM tenant_memberships tm
    WHERE tm.tenant_id = u.personal_tenant_id AND tm.user_id = u.id
  );

-- +goose Down
DELETE FROM tenant_memberships tm
WHERE tm.tenant_id IN (
    SELECT id FROM tenants WHERE kind = 'personal'
);

UPDATE users SET personal_tenant_id = NULL;

DELETE FROM tenants WHERE kind = 'personal';

DROP TABLE IF EXISTS default_models;

ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_owner_check;
ALTER TABLE api_keys DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE api_keys DROP COLUMN IF EXISTS kind;
DROP TYPE IF EXISTS api_key_kind;

ALTER TABLE tenants DROP COLUMN IF EXISTS kind;
DROP TYPE IF EXISTS tenant_kind;

ALTER TABLE users DROP COLUMN IF EXISTS personal_tenant_id;

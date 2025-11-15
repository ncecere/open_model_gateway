-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE tenant_status AS ENUM ('active', 'suspended');
CREATE TYPE membership_role AS ENUM ('owner', 'admin', 'viewer', 'user');

CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL UNIQUE,
    status        tenant_status NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       membership_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

CREATE TABLE api_keys (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    prefix         TEXT NOT NULL UNIQUE,
    secret_hash    TEXT NOT NULL,
    name           TEXT NOT NULL,
    scopes_json    JSONB NOT NULL DEFAULT '{}'::JSONB,
    quota_json     JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at     TIMESTAMPTZ,
    last_used_at   TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_tenant_created ON api_keys(tenant_id, created_at DESC);

CREATE TABLE model_catalog (
    alias              TEXT PRIMARY KEY,
    provider           TEXT NOT NULL,
    provider_model     TEXT NOT NULL,
    model_type         TEXT NOT NULL DEFAULT 'llm',
    context_window     INT NOT NULL,
    max_output_tokens  INT NOT NULL,
    modalities_json    JSONB NOT NULL DEFAULT '[]'::JSONB,
    supports_tools     BOOLEAN NOT NULL DEFAULT FALSE,
    price_input        NUMERIC(12,6) NOT NULL,
    price_output       NUMERIC(12,6) NOT NULL,
    currency           TEXT NOT NULL DEFAULT 'USD',
    enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alias           TEXT NOT NULL REFERENCES model_catalog(alias),
    provider        TEXT NOT NULL,
    provider_model  TEXT NOT NULL,
    weight          INT NOT NULL DEFAULT 100,
    params_json     JSONB NOT NULL DEFAULT '{}'::JSONB,
    region          TEXT,
    sticky_key      TEXT,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, alias, provider, provider_model)
);

CREATE INDEX idx_routes_tenant_alias ON routes(tenant_id, alias);

CREATE TABLE usage_records (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id     UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    ts             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model_alias    TEXT NOT NULL,
    provider       TEXT NOT NULL,
    input_tokens   BIGINT NOT NULL DEFAULT 0,
    output_tokens  BIGINT NOT NULL DEFAULT 0,
    requests       BIGINT NOT NULL DEFAULT 0,
    cost_cents     BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_usage_tenant_ts ON usage_records(tenant_id, ts DESC);
CREATE INDEX idx_usage_model_alias ON usage_records(model_alias);

CREATE TABLE requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id       UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    ts               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model_alias      TEXT NOT NULL,
    provider         TEXT NOT NULL,
    latency_ms       INT NOT NULL,
    status           INT NOT NULL,
    error_code       TEXT,
    input_tokens     BIGINT NOT NULL DEFAULT 0,
    output_tokens    BIGINT NOT NULL DEFAULT 0,
    cost_cents       BIGINT NOT NULL DEFAULT 0,
    idempotency_key  TEXT,
    trace_id         TEXT
);

CREATE INDEX idx_requests_tenant_ts ON requests(tenant_id, ts DESC);
CREATE INDEX idx_requests_model_alias ON requests(model_alias);
CREATE UNIQUE INDEX idx_requests_tenant_idempotency
    ON requests(tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_requests_model_alias;
DROP INDEX IF EXISTS idx_requests_tenant_idempotency;
DROP INDEX IF EXISTS idx_requests_tenant_ts;
DROP TABLE IF EXISTS requests;

DROP INDEX IF EXISTS idx_usage_model_alias;
DROP INDEX IF EXISTS idx_usage_tenant_ts;
DROP TABLE IF EXISTS usage_records;

DROP INDEX IF EXISTS idx_routes_tenant_alias;
DROP TABLE IF EXISTS routes;

DROP TABLE IF EXISTS model_catalog;

DROP INDEX IF EXISTS idx_api_keys_tenant_created;
DROP TABLE IF EXISTS api_keys;

DROP TABLE IF EXISTS tenant_memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

DROP TYPE IF EXISTS membership_role;
DROP TYPE IF EXISTS tenant_status;
-- extensions left in place intentionally

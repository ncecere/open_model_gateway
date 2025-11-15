CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE tenant_status AS ENUM ('active', 'suspended');
CREATE TYPE membership_role AS ENUM ('owner', 'admin', 'viewer', 'user');
CREATE TYPE tenant_kind AS ENUM ('organization', 'personal');
CREATE TYPE api_key_kind AS ENUM ('service', 'personal');

CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL UNIQUE,
    status        tenant_status NOT NULL DEFAULT 'active',
    kind          tenant_kind NOT NULL DEFAULT 'organization',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    theme_preference TEXT NOT NULL DEFAULT 'system' CHECK (theme_preference IN ('system', 'light', 'dark')),
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    personal_tenant_id UUID UNIQUE REFERENCES tenants(id) ON DELETE SET NULL,
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
    kind           api_key_kind NOT NULL DEFAULT 'service',
    owner_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at     TIMESTAMPTZ,
    last_used_at   TIMESTAMPTZ,
    CONSTRAINT api_keys_owner_check CHECK ((kind = 'personal' AND owner_user_id IS NOT NULL) OR (kind = 'service'))
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
    provider_config_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE default_models (
    alias TEXT PRIMARY KEY REFERENCES model_catalog(alias),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

CREATE TABLE tenant_models (
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alias      TEXT NOT NULL REFERENCES model_catalog(alias),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, alias)
);

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
    cost_cents     BIGINT NOT NULL DEFAULT 0,
    cost_usd_micros BIGINT NOT NULL DEFAULT 0
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
    cost_usd_micros   BIGINT NOT NULL DEFAULT 0,
    idempotency_key  TEXT,
    trace_id         TEXT
);

CREATE INDEX idx_requests_tenant_ts ON requests(tenant_id, ts DESC);
CREATE INDEX idx_requests_model_alias ON requests(model_alias);
CREATE UNIQUE INDEX idx_requests_tenant_idempotency
    ON requests(tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE TABLE admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_logs_created ON admin_audit_logs(created_at DESC);
CREATE INDEX idx_admin_audit_logs_user ON admin_audit_logs(user_id);

-- tenant budget overrides
CREATE TABLE IF NOT EXISTS tenant_budget_overrides (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    budget_usd NUMERIC(14,2) NOT NULL CHECK (budget_usd >= 0),
    warning_threshold NUMERIC NOT NULL CHECK (warning_threshold > 0 AND warning_threshold <= 1),
    refresh_schedule TEXT NOT NULL DEFAULT 'calendar_month',
    alert_emails TEXT[] DEFAULT '{}',
    alert_webhooks TEXT[] DEFAULT '{}',
    alert_cooldown_seconds INTEGER NOT NULL DEFAULT 3600,
    last_alert_at TIMESTAMPTZ,
    last_alert_level TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS budget_defaults (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    default_usd NUMERIC(14,2) NOT NULL CHECK (default_usd >= 0),
    warning_threshold NUMERIC NOT NULL CHECK (warning_threshold > 0 AND warning_threshold <= 1),
    refresh_schedule TEXT NOT NULL DEFAULT 'calendar_month',
    alert_emails TEXT[] DEFAULT '{}',
    alert_webhooks TEXT[] DEFAULT '{}',
    alert_cooldown_seconds INTEGER NOT NULL DEFAULT 3600,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS rate_limit_defaults (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    requests_per_minute INTEGER NOT NULL CHECK (requests_per_minute >= 0),
    tokens_per_minute INTEGER NOT NULL CHECK (tokens_per_minute >= 0),
    parallel_requests_key INTEGER NOT NULL CHECK (parallel_requests_key >= 0),
    parallel_requests_tenant INTEGER NOT NULL CHECK (parallel_requests_tenant >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    filename TEXT NOT NULL,
    purpose TEXT NOT NULL,
    content_type TEXT NOT NULL,
    bytes BIGINT NOT NULL,
    storage_backend TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    checksum TEXT,
    encrypted BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS files_tenant_idx ON files (tenant_id);
CREATE INDEX IF NOT EXISTS files_expires_at_idx ON files (expires_at);

CREATE TABLE IF NOT EXISTS batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    api_key_id UUID NOT NULL REFERENCES api_keys(id),
    status TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    input_file_id UUID REFERENCES files(id),
    result_file_id UUID REFERENCES files(id),
    error_file_id UUID REFERENCES files(id),
    errors JSONB,
    completion_window TEXT,
    max_concurrency INTEGER NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    request_count_total INTEGER NOT NULL DEFAULT 0,
    request_count_completed INTEGER NOT NULL DEFAULT 0,
    request_count_failed INTEGER NOT NULL DEFAULT 0,
    request_count_cancelled INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    in_progress_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancelling_at TIMESTAMPTZ,
    finalizing_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS batches_tenant_idx ON batches (tenant_id);
CREATE INDEX IF NOT EXISTS batches_status_idx ON batches (status);
CREATE INDEX IF NOT EXISTS batches_api_key_idx ON batches (api_key_id);
CREATE INDEX IF NOT EXISTS batches_created_cursor_idx ON batches (tenant_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS batches_created_global_idx ON batches (created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS batch_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    item_index BIGINT NOT NULL,
    status TEXT NOT NULL,
    custom_id TEXT,
    input JSONB NOT NULL,
    response JSONB,
    error JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS batch_items_idx_unique ON batch_items (batch_id, item_index);
CREATE INDEX IF NOT EXISTS batch_items_status_idx ON batch_items (status);
CREATE INDEX IF NOT EXISTS batch_items_batch_idx ON batch_items (batch_id);

CREATE TRIGGER budget_defaults_updated_at
    BEFORE UPDATE ON budget_defaults
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER rate_limit_defaults_updated_at
    BEFORE UPDATE ON rate_limit_defaults
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER tenant_budget_overrides_updated_at
    BEFORE UPDATE ON tenant_budget_overrides
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
-- helper function to set updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $func$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$func$ LANGUAGE plpgsql;

CREATE TABLE responses_cache (
    id          TEXT        PRIMARY KEY,
    tenant_id   UUID        NOT NULL,
    model       TEXT        NOT NULL,
    input       JSONB       NOT NULL,
    output      JSONB       NOT NULL,
    instructions TEXT,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '7 days')
);

CREATE INDEX idx_responses_cache_tenant ON responses_cache (tenant_id);
CREATE INDEX idx_responses_cache_expires ON responses_cache (expires_at);

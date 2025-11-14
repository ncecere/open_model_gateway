-- +goose Up
CREATE TABLE admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_logs_created ON admin_audit_logs(created_at DESC);
CREATE INDEX idx_admin_audit_logs_user ON admin_audit_logs(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_audit_logs_user;
DROP INDEX IF EXISTS idx_admin_audit_logs_created;
DROP TABLE IF EXISTS admin_audit_logs;

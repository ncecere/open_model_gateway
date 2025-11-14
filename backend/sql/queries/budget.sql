-- name: ListTenantBudgetOverrides :many
SELECT tenant_id,
       budget_usd,
       warning_threshold,
       refresh_schedule,
       alert_emails,
       alert_webhooks,
       alert_cooldown_seconds,
       last_alert_at,
       last_alert_level,
       created_at,
       updated_at
FROM tenant_budget_overrides
ORDER BY created_at DESC;

-- name: GetTenantBudgetOverride :one
SELECT tenant_id,
       budget_usd,
       warning_threshold,
       refresh_schedule,
       alert_emails,
       alert_webhooks,
       alert_cooldown_seconds,
       last_alert_at,
       last_alert_level,
       created_at,
       updated_at
FROM tenant_budget_overrides
WHERE tenant_id = $1;

-- name: UpsertTenantBudgetOverride :one
INSERT INTO tenant_budget_overrides (
    tenant_id,
    budget_usd,
    warning_threshold,
    refresh_schedule,
    alert_emails,
    alert_webhooks,
    alert_cooldown_seconds
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id) DO UPDATE
SET budget_usd = EXCLUDED.budget_usd,
    warning_threshold = EXCLUDED.warning_threshold,
    refresh_schedule = EXCLUDED.refresh_schedule,
    alert_emails = EXCLUDED.alert_emails,
    alert_webhooks = EXCLUDED.alert_webhooks,
    alert_cooldown_seconds = EXCLUDED.alert_cooldown_seconds,
    updated_at = NOW()
RETURNING tenant_id,
          budget_usd,
          warning_threshold,
          refresh_schedule,
          alert_emails,
          alert_webhooks,
          alert_cooldown_seconds,
          last_alert_at,
          last_alert_level,
          created_at,
          updated_at;

-- name: DeleteTenantBudgetOverride :exec
DELETE FROM tenant_budget_overrides
WHERE tenant_id = $1;

-- name: UpdateTenantBudgetAlertState :exec
UPDATE tenant_budget_overrides
SET last_alert_at = $2,
    last_alert_level = $3,
    updated_at = NOW()
WHERE tenant_id = $1;

-- name: GetBudgetDefaults :one
SELECT id,
       default_usd,
       warning_threshold,
       refresh_schedule,
       alert_emails,
       alert_webhooks,
       alert_cooldown_seconds,
       created_at,
       created_by_user_id,
       updated_at,
       updated_by_user_id
FROM budget_defaults
WHERE id = TRUE;

-- name: UpsertBudgetDefaults :one
INSERT INTO budget_defaults (
    id,
    default_usd,
    warning_threshold,
    refresh_schedule,
    alert_emails,
    alert_webhooks,
    alert_cooldown_seconds,
    created_by_user_id,
    updated_by_user_id
)
VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET default_usd = EXCLUDED.default_usd,
    warning_threshold = EXCLUDED.warning_threshold,
    refresh_schedule = EXCLUDED.refresh_schedule,
    alert_emails = EXCLUDED.alert_emails,
    alert_webhooks = EXCLUDED.alert_webhooks,
    alert_cooldown_seconds = EXCLUDED.alert_cooldown_seconds,
    updated_at = NOW(),
    updated_by_user_id = EXCLUDED.updated_by_user_id,
    created_by_user_id = COALESCE(budget_defaults.created_by_user_id, EXCLUDED.created_by_user_id)
RETURNING id,
          default_usd,
          warning_threshold,
          refresh_schedule,
          alert_emails,
          alert_webhooks,
          alert_cooldown_seconds,
          created_at,
          created_by_user_id,
          updated_at,
          updated_by_user_id;

-- name: InsertBudgetAlertEvent :exec
INSERT INTO budget_alert_events (
    tenant_id,
    level,
    channels,
    payload,
    success,
    error
)
VALUES ($1, $2, $3, $4, $5, $6);

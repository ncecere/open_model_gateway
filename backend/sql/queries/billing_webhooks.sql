-- name: CreateBillingWebhook :one
INSERT INTO billing_webhooks (
    tenant_id,
    name,
    url,
    secret,
    enabled
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateBillingWebhook :one
UPDATE billing_webhooks
SET name = $2,
    url = $3,
    secret = $4,
    enabled = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBillingWebhook :exec
DELETE FROM billing_webhooks
WHERE id = $1;

-- name: GetBillingWebhook :one
SELECT *
FROM billing_webhooks
WHERE id = $1;

-- name: ListBillingWebhooksAdmin :many
SELECT *
FROM billing_webhooks
ORDER BY created_at DESC;

-- name: ListBillingWebhooksByTenant :many
SELECT *
FROM billing_webhooks
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: InsertBillingWebhookEvent :one
INSERT INTO billing_webhook_events (
    webhook_id,
    tenant_id,
    period_start,
    period_end,
    payload,
    success,
    status_code,
    error
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListBillingWebhookEvents :many
SELECT *
FROM billing_webhook_events
WHERE webhook_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetBillingWebhookEvent :one
SELECT *
FROM billing_webhook_events
WHERE id = $1;

-- name: CreateBatch :one
INSERT INTO batches (
    tenant_id,
    api_key_id,
    status,
    endpoint,
    input_file_id,
    completion_window,
    max_concurrency,
    metadata,
    request_count_total,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: InsertBatchItem :one
INSERT INTO batch_items (
    batch_id,
    item_index,
    status,
    custom_id,
    input
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBatch :one
SELECT *
FROM batches
WHERE tenant_id = $1 AND id = $2;

-- name: GetBatchByID :one
SELECT *
FROM batches
WHERE id = $1;

-- name: ListBatches :many
SELECT *
FROM batches
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBatchesCursor :many
WITH anchor AS (
    SELECT b.created_at, b.id
    FROM batches b
    WHERE b.id = sqlc.narg(after_id)::uuid
)
SELECT b.*
FROM batches b
LEFT JOIN anchor a ON true
WHERE b.tenant_id = $1
  AND (
        a.created_at IS NULL
        OR b.created_at < a.created_at
        OR (b.created_at = a.created_at AND b.id < a.id)
  )
ORDER BY b.created_at DESC, b.id DESC
LIMIT $2;

-- name: ListBatchesAdmin :many
SELECT b.*, t.name AS tenant_name, COUNT(*) OVER() AS total_count
FROM batches b
JOIN tenants t ON t.id = b.tenant_id
WHERE (sqlc.narg(tenant_id)::uuid IS NULL OR b.tenant_id = sqlc.narg(tenant_id))
  AND (
    sqlc.narg(statuses)::text[] IS NULL
    OR cardinality(sqlc.narg(statuses)) = 0
    OR b.status = ANY(sqlc.narg(statuses))
  )
  AND (
    sqlc.narg(search)::text IS NULL
    OR b.endpoint ILIKE '%' || sqlc.narg(search) || '%'
    OR t.name ILIKE '%' || sqlc.narg(search) || '%'
    OR b.id::text ILIKE '%' || sqlc.narg(search) || '%'
    OR b.metadata::text ILIKE '%' || sqlc.narg(search) || '%'
  )
ORDER BY b.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetOldestQueuedBatch :one
SELECT *
FROM batches
WHERE status = 'validating'
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: MarkBatchInProgress :one
UPDATE batches
SET status = 'in_progress',
    in_progress_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND status = 'validating'
RETURNING *;

-- name: CancelBatch :one
UPDATE batches
SET status = CASE
        WHEN status = 'validating' THEN 'cancelled'
        ELSE 'cancelling'
    END,
    cancelled_at = CASE WHEN status = 'validating' THEN NOW() ELSE cancelled_at END,
    cancelling_at = CASE WHEN status <> 'validating' THEN NOW() ELSE cancelling_at END,
    updated_at = NOW()
WHERE tenant_id = $1 AND id = $2 AND status IN ('validating', 'in_progress', 'finalizing')
RETURNING *;

-- name: UpdateBatchCounts :one
UPDATE batches
SET request_count_total = request_count_total + $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ClaimNextBatchItem :one
WITH next_item AS (
    SELECT bi.id
    FROM batch_items bi
    JOIN batches b ON b.id = bi.batch_id
    WHERE bi.batch_id = $1
      AND bi.status = 'queued'
      AND b.status = 'in_progress'
    ORDER BY bi.item_index
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE batch_items
SET status = 'running',
    started_at = NOW()
WHERE id = (SELECT id FROM next_item)
RETURNING *;

-- name: CompleteBatchItem :exec
UPDATE batch_items
SET status = 'completed',
    completed_at = NOW(),
    response = $2
WHERE id = $1;

-- name: FailBatchItem :exec
UPDATE batch_items
SET status = 'failed',
    completed_at = NOW(),
    error = $2
WHERE id = $1;

-- name: IncrementBatchCounts :exec
UPDATE batches
SET request_count_completed = request_count_completed + $2,
    request_count_failed = request_count_failed + $3,
    request_count_cancelled = request_count_cancelled + $4,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkBatchFinalStatus :one
UPDATE batches
SET status = $2,
    completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END,
    failed_at = CASE WHEN $2 = 'failed' THEN NOW() ELSE failed_at END,
    finalizing_at = CASE WHEN $2 = 'finalizing' THEN NOW() ELSE finalizing_at END,
    cancelled_at = CASE WHEN $2 = 'cancelled' THEN NOW() ELSE cancelled_at END,
    cancelling_at = CASE WHEN $2 = 'cancelling' THEN NOW() ELSE cancelling_at END,
    expired_at = CASE WHEN $2 = 'expired' THEN NOW() ELSE expired_at END,
    result_file_id = $3,
    error_file_id = $4,
    errors = COALESCE(sqlc.narg(errors)::jsonb, errors),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListBatchItemsForOutput :many
SELECT *
FROM batch_items
WHERE batch_id = $1
ORDER BY item_index;

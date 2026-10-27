-- name: CreateStore :one
INSERT INTO stores (
    user_id,
    name,
    description,
    appointment_duration,
    timezone
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListStoresPaginated :many
SELECT * FROM stores
WHERE deleted_at IS NULL
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: GetStoreByID :one
SELECT * FROM stores
WHERE id = $1 
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetStoreByUserID :one
SELECT * FROM stores
WHERE user_id = $1 
  AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateStore :one
UPDATE stores
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    is_active = COALESCE($4, is_active),
    appointment_duration = COALESCE($5, appointment_duration),
    timezone = COALESCE($6, timezone),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteStore :exec
-- Soft delete: Updates `deleted_at` instead of deleting the row.
UPDATE stores
SET 
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 
  AND deleted_at IS NULL;

-- name: CreateStore :one
INSERT INTO stores (
    user_id,
    name,
    description
) VALUES (
    $1, $2, $3
)
RETURNING *;

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

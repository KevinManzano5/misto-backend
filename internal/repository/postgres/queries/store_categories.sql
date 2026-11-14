-- name: ListStoreCategories :many
-- Full catalog, for a frontend selection UI.
SELECT * FROM store_categories
ORDER BY name;

-- name: GetStoreCategoryByID :one
SELECT * FROM store_categories
WHERE id = $1
LIMIT 1;

-- name: CreateStoreCategory :one
INSERT INTO store_categories (name, slug, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateStoreCategory :one
UPDATE store_categories
SET
    name        = $2,
    slug        = $3,
    description = $4,
    updated_at  = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteStoreCategory :exec
DELETE FROM store_categories
WHERE id = $1;

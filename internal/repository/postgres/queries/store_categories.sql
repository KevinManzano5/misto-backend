-- name: ListStoreCategories :many
-- Full catalog, for a frontend selection UI.
SELECT * FROM store_categories
ORDER BY name;

-- name: GetStoreCategoryByID :one
SELECT * FROM store_categories
WHERE id = $1
LIMIT 1;

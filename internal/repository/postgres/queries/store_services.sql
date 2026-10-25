-- name: CreateStoreService :one
INSERT INTO store_services (
    store_id,
    name,
    description,
    price,
    duration_minutes
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetStoreServiceByID :one
SELECT * FROM store_services
WHERE id = $1 
LIMIT 1;

-- name: ListStoreServicesByStoreID :many
-- We filter by is_active = true to show only the services available to the client.
SELECT * FROM store_services
WHERE store_id = $1 
  AND is_active = TRUE
ORDER BY name ASC;

-- name: UpdateStoreService :one
UPDATE store_services
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    price = COALESCE(sqlc.narg('price'), price),
    duration_minutes = COALESCE(sqlc.narg('duration_minutes'), duration_minutes),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteStoreService :exec
DELETE FROM store_services
WHERE id = $1;

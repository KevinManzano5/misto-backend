-- name: CreateStoreService :one
INSERT INTO store_services (store_id, category_id, name, description, duration_minutes, price)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStoreServiceByID :one
SELECT * FROM store_services
WHERE id = $1
LIMIT 1;

-- name: ListStoreServicesByStore :many
SELECT * FROM store_services
WHERE store_id = $1 AND is_active = TRUE
ORDER BY name;

-- name: ListAllStoreServicesByStore :many
SELECT * FROM store_services
WHERE store_id = $1
ORDER BY name;

-- name: UpdateStoreService :one
UPDATE store_services
SET
    category_id      = $2,
    name             = $3,
    description      = $4,
    duration_minutes = $5,
    price            = $6,
    is_active        = $7,
    updated_at       = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteStoreService :exec
DELETE FROM store_services
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    password_hash,
    first_name,
    last_name
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1
LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET
    first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    is_active = COALESCE($4, is_active),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

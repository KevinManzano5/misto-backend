-- name: CreateStoreRating :one
INSERT INTO store_ratings (
    appointment_id,
    store_id,
    client_id,
    score,
    comment
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetStoreRatingByID :one
SELECT * FROM store_ratings
WHERE id = $1
LIMIT 1;

-- name: GetStoreRatingByAppointmentID :one
-- Used by the usecase to proactively reject a second rating for the same appointment with a
-- clean domain.ErrAlreadyExists before relying on the DB unique constraint.
SELECT * FROM store_ratings
WHERE appointment_id = $1
LIMIT 1;

-- name: ListStoreRatingsByStore :many
-- Paginated, newest first, each paired with the basic profile of the client who left it, for a
-- store's public rating list.
SELECT
    r.id,
    r.appointment_id,
    r.store_id,
    r.client_id,
    r.score,
    r.comment,
    r.created_at,
    r.updated_at,
    u.username AS client_username,
    u.first_name AS client_first_name,
    u.last_name AS client_last_name
FROM store_ratings r
JOIN users u ON u.id = r.client_id
WHERE r.store_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoreRatingsByStore :one
SELECT COUNT(*) FROM store_ratings
WHERE store_id = $1;

-- name: GetStoreRatingSummaryByStore :one
-- Average score + count, computed on demand rather than denormalized onto stores.
SELECT
    COALESCE(AVG(score)::float8, 0)::float8 AS average_score,
    COUNT(*) AS rating_count
FROM store_ratings
WHERE store_id = $1;

-- name: UpdateStoreRating :one
UPDATE store_ratings
SET
    score = $2,
    comment = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteStoreRating :exec
DELETE FROM store_ratings
WHERE id = $1;

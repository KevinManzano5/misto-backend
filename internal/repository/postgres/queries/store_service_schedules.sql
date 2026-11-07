-- name: UpsertStoreServiceScheduleDay :one
INSERT INTO store_service_schedules (store_service_id, day_of_week, is_closed)
VALUES ($1, $2, $3)
ON CONFLICT (store_service_id, day_of_week)
DO UPDATE SET
    is_closed = EXCLUDED.is_closed,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetStoreServiceScheduleDay :one
SELECT * FROM store_service_schedules
WHERE store_service_id = $1 AND day_of_week = $2 LIMIT 1;

-- name: ListStoreServiceScheduleDays :many
SELECT * FROM store_service_schedules
WHERE store_service_id = $1 ORDER BY day_of_week ASC;

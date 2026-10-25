-- name: UpsertStoreSchedule :one
INSERT INTO store_schedules (store_id, day_of_week, open_time, close_time, is_closed)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (store_id, day_of_week) 
DO UPDATE SET 
    open_time = EXCLUDED.open_time,
    close_time = EXCLUDED.close_time,
    is_closed = EXCLUDED.is_closed,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetStoreScheduleByDay :one
SELECT * FROM store_schedules
WHERE store_id = $1 AND day_of_week = $2 LIMIT 1;

-- name: ListStoreSchedules :many
SELECT * FROM store_schedules
WHERE store_id = $1 ORDER BY day_of_week ASC;

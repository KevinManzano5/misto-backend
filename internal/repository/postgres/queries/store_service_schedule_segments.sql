-- name: DeleteStoreServiceScheduleSegments :exec
DELETE FROM store_service_schedule_segments
WHERE store_service_schedule_id = $1;

-- name: CreateStoreServiceScheduleSegment :one
INSERT INTO store_service_schedule_segments (store_service_schedule_id, open_time, close_time)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListStoreServiceScheduleSegments :many
SELECT * FROM store_service_schedule_segments
WHERE store_service_schedule_id = $1 ORDER BY open_time ASC;

-- name: ListStoreServiceScheduleSegmentsByScheduleIDs :many
SELECT * FROM store_service_schedule_segments
WHERE store_service_schedule_id = ANY($1::uuid[]) ORDER BY store_service_schedule_id, open_time ASC;

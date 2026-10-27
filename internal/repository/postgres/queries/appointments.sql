-- name: CheckAppointmentOverlap :one
-- Check whether there is any active appointment whose time interval overlaps with the new requested range.
SELECT COUNT(*) FROM appointments
WHERE store_id = sqlc.arg(store_id)
  AND status != 'cancelled'
  AND start_time < sqlc.arg(new_end_time)    -- The existing appointment begins before the new one ends.
  AND end_time > sqlc.arg(new_start_time);   -- The existing appointment ends after the new one begins.

-- name: CreateAppointment :one
INSERT INTO appointments (
    store_id,
    client_id,
    start_time,
    end_time,
    status
) VALUES (
    $1, $2, $3, $4, 'confirmed'
)
RETURNING *;

-- name: ListAppointmentsByStoreDate :many
-- Useful for the store to view its daily/weekly calendar
SELECT * FROM appointments
WHERE store_id = $1
  AND start_time >= $2
  AND start_time <= $3
  AND status != 'cancelled'
ORDER BY start_time ASC;

-- name: CreateService :one
INSERT INTO store_services (
    store_id,
    name,
    description,
    duration_minutes,
    price
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListStoreServices :many
SELECT * FROM store_services
WHERE store_id = $1 
  AND is_active = TRUE
ORDER BY name ASC;

-- name: CheckAppointmentOverlap :one
-- Check whether there is any active appointment whose time interval overlaps with the new requested range.
SELECT COUNT(*) FROM appointments
WHERE store_id = $1
  AND status != 'cancelled'
  AND start_time < $3  -- The existing appointment begins before the new one ends.
  AND end_time > $2;   -- The existing appointment ends after the new one begins.

-- name: CreateAppointment :one
INSERT INTO appointments (
    store_id,
    service_id,
    client_id,
    start_time,
    end_time,
    status
) VALUES (
    $1, $2, $3, $4, $5, 'confirmed'
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
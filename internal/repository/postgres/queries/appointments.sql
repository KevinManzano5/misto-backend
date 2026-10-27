-- name: CheckAppointmentOverlap :one
-- Check whether there is any active appointment whose time interval overlaps with the new requested range.
SELECT COUNT(*) FROM appointments
WHERE store_id = sqlc.arg(store_id)
  AND status != 'cancelled'
  AND start_time < sqlc.arg(new_end_time)    -- The existing appointment begins before the new one ends.
  AND end_time > sqlc.arg(new_start_time);   -- The existing appointment ends after the new one begins.

-- name: CheckClientHasActiveAppointment :one
-- Check whether the client already has a pending or confirmed appointment at this store.
SELECT COUNT(*) FROM appointments
WHERE store_id = sqlc.arg(store_id)
  AND client_id = sqlc.arg(client_id)
  AND status IN ('pending', 'confirmed');

-- name: CreateAppointment :one
INSERT INTO appointments (
    store_id,
    client_id,
    start_time,
    end_time,
    status
) VALUES (
    $1, $2, $3, $4, $5
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

-- name: ListAllAppointmentsByStoreWithClient :many
-- Every appointment ever booked at a store, any status, along with the client's basic profile,
-- for the owner's full history view.
SELECT
    a.id,
    a.store_id,
    a.client_id,
    a.start_time,
    a.end_time,
    a.status,
    a.created_at,
    a.updated_at,
    u.username AS client_username,
    u.email AS client_email,
    u.first_name AS client_first_name,
    u.last_name AS client_last_name
FROM appointments a
JOIN users u ON u.id = a.client_id
WHERE a.store_id = $1
ORDER BY a.start_time DESC;

-- name: GetAppointmentByID :one
SELECT * FROM appointments
WHERE id = $1
LIMIT 1;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

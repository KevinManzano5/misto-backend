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
-- Every appointment ever booked at a store, any status, optionally restricted to an inclusive
-- start_time range, along with the client's basic profile, for the owner's history view. Either
-- bound may be NULL to leave that side of the range open.
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
  AND (sqlc.narg(start_date)::timestamptz IS NULL OR a.start_time >= sqlc.narg(start_date))
  AND (sqlc.narg(end_date)::timestamptz IS NULL OR a.start_time <= sqlc.narg(end_date))
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

-- name: ListAppointmentsByClient :many
-- Every appointment ever booked by a client, any status, most recent first, along with the
-- basic info of the store it was booked at, for the client's own "my appointments" view.
SELECT
    a.id,
    a.store_id,
    a.client_id,
    a.start_time,
    a.end_time,
    a.status,
    a.created_at,
    a.updated_at,
    s.name AS store_name
FROM appointments a
JOIN stores s ON s.id = a.store_id
WHERE a.client_id = $1
ORDER BY a.start_time DESC;

-- name: ExpireOverdueAppointments :many
-- Moves pending/confirmed appointments whose end_time has already passed to a terminal status:
-- confirmed appointments are assumed to have happened (completed), pending ones were never
-- reviewed in time by the store (expired).
UPDATE appointments
SET
    status = CASE status
        WHEN 'confirmed' THEN 'completed'
        WHEN 'pending' THEN 'expired'
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE status IN ('pending', 'confirmed')
  AND end_time < sqlc.arg(before)
RETURNING *;

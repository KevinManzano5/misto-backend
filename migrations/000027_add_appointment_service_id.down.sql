DROP INDEX IF EXISTS idx_appointments_service_id;
ALTER TABLE appointments
    DROP CONSTRAINT IF EXISTS fk_appointments_service,
    DROP COLUMN IF EXISTS service_id;

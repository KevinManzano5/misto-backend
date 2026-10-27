-- Stores now define a single default appointment duration instead of a catalog of services.
ALTER TABLE stores ADD COLUMN default_appointment_duration INT NOT NULL DEFAULT 30;

ALTER TABLE appointments DROP CONSTRAINT IF EXISTS fk_appointments_service;
ALTER TABLE appointments DROP COLUMN IF EXISTS service_id;

DROP TABLE IF EXISTS store_services CASCADE;

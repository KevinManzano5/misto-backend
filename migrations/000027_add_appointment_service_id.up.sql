-- Links an appointment to the specific store service that was booked, so its duration (and,
-- going forward, its price) derive from the service instead of a single store-wide default.
ALTER TABLE appointments
    ADD COLUMN service_id UUID,
    ADD CONSTRAINT fk_appointments_service FOREIGN KEY (service_id) REFERENCES store_services(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_appointments_service_id ON appointments(service_id);

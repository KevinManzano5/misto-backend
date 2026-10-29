-- Speeds up the periodic sweep that finds pending/confirmed appointments whose end_time
-- has already passed, so they can be auto-expired.
CREATE INDEX idx_appointments_status_end_time ON appointments(status, end_time);

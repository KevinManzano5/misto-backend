-- The store's own IANA timezone (e.g. "America/Bogota"), used to interpret its schedule's
-- open/close hours and to resolve which calendar day an appointment falls on, independent of UTC.
ALTER TABLE stores ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'UTC';

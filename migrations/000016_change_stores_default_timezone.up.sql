-- This business operates in Colombia; stores created without an explicit timezone should
-- default to America/Bogota instead of UTC.
ALTER TABLE stores ALTER COLUMN timezone SET DEFAULT 'America/Bogota';

-- Controls whether clients can book an appointment on the same calendar day it is requested,
-- in the store's own timezone. Defaults to TRUE to preserve existing behavior.
ALTER TABLE stores ADD COLUMN allow_same_day_appointments BOOLEAN NOT NULL DEFAULT TRUE;

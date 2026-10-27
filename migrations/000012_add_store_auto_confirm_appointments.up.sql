-- Controls whether new appointments are confirmed automatically or left pending for the
-- store owner to approve/reject. Defaults to TRUE (auto-confirm) so existing stores keep
-- their current behavior until the owner opts into manual approval.
ALTER TABLE stores ADD COLUMN auto_confirm_appointments BOOLEAN NOT NULL DEFAULT TRUE;

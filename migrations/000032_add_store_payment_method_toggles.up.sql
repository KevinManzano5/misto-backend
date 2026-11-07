-- Permite a la tienda encender/apagar cada método de pago soportado. Default TRUE
-- para preservar el comportamiento actual (ambos métodos habilitados).
ALTER TABLE stores ADD COLUMN cash_on_site_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE stores ADD COLUMN card_advance_enabled BOOLEAN NOT NULL DEFAULT TRUE;

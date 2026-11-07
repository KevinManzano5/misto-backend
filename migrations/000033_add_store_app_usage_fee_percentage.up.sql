-- Porcentaje (0-100) que la plataforma cobra a la tienda por el uso de la app. Default 0 para
-- preservar el comportamiento actual (sin comisión) en tiendas existentes. Visible/editable solo
-- por administradores de la plataforma, no por el dueño de la tienda.
ALTER TABLE stores ADD COLUMN app_usage_fee_percentage SMALLINT NOT NULL DEFAULT 0
  CHECK (app_usage_fee_percentage BETWEEN 0 AND 100);

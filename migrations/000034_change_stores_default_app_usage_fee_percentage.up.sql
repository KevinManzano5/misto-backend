-- La plataforma cobra un 10% por uso de la app a toda tienda, actual y futura. El default nuevo
-- cubre las tiendas que se creen de ahora en adelante; el UPDATE lleva las ya existentes al mismo
-- valor (todas estaban en 0, el default anterior).
ALTER TABLE stores ALTER COLUMN app_usage_fee_percentage SET DEFAULT 10;
UPDATE stores SET app_usage_fee_percentage = 10;

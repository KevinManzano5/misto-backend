ALTER TABLE stores ALTER COLUMN app_usage_fee_percentage SET DEFAULT 0;
UPDATE stores SET app_usage_fee_percentage = 0;

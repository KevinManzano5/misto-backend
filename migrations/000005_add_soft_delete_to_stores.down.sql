-- Revert changes
DROP INDEX IF EXISTS idx_stores_name_active;
ALTER TABLE stores ADD CONSTRAINT stores_name_key UNIQUE (name);
ALTER TABLE stores DROP COLUMN IF EXISTS deleted_at;

-- 1. Add the deleted_at column
ALTER TABLE stores ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- 2. Remove the previous standard unique constraint (stores_name_key)
-- to replace it with a unique partial index that applies only to active stores
ALTER TABLE stores DROP CONSTRAINT IF EXISTS stores_name_key;

-- 3. Create partial unique index: Allows duplicate names if the previous store was logically deleted.
CREATE UNIQUE INDEX idx_stores_name_active ON stores(name) WHERE deleted_at IS NULL;

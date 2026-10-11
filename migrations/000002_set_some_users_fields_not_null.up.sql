-- 1. Assign a default value to existing rows that have NULL
UPDATE users 
SET first_name = '' 
WHERE first_name IS NULL;

UPDATE users 
SET last_name = '' 
WHERE last_name IS NULL;

-- 2. Apply the NOT NULL constraint
ALTER TABLE users 
    ALTER COLUMN first_name SET NOT NULL,
    ALTER COLUMN last_name SET NOT NULL;

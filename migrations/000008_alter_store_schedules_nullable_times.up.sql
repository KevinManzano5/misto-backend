ALTER TABLE store_schedules
    ALTER COLUMN open_time DROP NOT NULL,
    ALTER COLUMN close_time DROP NOT NULL;

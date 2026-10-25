CREATE TABLE IF NOT EXISTS store_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL,
    day_of_week INT NOT NULL, --0 = Sunday, 1 = Monday, ..., 6 = Saturday
    open_time TIME NOT NULL,  -- Ej: '09:00:00'
    close_time TIME NOT NULL, -- Ej: '18:00:00'
    is_closed BOOLEAN DEFAULT FALSE NOT NULL, -- true if they are closed that day
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_schedule_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
    UNIQUE(store_id, day_of_week) -- A store can only have one configuration per day
);

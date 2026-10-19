-- Table of services offered by each store
CREATE TABLE IF NOT EXISTS store_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration_minutes INT NOT NULL, -- Service duration (e.g., 30, 45, 60 mins)
    price NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_services_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_store_services_store_id ON store_services(store_id);

-- Table of scheduled appointments
CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL,
    service_id UUID NOT NULL,
    client_id UUID NOT NULL, -- The user who schedules the appointment
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) DEFAULT 'confirmed' NOT NULL, -- confirmed, cancelled, completed
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_appointments_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_appointments_service FOREIGN KEY (service_id) REFERENCES store_services(id) ON DELETE CASCADE,
    CONSTRAINT fk_appointments_client FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Key indicators for quickly checking calendars and avoiding overlaps
CREATE INDEX IF NOT EXISTS idx_appointments_store_date ON appointments(store_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_appointments_client ON appointments(client_id);

DROP TABLE IF EXISTS store_schedule_segments;
DROP TABLE IF EXISTS store_schedules;

CREATE TABLE IF NOT EXISTS store_service_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_service_id UUID NOT NULL,
    day_of_week INT NOT NULL,
    is_closed BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_service_schedule_service FOREIGN KEY (store_service_id) REFERENCES store_services(id) ON DELETE CASCADE,
    UNIQUE(store_service_id, day_of_week)
);

CREATE TABLE IF NOT EXISTS store_service_schedule_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_service_schedule_id UUID NOT NULL,
    open_time TIME NOT NULL,
    close_time TIME NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_service_segment_schedule FOREIGN KEY (store_service_schedule_id) REFERENCES store_service_schedules(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_segment_open_before_close CHECK (open_time < close_time)
);

CREATE INDEX IF NOT EXISTS idx_service_schedule_segments_schedule_id
    ON store_service_schedule_segments (store_service_schedule_id);

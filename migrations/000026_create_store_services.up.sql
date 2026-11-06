-- Services a store offers, each scoped to one of the categories assigned to that store.
CREATE TABLE IF NOT EXISTS store_services (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id         UUID NOT NULL,
    category_id      UUID NOT NULL,
    name             VARCHAR(100) NOT NULL,
    description      TEXT,
    duration_minutes INT NOT NULL,
    price            NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    is_active        BOOLEAN DEFAULT TRUE NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at       TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_store_services_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_store_services_category FOREIGN KEY (category_id) REFERENCES store_categories(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_store_services_store_id ON store_services(store_id);
CREATE INDEX IF NOT EXISTS idx_store_services_category_id ON store_services(category_id);

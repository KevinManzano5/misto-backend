-- Many-to-many assignment of categories to stores.
CREATE TABLE IF NOT EXISTS store_category_assignments (
    store_id    UUID NOT NULL,
    category_id UUID NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (store_id, category_id),
    CONSTRAINT fk_store_category_assignments_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_store_category_assignments_category FOREIGN KEY (category_id) REFERENCES store_categories(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_store_category_assignments_category_id ON store_category_assignments(category_id);

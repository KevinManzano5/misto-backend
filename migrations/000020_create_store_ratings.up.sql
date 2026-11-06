-- Table of client ratings left for a store, one per completed appointment
CREATE TABLE IF NOT EXISTS store_ratings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL,
    store_id       UUID NOT NULL,
    client_id      UUID NOT NULL, -- The user who left the rating
    score          SMALLINT NOT NULL,
    comment        TEXT,
    created_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_ratings_appointment FOREIGN KEY (appointment_id) REFERENCES appointments(id) ON DELETE CASCADE,
    CONSTRAINT fk_ratings_store FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
    CONSTRAINT fk_ratings_client FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_ratings_appointment UNIQUE (appointment_id),
    CONSTRAINT chk_ratings_score CHECK (score BETWEEN 1 AND 5)
);

CREATE INDEX IF NOT EXISTS idx_store_ratings_store_id ON store_ratings(store_id);
CREATE INDEX IF NOT EXISTS idx_store_ratings_client_id ON store_ratings(client_id);

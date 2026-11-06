-- Catalog of store categories (e.g. "Barbershop", "Spa"), administrable directly via SQL
-- (insert/rename/deactivate rows) without requiring a code deploy.
CREATE TABLE IF NOT EXISTS store_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50) NOT NULL UNIQUE,
    slug        VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO store_categories (name, slug) VALUES
    ('Barbershop', 'barbershop'),
    ('Hair Salon', 'hair-salon'),
    ('Spa', 'spa'),
    ('Nail Salon', 'nail-salon'),
    ('Veterinary', 'veterinary'),
    ('Gym', 'gym'),
    ('Dental Clinic', 'dental-clinic'),
    ('Tattoo Studio', 'tattoo-studio');

ALTER TABLE stores ADD COLUMN slug VARCHAR(100);

-- Non-unique: slug is a cosmetic, frontend-URL-building field only (never used for backend
-- lookup/routing), so collisions between differently-named stores are tolerated. Indexed
-- (not unique) to support a possible future "find store by slug" read path cheaply.
CREATE INDEX idx_stores_slug ON stores(slug);

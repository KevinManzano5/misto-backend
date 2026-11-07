ALTER TABLE users
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'member'
    CHECK (role IN ('member', 'admin'));

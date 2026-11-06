ALTER TABLE appointments
    ADD COLUMN payment_method VARCHAR(50) NOT NULL DEFAULT 'cash_on_site',
    ADD COLUMN payment_status VARCHAR(50) NOT NULL DEFAULT 'not_required';

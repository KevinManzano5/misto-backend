-- Fila única (singleton) que permite a un administrador apagar un método de pago para toda la
-- plataforma, independientemente del toggle por-tienda. Default TRUE para preservar el
-- comportamiento actual (ambos métodos habilitados).
CREATE TABLE payment_method_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    cash_on_site_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    card_advance_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO payment_method_settings (id, cash_on_site_enabled, card_advance_enabled)
VALUES (1, TRUE, TRUE);

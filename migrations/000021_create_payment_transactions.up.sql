-- Append-only audit table of every attempt (successful or failed) to charge an appointment's
-- card-in-advance payment. appointments.payment_status remains the source of truth for "is this
-- appointment paid"; this table only adds history/detail alongside it.
CREATE TABLE IF NOT EXISTS payment_transactions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL,
    amount         BIGINT NOT NULL DEFAULT 0, -- smallest currency unit (cents); placeholder until pricing exists
    currency       VARCHAR(3) NOT NULL DEFAULT 'usd',
    status         VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    failure_reason TEXT,
    created_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- RESTRICT (not CASCADE): this is a financial audit trail, it must outlive/block deletion of
    -- the appointment it documents rather than disappear with it.
    CONSTRAINT fk_payment_transactions_appointment FOREIGN KEY (appointment_id) REFERENCES appointments(id) ON DELETE RESTRICT,
    CONSTRAINT chk_payment_transactions_status CHECK (status IN ('paid', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_appointment_id ON payment_transactions(appointment_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_created_at ON payment_transactions(created_at);

-- Aggregate balance a user owes a specific store, accumulated from cash_on_site appointments
-- whose money was never collected at booking time. One row per (user, store) pair.
CREATE TABLE IF NOT EXISTS user_debts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id   UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    balance    BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_user_debts_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT uq_user_debts_user_store UNIQUE (user_id, store_id)
);

-- Append-only ledger of every event that changes a user's debt with a store: opening debt
-- (charge), the store owner confirming it was paid (payment), or an appointment that opened debt
-- being cancelled before collection (reversal). user_debts.balance remains the source of truth
-- for "how much is currently owed"; this table only adds history/detail alongside it.
CREATE TABLE IF NOT EXISTS debt_transactions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    store_id       UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    appointment_id UUID REFERENCES appointments(id) ON DELETE RESTRICT,
    amount         BIGINT NOT NULL,
    type           VARCHAR(20) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_debt_transactions_type CHECK (type IN ('charge', 'payment', 'reversal'))
);

CREATE INDEX IF NOT EXISTS idx_user_debts_user_store ON user_debts(user_id, store_id);
CREATE INDEX IF NOT EXISTS idx_debt_transactions_user_store ON debt_transactions(user_id, store_id);
CREATE INDEX IF NOT EXISTS idx_debt_transactions_appointment_id ON debt_transactions(appointment_id);

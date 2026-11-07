-- Colombian peso (COP) has no decimal subunit in practice (ISO 4217 exponent 0): amounts are
-- always whole numbers. This migration converts store_services.price from NUMERIC(10,2) to an
-- integer type, matching payment_transactions.amount, and standardizes
-- payment_transactions.currency to 'cop', the only currency this app supports.

-- Round any residual decimals before changing the column type (existing values are NUMERIC(10,2)
-- so they are already .00-aligned in the common case, but ROUND guards against any stray value).
ALTER TABLE store_services
    ALTER COLUMN price TYPE BIGINT USING ROUND(price)::BIGINT;

ALTER TABLE store_services
    ALTER COLUMN price SET DEFAULT 0;

COMMENT ON COLUMN store_services.price IS 'Price in whole Colombian pesos (COP); COP has no decimal subunit.';

-- Standardize existing payment_transactions rows and the column default to 'cop'.
UPDATE payment_transactions SET currency = 'cop' WHERE currency <> 'cop';

ALTER TABLE payment_transactions
    ALTER COLUMN currency SET DEFAULT 'cop';

ALTER TABLE payment_transactions
    ADD CONSTRAINT chk_payment_transactions_currency CHECK (currency = 'cop');

COMMENT ON COLUMN payment_transactions.amount IS 'Amount in whole Colombian pesos (COP); COP has no decimal subunit.';

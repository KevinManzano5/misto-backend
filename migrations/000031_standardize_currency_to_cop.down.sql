ALTER TABLE payment_transactions DROP CONSTRAINT IF EXISTS chk_payment_transactions_currency;

ALTER TABLE payment_transactions
    ALTER COLUMN currency SET DEFAULT 'usd';

COMMENT ON COLUMN payment_transactions.amount IS 'Amount is in the smallest currency unit (cents); placeholder until pricing exists.';

COMMENT ON COLUMN store_services.price IS NULL;

ALTER TABLE store_services
    ALTER COLUMN price TYPE NUMERIC(10, 2) USING price::NUMERIC(10, 2);

ALTER TABLE store_services
    ALTER COLUMN price SET DEFAULT 0.00;

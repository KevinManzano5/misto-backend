ALTER TABLE payment_transactions DROP CONSTRAINT chk_payment_transactions_status;
ALTER TABLE payment_transactions ADD CONSTRAINT chk_payment_transactions_status
    CHECK (status IN ('paid', 'failed'));

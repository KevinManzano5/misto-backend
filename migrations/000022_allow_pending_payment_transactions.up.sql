-- Every payment method now gets an initial transaction row at appointment creation: 'paid'/
-- 'failed' when a card charge was attempted immediately, 'pending' otherwise (cash_on_site
-- always, card_advance awaiting manual store confirmation), resolved later by a new row.
ALTER TABLE payment_transactions DROP CONSTRAINT chk_payment_transactions_status;
ALTER TABLE payment_transactions ADD CONSTRAINT chk_payment_transactions_status
    CHECK (status IN ('pending', 'paid', 'failed'));

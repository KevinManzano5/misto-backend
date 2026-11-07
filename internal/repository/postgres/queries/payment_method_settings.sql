-- name: GetPaymentMethodSettings :one
SELECT * FROM payment_method_settings WHERE id = 1;

-- name: UpdatePaymentMethodSettings :one
UPDATE payment_method_settings
SET
    cash_on_site_enabled = COALESCE(sqlc.narg('cash_on_site_enabled'), cash_on_site_enabled),
    card_advance_enabled = COALESCE(sqlc.narg('card_advance_enabled'), card_advance_enabled),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

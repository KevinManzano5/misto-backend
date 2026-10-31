-- name: CreatePaymentTransaction :one
INSERT INTO payment_transactions (
    appointment_id,
    amount,
    currency,
    status,
    payment_method,
    failure_reason
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListPaymentTransactionsByAppointment :many
-- Full charge-attempt history for an appointment, most recent first.
SELECT * FROM payment_transactions
WHERE appointment_id = $1
ORDER BY created_at DESC;

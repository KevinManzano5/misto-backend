-- name: GetUserDebt :one
-- Zero rows is a normal, expected outcome here (the user has never owed this store anything),
-- not a not-found error.
SELECT * FROM user_debts
WHERE user_id = $1 AND store_id = $2
LIMIT 1;

-- name: GetUserDebtForUpdate :one
-- Same as GetUserDebt but locks the row (if it exists) for the duration of the caller's
-- transaction, so a concurrent SettleDebt/AddDebt/ReverseDebt against the same (user, store)
-- cannot read a stale balance.
SELECT * FROM user_debts
WHERE user_id = $1 AND store_id = $2
LIMIT 1
FOR UPDATE;

-- name: IncrementUserDebt :one
-- Upserts the balance: creates the row at `amount` if this is the user's first debt with this
-- store, otherwise adds `amount` to the existing balance.
INSERT INTO user_debts (user_id, store_id, balance)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, store_id) DO UPDATE
SET balance = user_debts.balance + EXCLUDED.balance,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DecrementUserDebt :one
-- Caller is responsible for ensuring amount does not exceed the current balance; the
-- chk_user_debts_balance_non_negative constraint is the last line of defense, not the primary
-- check.
UPDATE user_debts
SET balance = balance - $3,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND store_id = $2
RETURNING *;

-- name: CreateDebtTransaction :one
INSERT INTO debt_transactions (
    user_id,
    store_id,
    appointment_id,
    amount,
    type
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListDebtTransactionsByUserStore :many
-- Full history of debt-affecting events for a user at a store, most recent first.
SELECT * FROM debt_transactions
WHERE user_id = $1 AND store_id = $2
ORDER BY created_at DESC;

-- name: ListDebtorsByStore :many
-- Every user who currently owes this store money, along with their basic profile, for the
-- store owner's debtor list.
SELECT
    d.id,
    d.user_id,
    d.store_id,
    d.balance,
    d.created_at,
    d.updated_at,
    u.username,
    u.email,
    u.first_name,
    u.last_name
FROM user_debts d
JOIN users u ON u.id = d.user_id
WHERE d.store_id = $1
  AND d.balance > 0
ORDER BY d.updated_at DESC;

-- name: ListMyDebts :many
-- Every store the user currently owes money to, along with the store's basic info, for the
-- client's own "my debts" view.
SELECT
    d.id,
    d.user_id,
    d.store_id,
    d.balance,
    d.created_at,
    d.updated_at,
    s.name AS store_name
FROM user_debts d
JOIN stores s ON s.id = d.store_id
WHERE d.user_id = $1
  AND d.balance > 0
ORDER BY d.updated_at DESC;

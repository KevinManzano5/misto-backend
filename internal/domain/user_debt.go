package domain

import "time"

// UserDebt is a user's outstanding balance owed to a specific store, accumulated from
// cash_on_site appointments whose money was never collected. A user may owe different amounts to
// different stores; there is no platform-wide total.
type UserDebt struct {
	ID      string
	UserID  string
	StoreID string
	// Balance is in whole Colombian pesos (COP) and is always >= 0.
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserDebtWithUser pairs a debt with the basic profile of the user who owes it, for a store
// owner's debtor list.
type UserDebtWithUser struct {
	UserDebt
	Username  string
	Email     string
	FirstName string
	LastName  string
}

// UserDebtWithStore pairs a debt with the basic info of the store it is owed to, for a client's
// own "my debts" view.
type UserDebtWithStore struct {
	UserDebt
	StoreName string
}

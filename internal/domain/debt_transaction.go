package domain

import "time"

// Debt transaction type values.
const (
	// DebtTransactionTypeCharge records a cash_on_site appointment opening new debt.
	DebtTransactionTypeCharge = "charge"
	// DebtTransactionTypePayment records the store owner confirming money was collected against
	// an existing debt.
	DebtTransactionTypePayment = "payment"
	// DebtTransactionTypeReversal records debt being removed because the appointment that opened
	// it was cancelled before the money was ever collected.
	DebtTransactionTypeReversal = "reversal"
)

// DebtTransaction records one state-changing event against a user's debt with a store: opening
// debt, paying it off, or reversing it. Rows are immutable once inserted (hence no UpdatedAt),
// the same append-only pattern as PaymentTransaction.
type DebtTransaction struct {
	ID      string
	UserID  string
	StoreID string
	// AppointmentID is the appointment that caused this entry: the one that opened the debt for
	// a charge/reversal. Empty for a payment entry, which settles the aggregate balance rather
	// than a single appointment.
	AppointmentID string
	// Amount is in whole Colombian pesos (COP).
	Amount    int64
	Type      string
	CreatedAt time.Time
}

// NewDebtTransaction constructor. amount is in whole COP.
func NewDebtTransaction(userID, storeID, appointmentID string, amount int64, transactionType string) *DebtTransaction {
	return &DebtTransaction{
		UserID:        userID,
		StoreID:       storeID,
		AppointmentID: appointmentID,
		Amount:        amount,
		Type:          transactionType,
		CreatedAt:     time.Now(),
	}
}

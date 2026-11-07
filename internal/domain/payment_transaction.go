package domain

import "time"

// PaymentTransaction records the lifecycle of an appointment's payment, one row per
// state-changing event: every appointment gets an initial row the moment it's created (pending,
// or already paid/failed when a card charge was attempted immediately), and a later confirmation
// or charge attempt appends another row rather than updating the first — a failed attempt
// followed by a successful retry is two rows, not an update of one. Rows are immutable once
// inserted (hence no UpdatedAt).
type PaymentTransaction struct {
	ID            string
	AppointmentID string
	// Amount is in whole Colombian pesos (COP); COP has no decimal subunit. It is the service's
	// price plus the store's app usage fee percentage (see ApplyPercentageFee), i.e. what the
	// client actually pays, not what the store receives.
	Amount   int64
	Currency string
	// Status reuses PaymentStatusPending/PaymentStatusPaid/PaymentStatusFailed.
	Status string
	// PaymentMethod reuses PaymentMethodCashOnSite/PaymentMethodCardAdvance.
	PaymentMethod string
	// FailureReason is the processor's error message when Status == PaymentStatusFailed, empty
	// otherwise.
	FailureReason string
	CreatedAt     time.Time
}

// defaultPaymentTransactionCurrency is the only currency this app supports (Colombian pesos).
const defaultPaymentTransactionCurrency = "cop"

// NewPaymentTransaction constructor. amount is in whole COP (see ApplyPercentageFee).
func NewPaymentTransaction(appointmentID string, amount int64, status, paymentMethod, failureReason string) *PaymentTransaction {
	return &PaymentTransaction{
		AppointmentID: appointmentID,
		Amount:        amount,
		Currency:      defaultPaymentTransactionCurrency,
		Status:        status,
		PaymentMethod: paymentMethod,
		FailureReason: failureReason,
		CreatedAt:     time.Now(),
	}
}

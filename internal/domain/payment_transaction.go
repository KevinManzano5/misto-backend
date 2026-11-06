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
	// Amount is in the smallest currency unit (cents). Always 0 today: no pricing model exists
	// yet, this is a placeholder for a future pricing iteration, not a bug.
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

// defaultPaymentTransactionCurrency is the placeholder currency used until a real pricing model
// introduces multi-currency support.
const defaultPaymentTransactionCurrency = "usd"

// NewPaymentTransaction constructor. amountCents is always 0 until a real pricing model exists.
func NewPaymentTransaction(appointmentID string, amountCents int64, status, paymentMethod, failureReason string) *PaymentTransaction {
	return &PaymentTransaction{
		AppointmentID: appointmentID,
		Amount:        amountCents,
		Currency:      defaultPaymentTransactionCurrency,
		Status:        status,
		PaymentMethod: paymentMethod,
		FailureReason: failureReason,
		CreatedAt:     time.Now(),
	}
}

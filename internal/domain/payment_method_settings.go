package domain

import "time"

// PaymentMethodSettings controls, at the platform level, which payment methods can be used by
// any store. This is independent of a store's own per-store toggle (Store.CashOnSiteEnabled /
// CardAdvanceEnabled): a payment method must be enabled both globally and for the store for a
// client to use it when booking an appointment.
type PaymentMethodSettings struct {
	CashOnSiteEnabled  bool
	CardAdvanceEnabled bool
	UpdatedAt          time.Time
}

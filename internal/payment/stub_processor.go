// Package payment contains payment processing implementations.
package payment

import (
	"context"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StubProcessor is a placeholder PaymentProcessor that always approves a charge. It exists so the
// appointment payment flow (when a charge is attempted, how a failure is handled) is fully wired
// end-to-end before a real payment gateway (e.g. Stripe) is integrated; swapping it out later
// requires no change to AppointmentUseCase.
type StubProcessor struct{}

// NewStubProcessor constructor
func NewStubProcessor() *StubProcessor {
	return &StubProcessor{}
}

// Charge always succeeds; replace with a real payment gateway integration.
func (p *StubProcessor) Charge(_ context.Context, _ *domain.Appointment) error {
	return nil
}

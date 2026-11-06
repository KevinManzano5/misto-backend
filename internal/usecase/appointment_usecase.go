package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// AppointmentRepository interface
type AppointmentRepository interface {
	// CreateAppointment atomically checks for a scheduling conflict and inserts the appointment.
	CreateAppointment(ctx context.Context, appointment *domain.Appointment) error
	ListAppointmentsByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error)
	ListAllAppointmentsByStoreWithClient(ctx context.Context, storeID string, startTime, endTime *time.Time) ([]*domain.AppointmentWithClient, error)
	ListAppointmentsByClient(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error)
	GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error)
	GetActiveAppointmentByStoreAndClient(ctx context.Context, storeID, clientID string) (*domain.Appointment, error)
	UpdateAppointmentStatus(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error)
	UpdatePaymentStatus(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error)
	ExpireOverdueAppointments(ctx context.Context, before time.Time) ([]*domain.Appointment, error)
}

// PaymentProcessor charges a card payment made in advance for an appointment.
type PaymentProcessor interface {
	Charge(ctx context.Context, appointment *domain.Appointment) error
}

// StoreScheduleRepository interface (Added for checking availability)
type StoreScheduleRepository interface {
	GetStoreScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error)
}

// AppointmentStoreRepository interface (narrowed to what AppointmentUseCase needs from a store)
type AppointmentStoreRepository interface {
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
}

// AppointmentStoreServiceRepository interface (narrowed to what AppointmentUseCase needs from a
// store service)
type AppointmentStoreServiceRepository interface {
	GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error)
}

// PaymentTransactionRepository interface (narrowed to what AppointmentUseCase needs to record and
// list charge attempts)
type PaymentTransactionRepository interface {
	CreatePaymentTransaction(ctx context.Context, transaction *domain.PaymentTransaction) error
	ListPaymentTransactionsByAppointment(ctx context.Context, appointmentID string) ([]*domain.PaymentTransaction, error)
}

// AppointmentUseCase struct
type AppointmentUseCase struct {
	repo                   AppointmentRepository
	scheduleRepo           StoreScheduleRepository
	storeRepo              AppointmentStoreRepository
	storeServiceRepo       AppointmentStoreServiceRepository
	paymentProcessor       PaymentProcessor
	paymentTransactionRepo PaymentTransactionRepository
	log                    *slog.Logger
}

// NewAppointmentUseCase constructor
func NewAppointmentUseCase(repo AppointmentRepository, scheduleRepo StoreScheduleRepository, storeRepo AppointmentStoreRepository, storeServiceRepo AppointmentStoreServiceRepository, paymentProcessor PaymentProcessor, paymentTransactionRepo PaymentTransactionRepository, log *slog.Logger) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:                   repo,
		scheduleRepo:           scheduleRepo,
		storeRepo:              storeRepo,
		storeServiceRepo:       storeServiceRepo,
		paymentProcessor:       paymentProcessor,
		paymentTransactionRepo: paymentTransactionRepo,
		log:                    log,
	}
}

// resolveService loads serviceID and verifies it belongs to storeID and is currently active, so
// slot/appointment duration always reflects a real, currently-offered service of this store.
func (usecase *AppointmentUseCase) resolveService(ctx context.Context, storeID, serviceID string) (*domain.StoreService, error) {
	service, err := usecase.storeServiceRepo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: service not found", domain.ErrInvalidInput)
		}
		return nil, fmt.Errorf("getting store service: %w", err)
	}

	if service.StoreID != storeID {
		return nil, fmt.Errorf("%w: service does not belong to this store", domain.ErrInvalidInput)
	}

	if !service.IsActive {
		return nil, fmt.Errorf("%w: service is not currently offered", domain.ErrInvalidInput)
	}

	return service, nil
}

// recordPaymentTransaction best-effort-records the outcome of a Charge call as a new, immutable
// payment_transactions row. A failure here is logged but intentionally does not abort the
// caller's flow: appointments.payment_status (the source of truth) has already been decided
// independently of this write, and failing a user's booking/confirmation over an audit-log
// failure would be worse than a best-effort gap in the history.
func (usecase *AppointmentUseCase) recordPaymentTransaction(ctx context.Context, appointmentID, paymentMethod, status, failureReason string) {
	transaction := domain.NewPaymentTransaction(appointmentID, 0, status, paymentMethod, failureReason)
	if err := usecase.paymentTransactionRepo.CreatePaymentTransaction(ctx, transaction); err != nil {
		usecase.log.Error("failed to record payment transaction", slog.String("appointment_id", appointmentID), slog.Any("error", err))
	}
}

// storeLocation loads store's IANA timezone, used to interpret its schedule's open/close hours
// and to resolve which calendar day/weekday an instant falls on. It falls back to UTC
// defensively if the stored value somehow isn't loadable (StoreUseCase validates it on write).
func storeLocation(store *domain.Store) *time.Location {
	loc, err := time.LoadLocation(store.Timezone)
	if err != nil {
		return time.UTC
	}

	return loc
}

// Create method
func (usecase *AppointmentUseCase) Create(ctx context.Context, storeID, serviceID, userID, paymentMethod string, startTime time.Time) (*domain.Appointment, error) {
	if paymentMethod != domain.PaymentMethodCashOnSite && paymentMethod != domain.PaymentMethodCardAdvance {
		return nil, fmt.Errorf("%w: payment method must be %q or %q", domain.ErrInvalidInput, domain.PaymentMethodCashOnSite, domain.PaymentMethodCardAdvance)
	}

	if startTime.Before(time.Now()) {
		return nil, fmt.Errorf("%w: cannot schedule an appointment in the past", domain.ErrInvalidInput)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	service, err := usecase.resolveService(ctx, storeID, serviceID)
	if err != nil {
		return nil, err
	}

	if !store.AllowSameDayAppointments && isSameLocalDay(startTime, time.Now(), storeLocation(store)) {
		return nil, fmt.Errorf("%w: this store does not accept same-day appointments", domain.ErrInvalidInput)
	}

	endTime := startTime.Add(time.Duration(service.DurationMinutes) * time.Minute)

	if err := usecase.validateWithinStoreSchedule(ctx, store, startTime, endTime); err != nil {
		return nil, err
	}

	status := domain.AppointmentStatusPending
	if store.AutoConfirmAppointments {
		status = domain.AppointmentStatusConfirmed
	}

	// Every payment method starts pending: a cash_on_site payment stays pending until the store
	// owner confirms it was received (see ConfirmCashPayment); a card_advance payment stays
	// pending until a charge is attempted, immediately below or later at manual confirmation
	// (see UpdateStatus).
	paymentStatus := domain.PaymentStatusPending

	appointment := domain.NewAppointment(storeID, serviceID, userID, status, paymentMethod, paymentStatus, startTime, endTime)

	// Auto-confirming stores charge the card immediately; a store that reviews appointments
	// manually only charges when it confirms (see UpdateStatus).
	chargeAttempted := false
	var chargeErr error
	if paymentMethod == domain.PaymentMethodCardAdvance && store.AutoConfirmAppointments {
		chargeAttempted = true
		chargeErr = usecase.paymentProcessor.Charge(ctx, appointment)
		if chargeErr != nil {
			appointment.Status = domain.AppointmentStatusPending
			appointment.PaymentStatus = domain.PaymentStatusFailed
		} else {
			appointment.PaymentStatus = domain.PaymentStatusPaid
		}
	}

	if err := usecase.repo.CreateAppointment(ctx, appointment); err != nil {
		return nil, fmt.Errorf("creating appointment: %w", err)
	}

	// Every new appointment gets exactly one initial transaction row, recorded only after the
	// insert above since it references appointment.ID: the resolved outcome when a charge was
	// attempted immediately, or a pending placeholder otherwise, that a later action
	// (ConfirmCashPayment or UpdateStatus) resolves with a new row.
	transactionStatus, failureReason := domain.PaymentStatusPending, ""
	if chargeAttempted {
		transactionStatus = domain.PaymentStatusPaid
		if chargeErr != nil {
			transactionStatus, failureReason = domain.PaymentStatusFailed, chargeErr.Error()
		}
	}
	usecase.recordPaymentTransaction(ctx, appointment.ID, paymentMethod, transactionStatus, failureReason)

	return appointment, nil
}

// ConfirmCashPayment marks a cash_on_site appointment's payment as received. Only the store
// owner may call this, and only while the payment is still pending: a card_advance appointment,
// or one already paid/failed, is rejected with domain.ErrInvalidInput since there's nothing to
// confirm this way.
func (usecase *AppointmentUseCase) ConfirmCashPayment(ctx context.Context, appointmentID, requesterUserID string) (*domain.Appointment, error) {
	appointment, err := usecase.repo.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("getting appointment: %w", err)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, appointment.StoreID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can confirm a cash payment", domain.ErrForbidden)
	}

	if appointment.PaymentMethod != domain.PaymentMethodCashOnSite {
		return nil, fmt.Errorf("%w: only a cash_on_site appointment's payment can be confirmed this way", domain.ErrInvalidInput)
	}

	if appointment.PaymentStatus != domain.PaymentStatusPending {
		return nil, fmt.Errorf("%w: payment is not pending confirmation", domain.ErrInvalidInput)
	}

	updated, err := usecase.repo.UpdatePaymentStatus(ctx, appointmentID, domain.PaymentStatusPaid)
	if err != nil {
		return nil, fmt.Errorf("confirming cash payment: %w", err)
	}

	usecase.recordPaymentTransaction(ctx, appointmentID, domain.PaymentMethodCashOnSite, domain.PaymentStatusPaid, "")

	return updated, nil
}

// validateWithinStoreSchedule ensures the requested [startTime, endTime) range falls entirely
// within the store's configured opening hours for that day of the week, with the day of week and
// opening hours resolved in the store's own timezone rather than whatever offset the client's
// startTime happened to carry.
func (usecase *AppointmentUseCase) validateWithinStoreSchedule(ctx context.Context, store *domain.Store, startTime, endTime time.Time) error {
	localStart := startTime.In(storeLocation(store))
	dayOfWeek := int32(localStart.Weekday())

	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, store.ID, dayOfWeek)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("%w: store has no schedule configured for this day", domain.ErrInvalidInput)
		}
		return fmt.Errorf("getting store schedule: %w", err)
	}

	if schedule.IsClosed || schedule.OpenTime == "" || schedule.CloseTime == "" {
		return fmt.Errorf("%w: store is closed on this day", domain.ErrInvalidInput)
	}

	openTime, err := parseScheduleClockTime(localStart, schedule.OpenTime)
	if err != nil {
		return fmt.Errorf("parsing store open time: %w", err)
	}

	closeTime, err := parseScheduleClockTime(localStart, schedule.CloseTime)
	if err != nil {
		return fmt.Errorf("parsing store close time: %w", err)
	}

	if startTime.Before(openTime) || endTime.After(closeTime) {
		return fmt.Errorf("%w: appointment falls outside store opening hours", domain.ErrInvalidInput)
	}

	return nil
}

// isSameLocalDay reports whether a and b fall on the same calendar day in loc.
func isSameLocalDay(a, b time.Time, loc *time.Location) bool {
	a, b = a.In(loc), b.In(loc)
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// parseScheduleClockTime combines a "HH:MM" or "HH:MM:SS" clock string (as stored on a
// StoreSchedule) with the date portion of date, in date's location.
func parseScheduleClockTime(date time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", clock)
	if err != nil {
		parsed, err = time.Parse("15:04", clock)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w: invalid schedule time %q", domain.ErrInvalidInput, clock)
		}
	}

	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, date.Location()), nil
}

// ListByStoreDate method
func (usecase *AppointmentUseCase) ListByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error) {
	appointments, err := usecase.repo.ListAppointmentsByStoreDate(ctx, storeID, startTime, endTime)

	if err != nil {
		return nil, fmt.Errorf("listing appointments: %w", err)
	}

	return appointments, nil
}

// daySchedule bundles what both GetAvailableSlots and GetBookedAppointments need to resolve a
// store's opening hours and booked appointments for a specific calendar date, in the store's
// own timezone.
type daySchedule struct {
	location        *time.Location
	targetDate      time.Time
	durationMinutes int32
	openTime        time.Time
	closeTime       time.Time
	isOpen          bool
	appointments    []*domain.Appointment
}

// resolveDaySchedule parses dateStr as a calendar date in the store's own timezone (not UTC),
// loads the store's default appointment duration, its configured opening hours for that date's
// day of week, and the appointments already booked that day. isOpen is false (with zero-value
// times) when the store has no schedule for that day or is explicitly closed, which callers
// treat as "no availability" rather than an error.
func (usecase *AppointmentUseCase) resolveDaySchedule(ctx context.Context, storeID, serviceID, dateStr string) (*daySchedule, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	service, err := usecase.resolveService(ctx, storeID, serviceID)
	if err != nil {
		return nil, err
	}

	loc := storeLocation(store)

	targetDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	today := now.In(loc)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	if targetDate.Before(todayDate) {
		return nil, fmt.Errorf("%w: cannot get availability for a past date", domain.ErrInvalidInput)
	}

	if !store.AllowSameDayAppointments && isSameLocalDay(targetDate, now, loc) {
		return &daySchedule{location: loc, targetDate: targetDate, durationMinutes: service.DurationMinutes}, nil
	}

	dayOfWeek := int32(targetDate.Weekday())
	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, storeID, dayOfWeek)
	if err != nil {
		// If there is no schedule in the DB for that day, we assume it is closed.
		if errors.Is(err, domain.ErrNotFound) {
			return &daySchedule{location: loc, targetDate: targetDate, durationMinutes: service.DurationMinutes}, nil
		}
		return nil, fmt.Errorf("getting store schedule: %w", err)
	}

	if schedule.IsClosed || schedule.OpenTime == "" || schedule.CloseTime == "" {
		return &daySchedule{location: loc, targetDate: targetDate, durationMinutes: service.DurationMinutes}, nil
	}

	openTime, err := parseScheduleClockTime(targetDate, schedule.OpenTime)
	if err != nil {
		return nil, fmt.Errorf("parsing store open time: %w", err)
	}

	closeTime, err := parseScheduleClockTime(targetDate, schedule.CloseTime)
	if err != nil {
		return nil, fmt.Errorf("parsing store close time: %w", err)
	}

	startOfDay := targetDate
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	appointments, err := usecase.repo.ListAppointmentsByStoreDate(ctx, storeID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("listing appointments for availability: %w", err)
	}

	return &daySchedule{
		location:        loc,
		targetDate:      targetDate,
		durationMinutes: service.DurationMinutes,
		openTime:        openTime,
		closeTime:       closeTime,
		isOpen:          true,
		appointments:    appointments,
	}, nil
}

// availableSlots computes the free start times for day, in "HH:MM" format (the store's local
// time), skipping slots that overlap an existing appointment or that have already passed (when
// day is today in the store's timezone).
func (day *daySchedule) availableSlots() []string {
	availableSlots := make([]string, 0)
	if !day.isOpen {
		return availableSlots
	}

	slotDuration := time.Duration(day.durationMinutes) * time.Minute
	currentSlot := day.openTime
	now := time.Now().In(day.location)

	for currentSlot.Add(slotDuration).Compare(day.closeTime) <= 0 {
		slotEnd := currentSlot.Add(slotDuration)
		isAvailable := true

		// If the selected schedule is for today (in the store's timezone), omit hours already passed.
		if day.targetDate.YearDay() == now.YearDay() && day.targetDate.Year() == now.Year() && currentSlot.Before(now) {
			isAvailable = false
		}

		if isAvailable {
			for _, appointment := range day.appointments {
				// A conflict occurs if the time slot begins before the appointment ends, and ends after the appointment begins.
				if currentSlot.Before(appointment.EndTime) && slotEnd.After(appointment.StartTime) {
					isAvailable = false
					break
				}
			}
		}

		if isAvailable {
			availableSlots = append(availableSlots, currentSlot.Format("15:04"))
		}

		currentSlot = currentSlot.Add(slotDuration)
	}

	return availableSlots
}

// GetAvailableSlots method en AppointmentUseCase
func (usecase *AppointmentUseCase) GetAvailableSlots(ctx context.Context, storeID, serviceID, dateStr string) ([]string, error) {
	day, err := usecase.resolveDaySchedule(ctx, storeID, serviceID, dateStr)
	if err != nil {
		return nil, err
	}

	return day.availableSlots(), nil
}

// GetBookedAppointments returns every appointment ever booked at storeID, any status, each
// paired with the basic profile of the client who booked it. Only the store's owner
// (requesterUserID) may see this: anyone else gets domain.ErrForbidden.
//
// startDateStr/endDateStr are optional "YYYY-MM-DD" dates, interpreted as a day boundary in the
// store's own timezone, that restrict results to appointments starting within that inclusive
// range; an empty string leaves that side of the range open.
func (usecase *AppointmentUseCase) GetBookedAppointments(ctx context.Context, storeID, requesterUserID, startDateStr, endDateStr string) ([]*domain.AppointmentWithClient, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can view its booked appointments", domain.ErrForbidden)
	}

	startTime, endTime, err := parseBookedAppointmentsDateRange(storeLocation(store), startDateStr, endDateStr)
	if err != nil {
		return nil, err
	}

	appointments, err := usecase.repo.ListAllAppointmentsByStoreWithClient(ctx, storeID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("listing booked appointments: %w", err)
	}

	return appointments, nil
}

// parseBookedAppointmentsDateRange parses the optional "YYYY-MM-DD" start/end query dates for
// GetBookedAppointments in loc, expanding each to the start/end instant of that calendar day so
// the range covers the whole day. Either string may be empty to leave that side of the range
// open; a nil *time.Time result means "no bound".
func parseBookedAppointmentsDateRange(loc *time.Location, startDateStr, endDateStr string) (startTime, endTime *time.Time, err error) {
	if startDateStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDateStr, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid start_date, expected YYYY-MM-DD", domain.ErrInvalidInput)
		}
		startTime = &parsed
	}

	if endDateStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDateStr, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid end_date, expected YYYY-MM-DD", domain.ErrInvalidInput)
		}
		endOfDay := parsed.Add(24 * time.Hour).Add(-time.Nanosecond)
		endTime = &endOfDay
	}

	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return nil, nil, fmt.Errorf("%w: end_date must not be before start_date", domain.ErrInvalidInput)
	}

	return startTime, endTime, nil
}

// ListMyAppointments returns every appointment ever booked by userID, any status, most recent
// first, each paired with the basic info of the store it was booked at.
func (usecase *AppointmentUseCase) ListMyAppointments(ctx context.Context, userID string) ([]*domain.AppointmentWithStore, error) {
	appointments, err := usecase.repo.ListAppointmentsByClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing appointments by client: %w", err)
	}

	return appointments, nil
}

// GetActiveAppointmentStatus reports whether userID currently has a pending/confirmed
// appointment at storeID. It returns (nil, nil) when none exists — an expected outcome, not an
// error. No authorization check beyond "userID is the authenticated caller" is needed: userID
// always comes from middlewares.GetUserID, never a path/query parameter, so unlike
// GetBookedAppointments/Cancel there is no "someone else's identity" to guard against.
func (usecase *AppointmentUseCase) GetActiveAppointmentStatus(ctx context.Context, storeID, userID string) (*domain.Appointment, error) {
	appointment, err := usecase.repo.GetActiveAppointmentByStoreAndClient(ctx, storeID, userID)
	if err != nil {
		return nil, fmt.Errorf("getting active appointment status: %w", err)
	}

	return appointment, nil
}

// Cancel moves a pending or confirmed appointment to cancelled. Either the client who booked it
// (requesterUserID matches the appointment's UserID) or the store's owner may cancel it; anyone
// else gets domain.ErrForbidden. An appointment that is already cancelled/completed/expired is
// rejected with domain.ErrInvalidInput since there's nothing left to cancel. A non-empty reason
// is required so cancellations are always auditable.
func (usecase *AppointmentUseCase) Cancel(ctx context.Context, appointmentID, requesterUserID, reason string) (*domain.Appointment, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: a cancellation reason is required", domain.ErrInvalidInput)
	}

	appointment, err := usecase.repo.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("getting appointment: %w", err)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, appointment.StoreID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if requesterUserID != appointment.UserID && requesterUserID != store.UserID {
		return nil, fmt.Errorf("%w: only the client who booked it or the store owner can cancel this appointment", domain.ErrForbidden)
	}

	if appointment.Status != domain.AppointmentStatusPending && appointment.Status != domain.AppointmentStatusConfirmed {
		return nil, fmt.Errorf("%w: only a pending or confirmed appointment can be cancelled", domain.ErrInvalidInput)
	}

	cancelled, err := usecase.repo.UpdateAppointmentStatus(ctx, appointmentID, domain.AppointmentStatusCancelled, reason)
	if err != nil {
		return nil, fmt.Errorf("cancelling appointment: %w", err)
	}

	return cancelled, nil
}

// UpdateStatus moves a pending appointment to confirmed (approved) or cancelled (rejected).
// Only the store's owner (requesterUserID) may do this, and only while the appointment is still
// pending: anyone else gets domain.ErrForbidden, and any other starting status is rejected with
// domain.ErrInvalidInput since it's no longer awaiting a decision. Rejecting (newStatus ==
// cancelled) requires a non-empty reason, auditable like any other cancellation.
func (usecase *AppointmentUseCase) UpdateStatus(ctx context.Context, appointmentID, requesterUserID, newStatus, reason string) (*domain.Appointment, error) {
	if newStatus != domain.AppointmentStatusConfirmed && newStatus != domain.AppointmentStatusCancelled {
		return nil, fmt.Errorf("%w: status must be %q or %q", domain.ErrInvalidInput, domain.AppointmentStatusConfirmed, domain.AppointmentStatusCancelled)
	}

	if newStatus == domain.AppointmentStatusCancelled && strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: a cancellation reason is required", domain.ErrInvalidInput)
	}

	appointment, err := usecase.repo.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("getting appointment: %w", err)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, appointment.StoreID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can review this appointment", domain.ErrForbidden)
	}

	if appointment.Status != domain.AppointmentStatusPending {
		return nil, fmt.Errorf("%w: appointment is not pending approval", domain.ErrInvalidInput)
	}

	if newStatus == domain.AppointmentStatusConfirmed &&
		appointment.PaymentMethod == domain.PaymentMethodCardAdvance &&
		appointment.PaymentStatus != domain.PaymentStatusPaid {
		chargeErr := usecase.paymentProcessor.Charge(ctx, appointment)

		if chargeErr != nil {
			usecase.recordPaymentTransaction(ctx, appointmentID, appointment.PaymentMethod, domain.PaymentStatusFailed, chargeErr.Error())

			if _, updateErr := usecase.repo.UpdatePaymentStatus(ctx, appointmentID, domain.PaymentStatusFailed); updateErr != nil {
				return nil, fmt.Errorf("recording failed payment: %w", updateErr)
			}
			return nil, fmt.Errorf("charging card payment: %w", domain.ErrPaymentFailed)
		}

		usecase.recordPaymentTransaction(ctx, appointmentID, appointment.PaymentMethod, domain.PaymentStatusPaid, "")

		if _, err := usecase.repo.UpdatePaymentStatus(ctx, appointmentID, domain.PaymentStatusPaid); err != nil {
			return nil, fmt.Errorf("recording payment: %w", err)
		}
	}

	updated, err := usecase.repo.UpdateAppointmentStatus(ctx, appointmentID, newStatus, reason)
	if err != nil {
		return nil, fmt.Errorf("updating appointment status: %w", err)
	}

	return updated, nil
}

// ListPaymentTransactions returns the full charge-attempt history for an appointment, most
// recent first. Only the client who booked it or the store owner may view it.
func (usecase *AppointmentUseCase) ListPaymentTransactions(ctx context.Context, appointmentID, requesterUserID string) ([]*domain.PaymentTransaction, error) {
	appointment, err := usecase.repo.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("getting appointment: %w", err)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, appointment.StoreID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if requesterUserID != appointment.UserID && requesterUserID != store.UserID {
		return nil, fmt.Errorf("%w: only the client who booked it or the store owner can view its payment transactions", domain.ErrForbidden)
	}

	transactions, err := usecase.paymentTransactionRepo.ListPaymentTransactionsByAppointment(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("listing payment transactions: %w", err)
	}

	return transactions, nil
}

// ExpireOverdueAppointments moves pending/confirmed appointments whose end_time has already
// passed to a terminal status (completed/expired) and returns how many were changed. It is a
// system action with no requester to authorize against, meant to be called by a periodic job
// rather than from an HTTP handler.
func (usecase *AppointmentUseCase) ExpireOverdueAppointments(ctx context.Context) (int, error) {
	expired, err := usecase.repo.ExpireOverdueAppointments(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("expiring overdue appointments: %w", err)
	}

	return len(expired), nil
}

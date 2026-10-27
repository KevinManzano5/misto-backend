package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// AppointmentRepository interface
type AppointmentRepository interface {
	// CreateAppointment atomically checks for a scheduling conflict and inserts the appointment.
	CreateAppointment(ctx context.Context, appointment *domain.Appointment) error
	ListAppointmentsByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error)
}

// StoreScheduleRepository interface (Added for checking availability)
type StoreScheduleRepository interface {
	GetStoreScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error)
}

// AppointmentStoreRepository interface (narrowed to what AppointmentUseCase needs from a store)
type AppointmentStoreRepository interface {
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
}

// AppointmentUseCase struct
type AppointmentUseCase struct {
	repo         AppointmentRepository
	scheduleRepo StoreScheduleRepository
	storeRepo    AppointmentStoreRepository
}

// NewAppointmentUseCase constructor
func NewAppointmentUseCase(repo AppointmentRepository, scheduleRepo StoreScheduleRepository, storeRepo AppointmentStoreRepository) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:         repo,
		scheduleRepo: scheduleRepo,
		storeRepo:    storeRepo,
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
func (usecase *AppointmentUseCase) Create(ctx context.Context, storeID, userID string, startTime time.Time) (*domain.Appointment, error) {
	if startTime.Before(time.Now()) {
		return nil, fmt.Errorf("%w: cannot schedule an appointment in the past", domain.ErrInvalidInput)
	}

	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	endTime := startTime.Add(time.Duration(store.AppointmentDuration) * time.Minute)

	if err := usecase.validateWithinStoreSchedule(ctx, store, startTime, endTime); err != nil {
		return nil, err
	}

	appointment := domain.NewAppointment(storeID, userID, "", startTime, endTime)

	if err := usecase.repo.CreateAppointment(ctx, appointment); err != nil {
		return nil, fmt.Errorf("creating appointment: %w", err)
	}

	return appointment, nil
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
func (usecase *AppointmentUseCase) resolveDaySchedule(ctx context.Context, storeID, dateStr string) (*daySchedule, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	loc := storeLocation(store)

	targetDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	dayOfWeek := int32(targetDate.Weekday())
	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, storeID, dayOfWeek)
	if err != nil {
		// If there is no schedule in the DB for that day, we assume it is closed.
		if errors.Is(err, domain.ErrNotFound) {
			return &daySchedule{location: loc, targetDate: targetDate, durationMinutes: store.AppointmentDuration}, nil
		}
		return nil, fmt.Errorf("getting store schedule: %w", err)
	}

	if schedule.IsClosed || schedule.OpenTime == "" || schedule.CloseTime == "" {
		return &daySchedule{location: loc, targetDate: targetDate, durationMinutes: store.AppointmentDuration}, nil
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
		durationMinutes: store.AppointmentDuration,
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
func (usecase *AppointmentUseCase) GetAvailableSlots(ctx context.Context, storeID, dateStr string) ([]string, error) {
	day, err := usecase.resolveDaySchedule(ctx, storeID, dateStr)
	if err != nil {
		return nil, err
	}

	return day.availableSlots(), nil
}

// GetBookedAppointments returns the appointments already reserved at storeID on dateStr (a
// calendar date interpreted in the store's own timezone). Only the store's owner
// (requesterUserID) may see this: anyone else gets domain.ErrForbidden.
func (usecase *AppointmentUseCase) GetBookedAppointments(ctx context.Context, storeID, requesterUserID, dateStr string) ([]*domain.Appointment, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can view its booked appointments", domain.ErrForbidden)
	}

	targetDate, err := time.ParseInLocation("2006-01-02", dateStr, storeLocation(store))
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	startOfDay := targetDate
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	appointments, err := usecase.ListByStoreDate(ctx, storeID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	if appointments == nil {
		appointments = make([]*domain.Appointment, 0)
	}

	return appointments, nil
}

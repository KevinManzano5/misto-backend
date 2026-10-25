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

// AppointmentUseCase struct
type AppointmentUseCase struct {
	repo         AppointmentRepository
	scheduleRepo StoreScheduleRepository
	serviceRepo  StoreServiceRepository
}

// NewAppointmentUseCase constructor
func NewAppointmentUseCase(repo AppointmentRepository, scheduleRepo StoreScheduleRepository, serviceRepo StoreServiceRepository) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:         repo,
		scheduleRepo: scheduleRepo,
		serviceRepo:  serviceRepo,
	}
}

// Create method
func (usecase *AppointmentUseCase) Create(ctx context.Context, storeID, serviceID, userID string, startTime time.Time) (*domain.Appointment, error) {
	if startTime.Before(time.Now()) {
		return nil, fmt.Errorf("%w: cannot schedule an appointment in the past", domain.ErrInvalidInput)
	}

	service, err := usecase.serviceRepo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting store service: %w", err)
	}

	endTime := startTime.Add(time.Duration(service.DurationMinutes) * time.Minute)

	if err := usecase.validateWithinStoreSchedule(ctx, storeID, startTime, endTime); err != nil {
		return nil, err
	}

	appointment := domain.NewAppointment(storeID, serviceID, userID, "", startTime, endTime)

	if err := usecase.repo.CreateAppointment(ctx, appointment); err != nil {
		return nil, fmt.Errorf("creating appointment: %w", err)
	}

	return appointment, nil
}

// validateWithinStoreSchedule ensures the requested [startTime, endTime) range falls entirely
// within the store's configured opening hours for that day of the week.
func (usecase *AppointmentUseCase) validateWithinStoreSchedule(ctx context.Context, storeID string, startTime, endTime time.Time) error {
	dayOfWeek := int32(startTime.Weekday())

	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, storeID, dayOfWeek)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("%w: store has no schedule configured for this day", domain.ErrInvalidInput)
		}
		return fmt.Errorf("getting store schedule: %w", err)
	}

	if schedule.IsClosed || schedule.OpenTime == "" || schedule.CloseTime == "" {
		return fmt.Errorf("%w: store is closed on this day", domain.ErrInvalidInput)
	}

	openTime, err := parseScheduleClockTime(startTime, schedule.OpenTime)
	if err != nil {
		return fmt.Errorf("parsing store open time: %w", err)
	}

	closeTime, err := parseScheduleClockTime(startTime, schedule.CloseTime)
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

// GetAvailableSlots method en AppointmentUseCase
func (usecase *AppointmentUseCase) GetAvailableSlots(ctx context.Context, storeID string, durationMinutes int32, dateStr string) ([]string, error) {
	// 1. Parse the requested date (assuming YYYY-MM-DD format)
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// 2. Check the schedule configured for that day of the week
	dayOfWeek := int32(targetDate.Weekday())
	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, storeID, dayOfWeek)

	if err != nil {
		// If there is no schedule in the DB for that day, we assume it is closed (we return an empty array).
		if errors.Is(err, domain.ErrNotFound) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("getting store schedule: %w", err)
	}

	// If the day is explicitly marked as closed or has no hours, there is no availability.
	if schedule.IsClosed || schedule.OpenTime == "" || schedule.CloseTime == "" {
		return []string{}, nil
	}

	// 3. Convert the schedule text strings into actual date objects for that day
	openTime, err := parseScheduleClockTime(targetDate, schedule.OpenTime)
	if err != nil {
		return nil, fmt.Errorf("parsing store open time: %w", err)
	}

	closeTime, err := parseScheduleClockTime(targetDate, schedule.CloseTime)
	if err != nil {
		return nil, fmt.Errorf("parsing store close time: %w", err)
	}

	// 4. Get the appointments scheduled for that ENTIRE day
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	appointments, err := usecase.repo.ListAppointmentsByStoreDate(ctx, storeID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("listing appointments for availability: %w", err)
	}

	var availableSlots []string
	slotDuration := time.Duration(durationMinutes) * time.Minute
	currentSlot := openTime
	now := time.Now()

	// 5. Generate intervals and filter out those that overlap or have already passed
	for currentSlot.Add(slotDuration).Compare(closeTime) <= 0 {
		slotEnd := currentSlot.Add(slotDuration)
		isAvailable := true

		// If the selected schedule is for today, omit the hours that have already passed.
		if targetDate.YearDay() == now.YearDay() && targetDate.Year() == now.Year() {
			if currentSlot.Before(now) {
				isAvailable = false
			}
		}

		if isAvailable {
			// Iterate over the appointments from the database
			for _, appointment := range appointments {
				// A conflict occurs if the time slot begins before the appointment ends, and ends after the appointment begins.
				if currentSlot.Before(appointment.EndTime) && slotEnd.After(appointment.StartTime) {
					isAvailable = false
					break
				}
			}
		}

		if isAvailable {
			availableSlots = append(availableSlots, currentSlot.Format("15:04")) // HH:MM format
		}

		// Advance to the next available time slot
		// (You can change this to advance in 30-minute increments instead of skipping the entire duration of the service)
		currentSlot = currentSlot.Add(slotDuration)
	}

	// Avoid returning null in the JSON if there are no time slots.
	if availableSlots == nil {
		availableSlots = make([]string, 0)
	}

	return availableSlots, nil
}

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
	CheckOverlap(ctx context.Context, storeID string, startTime, endTime time.Time) (bool, error)
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
}

// NewAppointmentUseCase constructor
func NewAppointmentUseCase(repo AppointmentRepository, scheduleRepo StoreScheduleRepository) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo:         repo,
		scheduleRepo: scheduleRepo,
	}
}

// Create method
func (usecase *AppointmentUseCase) Create(ctx context.Context, storeID, serviceID, userID string, startTime time.Time, durationMinutes int) (*domain.Appointment, error) {
	if startTime.Before(time.Now()) {
		return nil, fmt.Errorf("%w: cannot schedule an appointment in the past", domain.ErrInvalidInput)
	}

	endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)

	isOverlapping, err := usecase.repo.CheckOverlap(ctx, storeID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("checking availability: %w", err)
	}

	if isOverlapping {
		return nil, fmt.Errorf("%w: time slot already booked", domain.ErrAlreadyExists)
	}

	appointment := domain.NewAppointment(storeID, serviceID, userID, "", startTime, endTime)

	if err := usecase.repo.CreateAppointment(ctx, appointment); err != nil {
		return nil, fmt.Errorf("creating appointment: %w", err)
	}

	return appointment, nil
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
	openParsed, _ := time.Parse("15:04:05", schedule.OpenTime)
	if openParsed.IsZero() {
		openParsed, _ = time.Parse("15:04", schedule.OpenTime)
	}
	closeParsed, _ := time.Parse("15:04:05", schedule.CloseTime)
	if closeParsed.IsZero() {
		closeParsed, _ = time.Parse("15:04", schedule.CloseTime)
	}

	openTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), openParsed.Hour(), openParsed.Minute(), 0, 0, targetDate.Location())
	closeTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), closeParsed.Hour(), closeParsed.Minute(), 0, 0, targetDate.Location())

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

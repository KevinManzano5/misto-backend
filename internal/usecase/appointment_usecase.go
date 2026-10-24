package usecase

import (
	"context"
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

// AppointmentUseCase struct
type AppointmentUseCase struct {
	repo AppointmentRepository
}

// NewAppointmentUseCase constructor
func NewAppointmentUseCase(repo AppointmentRepository) *AppointmentUseCase {
	return &AppointmentUseCase{
		repo: repo,
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
	// Parse the requested date (assuming YYYY-MM-DD format)
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	// 1. Retrieve appointments scheduled for that day (from the repository)
	appointments, err := usecase.repo.ListAppointmentsByStoreDate(ctx, storeID, targetDate, targetDate)
	// Error handling...

	// 2. Define working hours (e.g., 09:00 to 18:00)
	openTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 9, 0, 0, 0, targetDate.Location())
	closeTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 18, 0, 0, 0, targetDate.Location())

	var availableSlots []string
	slotDuration := time.Duration(durationMinutes) * time.Minute
	currentSlot := openTime

	// 3. Generate strips and filter overlaps
	for currentSlot.Add(slotDuration).Compare(closeTime) <= 0 {
		slotEnd := currentSlot.Add(slotDuration)
		isAvailable := true

		// Here you iterate over the 'appointments' array from the DB.
		for _, appointment := range appointments {
			if currentSlot.Before(appointment.EndTime) && slotEnd.After(appointment.StartTime) {
				isAvailable = false
				break
			}
		}

		if isAvailable {
			availableSlots = append(availableSlots, currentSlot.Format("15:04")) // HH:MM format
		}

		// Advance to the next available time slot (you can advance by the duration of the service or in fixed 30-minute intervals)
		currentSlot = currentSlot.Add(slotDuration)
	}

	return availableSlots, nil
}

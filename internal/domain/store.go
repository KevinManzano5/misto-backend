package domain

import "time"

// Store struct represents system user
type Store struct {
	ID                  string
	UserID              string
	Name                string
	Description         string
	IsActive            bool
	AppointmentDuration int32
	// Timezone is the store's IANA timezone (e.g. "America/Bogota"), used to interpret its
	// schedule's open/close hours and to resolve which calendar day an appointment falls on.
	Timezone string
	// AutoConfirmAppointments controls whether a new appointment is confirmed immediately or
	// left pending for the owner to approve/reject.
	AutoConfirmAppointments bool
	// Email is the store's contact email address.
	Email string
	// Address is the store's physical address.
	Address string
	// AllowSameDayAppointments controls whether a client can book an appointment on the same
	// calendar day it is requested, in the store's own timezone.
	AllowSameDayAppointments bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// NewStore function
func NewStore(userID, name, description string, isActive bool, appointmentDuration int32, timezone string, autoConfirmAppointments bool, email, address string, allowSameDayAppointments bool) *Store {
	now := time.Now()

	return &Store{
		UserID:                   userID,
		Name:                     name,
		Description:              description,
		IsActive:                 isActive,
		AppointmentDuration:      appointmentDuration,
		Timezone:                 timezone,
		AutoConfirmAppointments:  autoConfirmAppointments,
		Email:                    email,
		Address:                  address,
		AllowSameDayAppointments: allowSameDayAppointments,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

// IsValid method
func (store *Store) IsValid() bool {
	return store.UserID != "" && store.Name != "" && store.AppointmentDuration > 0 && store.Timezone != ""
}

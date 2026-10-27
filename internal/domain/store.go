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
	Timezone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewStore function
func NewStore(userID, name, description string, isActive bool, appointmentDuration int32, timezone string) *Store {
	now := time.Now()

	return &Store{
		UserID:              userID,
		Name:                name,
		Description:         description,
		IsActive:            isActive,
		AppointmentDuration: appointmentDuration,
		Timezone:            timezone,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

// IsValid method
func (store *Store) IsValid() bool {
	return store.UserID != "" && store.Name != "" && store.AppointmentDuration > 0 && store.Timezone != ""
}

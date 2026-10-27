package domain

import "time"

// Appointment status values.
const (
	AppointmentStatusPending   = "pending"
	AppointmentStatusConfirmed = "confirmed"
	AppointmentStatusCancelled = "cancelled"
	AppointmentStatusCompleted = "completed"
)

// Appointment struct
type Appointment struct {
	ID        string
	StoreID   string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AppointmentWithClient pairs an appointment with the basic profile of the client who booked it,
// for views (like a store owner's appointment list) that need both together.
type AppointmentWithClient struct {
	Appointment
	ClientUsername  string
	ClientEmail     string
	ClientFirstName string
	ClientLastName  string
}

// NewAppointment constructor
func NewAppointment(storeID, userID, status string, startTime, endtime time.Time) *Appointment {
	now := time.Now()

	return &Appointment{
		StoreID:   storeID,
		UserID:    userID,
		Status:    status,
		StartTime: startTime,
		EndTime:   endtime,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

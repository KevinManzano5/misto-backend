package domain

import "time"

// Appointment status values.
const (
	AppointmentStatusPending   = "pending"
	AppointmentStatusConfirmed = "confirmed"
	AppointmentStatusCancelled = "cancelled"
	AppointmentStatusCompleted = "completed"
	AppointmentStatusExpired   = "expired"
)

// Appointment struct
type Appointment struct {
	ID        string
	StoreID   string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	// CancellationReason is the reason given when the appointment was cancelled, empty otherwise.
	CancellationReason string
	CreatedAt          time.Time
	UpdatedAt          time.Time
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

// AppointmentWithStore pairs an appointment with the basic info of the store it was booked at,
// for a client's own "my appointments" view.
type AppointmentWithStore struct {
	Appointment
	StoreName string
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

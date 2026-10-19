package domain

import "time"

// Appointment struct
type Appointment struct {
	ID        string
	StoreID   string
	ServiceID string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAppointment constructor
func NewAppointment(storeID, serviceID, userID, status string, startTime, endtime time.Time) *Appointment {
	now := time.Now()

	return &Appointment{
		StoreID:   storeID,
		ServiceID: serviceID,
		UserID:    userID,
		Status:    status,
		StartTime: startTime,
		EndTime:   endtime,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

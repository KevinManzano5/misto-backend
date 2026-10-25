package domain

import "time"

// StoreSchedule struct represents the operating hours of a store for a specific day
type StoreSchedule struct {
	ID        string
	StoreID   string
	DayOfWeek int32  // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
	OpenTime  string // Expected format: "15:04" or "15:04:05" (HH:MM / HH:MM:SS)
	CloseTime string // Expected format: "15:04" or "15:04:05"
	IsClosed  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewStoreSchedule function
func NewStoreSchedule(storeID string, dayOfWeek int32, openTime, closeTime string, isClosed bool) *StoreSchedule {
	now := time.Now()

	// If the day is marked as closed, we can clear the hours to maintain consistency.
	if isClosed {
		openTime = ""
		closeTime = ""
	}

	return &StoreSchedule{
		StoreID:   storeID,
		DayOfWeek: dayOfWeek,
		OpenTime:  openTime,
		CloseTime: closeTime,
		IsClosed:  isClosed,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsValid method validates the basic business rules for a store schedule
func (schedule *StoreSchedule) IsValid() bool {
	// The store ID is mandatory, and the day must be between 0 and 6.
	if schedule.StoreID == "" || schedule.DayOfWeek < 0 || schedule.DayOfWeek > 6 {
		return false
	}

	// If the store is NOT closed that day, it must have specified opening and closing times.
	if !schedule.IsClosed && (schedule.OpenTime == "" || schedule.CloseTime == "") {
		return false
	}

	return true
}

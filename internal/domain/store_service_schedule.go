package domain

import (
	"sort"
	"time"
)

// StoreServiceScheduleSegment represents one continuous block of time, within a single day,
// during which a store service accepts appointments (e.g. "08:00"-"12:00").
type StoreServiceScheduleSegment struct {
	ID        string
	OpenTime  string // Expected format: "15:04" or "15:04:05" (HH:MM / HH:MM:SS)
	CloseTime string // Expected format: "15:04" or "15:04:05"
}

// StoreServiceSchedule struct represents the operating hours of a store service for a specific
// day of the week, as zero or more non-overlapping time segments
type StoreServiceSchedule struct {
	ID             string
	StoreServiceID string
	DayOfWeek      int32 // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
	Segments       []StoreServiceScheduleSegment
	IsClosed       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewStoreServiceSchedule function
func NewStoreServiceSchedule(storeServiceID string, dayOfWeek int32, segments []StoreServiceScheduleSegment, isClosed bool) *StoreServiceSchedule {
	now := time.Now()

	// If the day is marked as closed, we can clear the segments to maintain consistency.
	if isClosed {
		segments = nil
	}

	return &StoreServiceSchedule{
		StoreServiceID: storeServiceID,
		DayOfWeek:      dayOfWeek,
		Segments:       segments,
		IsClosed:       isClosed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsValid method validates the basic business rules for a store service schedule
func (schedule *StoreServiceSchedule) IsValid() bool {
	// The store service ID is mandatory, and the day must be between 0 and 6.
	if schedule.StoreServiceID == "" || schedule.DayOfWeek < 0 || schedule.DayOfWeek > 6 {
		return false
	}

	// If the service is NOT closed that day, it must have at least one segment.
	if !schedule.IsClosed && len(schedule.Segments) == 0 {
		return false
	}

	return segmentsAreValid(schedule.Segments)
}

// segmentsAreValid reports whether every segment is internally consistent (parseable
// "HH:MM"/"HH:MM:SS", open strictly before close) and no two segments overlap.
func segmentsAreValid(segments []StoreServiceScheduleSegment) bool {
	type parsedSegment struct {
		open, close time.Time
	}

	parsed := make([]parsedSegment, 0, len(segments))

	for _, segment := range segments {
		open, err := parseClockTime(segment.OpenTime)
		if err != nil {
			return false
		}

		closeTime, err := parseClockTime(segment.CloseTime)
		if err != nil {
			return false
		}

		if !open.Before(closeTime) {
			return false
		}

		parsed = append(parsed, parsedSegment{open: open, close: closeTime})
	}

	sort.Slice(parsed, func(i, j int) bool { return parsed[i].open.Before(parsed[j].open) })

	for i := 1; i < len(parsed); i++ {
		if parsed[i].open.Before(parsed[i-1].close) {
			return false
		}
	}

	return true
}

// parseClockTime parses a clock time string in "15:04:05" or "15:04" format.
func parseClockTime(value string) (time.Time, error) {
	if parsed, err := time.Parse("15:04:05", value); err == nil {
		return parsed, nil
	}

	return time.Parse("15:04", value)
}

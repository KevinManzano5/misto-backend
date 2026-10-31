package domain

import "time"

// StoreRating struct represents a client's review of a completed appointment at a store.
type StoreRating struct {
	ID            string
	AppointmentID string
	StoreID       string
	UserID        string
	Score         int32
	// Comment is optional free text left by the client, empty when not provided.
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StoreRatingWithClient pairs a rating with the basic profile of the client who left it, for a
// store's public rating list.
type StoreRatingWithClient struct {
	StoreRating
	ClientUsername  string
	ClientFirstName string
	ClientLastName  string
}

// StoreRatingSummary is the aggregate average score and count of ratings for a store.
type StoreRatingSummary struct {
	StoreID      string
	AverageScore float64
	RatingCount  int64
}

// NewStoreRating constructor
func NewStoreRating(appointmentID, storeID, userID string, score int32, comment string) *StoreRating {
	now := time.Now()

	return &StoreRating{
		AppointmentID: appointmentID,
		StoreID:       storeID,
		UserID:        userID,
		Score:         score,
		Comment:       comment,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsValid method
func (rating *StoreRating) IsValid() bool {
	return rating.AppointmentID != "" && rating.StoreID != "" && rating.UserID != "" &&
		rating.Score >= 1 && rating.Score <= 5
}

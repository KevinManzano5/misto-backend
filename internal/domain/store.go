package domain

import "time"

// Store struct represents system user
type Store struct {
	ID          string
	UserID      string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewStore function
func NewStore(userID, name, description string, isActive bool) *Store {
	now := time.Now()

	return &Store{
		UserID:      userID,
		Name:        name,
		Description: description,
		IsActive:    isActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsValid method
func (store *Store) IsValid() bool {
	return store.UserID != "" && store.Name != ""
}

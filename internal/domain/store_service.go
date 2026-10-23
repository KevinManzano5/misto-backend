package domain

import "time"

// StoreService struct represents a service offered by a store
type StoreService struct {
	ID              string
	StoreID         string
	Name            string
	Description     string
	DurationMinutes int32
	Price           float64
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewStoreService function
func NewStoreService(storeID, name, description string, durationMinutes int32, price float64, isActive bool) *StoreService {
	now := time.Now()

	return &StoreService{
		StoreID:         storeID,
		Name:            name,
		Description:     description,
		DurationMinutes: durationMinutes,
		Price:           price,
		IsActive:        isActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// IsValid method validates the basic business rules for a store service
func (service *StoreService) IsValid() bool {
	return service.StoreID != "" && service.Name != "" && service.DurationMinutes > 0 && service.Price >= 0
}

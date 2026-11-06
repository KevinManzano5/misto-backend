package domain

import "time"

// StoreService represents a service a store offers (e.g. "Haircut", "Manicure"), scoped to one
// of the categories assigned to that store.
type StoreService struct {
	ID                 string
	StoreID            string
	CategoryID         string
	Name               string
	Description        string
	DurationMinutes    int32
	Price              float64
	IsActive           bool
	MinLeadTimeMinutes int32
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewStoreService constructor
func NewStoreService(storeID, categoryID, name, description string, durationMinutes int32, price float64, minLeadTimeMinutes int32) *StoreService {
	now := time.Now()

	return &StoreService{
		StoreID:            storeID,
		CategoryID:         categoryID,
		Name:               name,
		Description:        description,
		DurationMinutes:    durationMinutes,
		Price:              price,
		IsActive:           true,
		MinLeadTimeMinutes: minLeadTimeMinutes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// IsValid reports whether the service has the minimum data required to be persisted.
func (service *StoreService) IsValid() bool {
	return service.StoreID != "" && service.CategoryID != "" && service.Name != "" &&
		service.DurationMinutes > 0 && service.Price >= 0 && service.MinLeadTimeMinutes >= 0
}

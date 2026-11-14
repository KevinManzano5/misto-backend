package domain

import "time"

// StoreCategory represents an entry in the catalog of store categories (e.g. "Barbershop",
// "Spa"), used to classify stores and, in the future, to power service suggestions.
type StoreCategory struct {
	ID          string
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewStoreCategory constructor
func NewStoreCategory(name, slug, description string) *StoreCategory {
	now := time.Now()

	return &StoreCategory{
		Name:        name,
		Slug:        slug,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsValid reports whether the category has the minimum data required to be persisted.
func (category *StoreCategory) IsValid() bool {
	return category.Name != "" && category.Slug != ""
}

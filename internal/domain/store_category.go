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

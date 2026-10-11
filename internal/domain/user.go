package domain

import "time"

// User struct represents system user
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser function
func NewUser(username, email, passwordHash, firstName, lastName string, isActive bool) *User {
	now := time.Now()

	return &User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		IsActive:     isActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsValid method
func (user *User) IsValid() bool {
	return user.Username != "" && user.Email != "" && user.FirstName != "" && user.LastName != ""
}

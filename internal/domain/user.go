package domain

import "time"

const (
	// RoleMember is the default role assigned to every self-registered user.
	RoleMember = "member"
	// RoleAdmin grants access to system-wide administration capabilities.
	RoleAdmin = "admin"
)

// User struct represents system user
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	Role         string
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
		Role:         RoleMember,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsValid method
func (user *User) IsValid() bool {
	return user.Username != "" && user.Email != "" && user.FirstName != "" && user.LastName != ""
}

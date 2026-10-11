// Package usecase contains user use cases
package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// UserRepository interface
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// UserUseCase struct handles business operations over user
type UserUseCase struct {
	repo UserRepository
}

// NewUserUseCase function
func NewUserUseCase(repo UserRepository) *UserUseCase {
	return &UserUseCase{
		repo: repo,
	}
}

// GetUser function
func (usecase *UserUseCase) GetUser(ctx context.Context, id string) (*domain.User, error) {
	user, err := usecase.repo.GetByID(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("getting user %s: %w", id, err)
	}

	return user, nil
}

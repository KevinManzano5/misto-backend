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
	List(ctx context.Context, limit, offset int32) ([]*domain.User, error)
	Count(ctx context.Context) (int64, error)
}

const (
	defaultUsersPage     = 1
	defaultUsersPageSize = 20
	maxUsersPageSize     = 100
)

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

// ListUsers returns a page of all users ordered by most recently created, along with the total
// count across all pages. Admin-only: authorization is enforced by the handler/route, not here.
// page/pageSize are normalized with sane defaults and an upper bound.
func (usecase *UserUseCase) ListUsers(ctx context.Context, page, pageSize int32) ([]*domain.User, int64, error) {
	if page < 1 {
		page = defaultUsersPage
	}

	if pageSize < 1 {
		pageSize = defaultUsersPageSize
	}

	if pageSize > maxUsersPageSize {
		pageSize = maxUsersPageSize
	}

	offset := (page - 1) * pageSize

	users, err := usecase.repo.List(ctx, pageSize, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("listing users: %w", err)
	}

	total, err := usecase.repo.Count(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("counting users: %w", err)
	}

	return users, total, nil
}

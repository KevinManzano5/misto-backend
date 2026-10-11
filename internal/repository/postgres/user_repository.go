// Package postgres configuration
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository struct
type UserRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewUserRepository constructor
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// Create method
func (repository *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result, err := repository.queries.CreateUser(ctx, postgres.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    &user.FirstName,
		LastName:     &user.LastName,
	})

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w", domain.ErrAlreadyExists)
		}

		return fmt.Errorf("creating user in db: %w", err)
	}

	user.ID = result.ID.String()
	user.CreatedAt = result.CreatedAt.Time
	user.UpdatedAt = result.UpdatedAt.Time

	return nil
}

// GetByID method
func (repository *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var pgID pgtype.UUID

	if err := pgID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid id %s: %w", id, domain.ErrNotFound)
	}

	result, err := repository.queries.GetUserByID(ctx, pgID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("id %s: %w", id, domain.ErrNotFound)
		}

		return nil, fmt.Errorf("searching user %s: %w", id, err)
	}

	return toDomainUser(result), nil
}

// GetByEmail method
func (repository *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	result, err := repository.queries.GetUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("email %s: %w", email, domain.ErrNotFound)
		}

		return nil, fmt.Errorf("searching user %s: %w", email, err)
	}

	return toDomainUser(result), nil
}

func toDomainUser(user postgres.User) *domain.User {
	return &domain.User{
		ID:           user.ID.String(),
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    *user.FirstName,
		LastName:     *user.LastName,
		IsActive:     user.IsActive,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}
}

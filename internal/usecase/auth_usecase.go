// Package usecase contains use cases
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/auth"
	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// authUserRepository interface
type authUserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// AuthTokens struct
type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// AuthUseCase struct
type AuthUseCase struct {
	userRepo             authUserRepository
	jwtManager           *auth.JWTManager
	tokenStore           *auth.TokenStore
	refreshTokenDuration time.Duration
}

// NewAuthUseCase function
func NewAuthUseCase(
	userRepo authUserRepository,
	jwtManager *auth.JWTManager,
	tokenStore *auth.TokenStore,
	refreshTokenDuration time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:             userRepo,
		jwtManager:           jwtManager,
		tokenStore:           tokenStore,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// generateTokenPair method
func (usecase *AuthUseCase) generateTokenPair(ctx context.Context, userID, email, role string) (*AuthTokens, error) {
	accessToken, err := usecase.jwtManager.GenerateAccessToken(userID, email, role)

	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, err := usecase.jwtManager.GenerateRefreshToken(userID, email, role)

	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	refreshClaims, err := usecase.jwtManager.VerifyToken(refreshToken)

	if err != nil {
		return nil, fmt.Errorf("extracting claims from refresh token: %w", err)
	}

	if err := usecase.tokenStore.StoreRefreshToken(ctx, refreshClaims.ID, userID, usecase.refreshTokenDuration); err != nil {
		return nil, fmt.Errorf("saving refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    15 * time.Minute,
	}, nil
}

// SignUp method
func (usecase *AuthUseCase) SignUp(ctx context.Context, username, email, firstName, lastName, passwordHash string) (*AuthTokens, error) {
	passwordHash, err := auth.HashPassword(passwordHash)

	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := domain.NewUser(username, email, passwordHash, firstName, lastName, true)

	if err := usecase.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	tokens, err := usecase.generateTokenPair(ctx, user.ID, user.Email, "member")

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// SignIn method
func (usecase *AuthUseCase) SignIn(ctx context.Context, email, password string) (*AuthTokens, error) {
	user, err := usecase.userRepo.GetByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("invalid credentials: %w", domain.ErrUnauthorized)
		}

		return nil, fmt.Errorf("searching user: %w", err)
	}

	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		if errors.Is(err, auth.ErrPasswordMissMatch) {
			return nil, fmt.Errorf("invalid credentials: %w", domain.ErrUnauthorized)
		}

		return nil, fmt.Errorf("validating password: %w", err)
	}

	tokens, err := usecase.generateTokenPair(ctx, user.ID, user.Email, "member")

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// RefreshTokens method
func (usecase *AuthUseCase) RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	claims, err := usecase.jwtManager.VerifyToken(refreshToken)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", domain.ErrUnauthorized)
	}

	_, err = usecase.tokenStore.ValidateRefreshToken(ctx, claims.ID)

	if err != nil {
		if errors.Is(err, auth.ErrRevokedToken) || errors.Is(err, auth.ErrTokenNotFound) {
			return nil, fmt.Errorf("invalid refresh token: %w", domain.ErrUnauthorized)
		}

		return nil, fmt.Errorf("validating refresh token: %w", err)
	}

	if err := usecase.tokenStore.RevokedRefreshToken(ctx, claims.Issuer, usecase.refreshTokenDuration); err != nil {
		return nil, fmt.Errorf("revoking token: %w", err)
	}

	tokens, err := usecase.generateTokenPair(ctx, claims.UserID, claims.Email, claims.Role)

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// SignOut method
func (usecase *AuthUseCase) SignOut(ctx context.Context, refreshToken string) error {
	claims, err := usecase.jwtManager.VerifyToken(refreshToken)

	if err != nil {
		return nil
	}

	if err := usecase.tokenStore.RevokedRefreshToken(ctx, claims.ID, usecase.refreshTokenDuration); err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}

	return nil
}

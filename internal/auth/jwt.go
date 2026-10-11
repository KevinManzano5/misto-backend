// Package auth contains auth
package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrExpiredToken error
var ErrExpiredToken = errors.New("expired token")

// ErrInvalidToken error
var ErrInvalidToken = errors.New("invalid token")

// Claims struct
type Claims struct {
	UserID               string `json:"id"`
	Email                string `json:"email"`
	Role                 string `json:"role"`
	jwt.RegisteredClaims        // JWT standard fields
}

// JWTConfig struct
type JWTConfig struct {
	SecretKey            []byte
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// JWTManager struct
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager function
func NewJWTManager(config JWTConfig) (*JWTManager, error) {
	if len(config.SecretKey) < 32 {
		return nil, fmt.Errorf("secret key must have at least 32 bytes, now has: %d", len(config.SecretKey))
	}

	return &JWTManager{
		config: config,
	}, nil
}

// generateTokenID function
func generateTokenID() string {
	b := make([]byte, 16)

	_, err := rand.Read(b)

	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GenerateToken method
func (manager *JWTManager) GenerateToken(userID, email, role string, duration time.Duration) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
			Issuer:    "devboard",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(manager.config.SecretKey)

	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// GenerateAccessToken method
func (manager *JWTManager) GenerateAccessToken(userID, email, role string) (string, error) {
	return manager.GenerateToken(userID, email, role, manager.config.AccessTokenDuration)
}

// GenerateRefreshToken method
func (manager *JWTManager) GenerateRefreshToken(userID, email, role string) (string, error) {
	return manager.GenerateToken(userID, email, role, manager.config.RefreshTokenDuration)
}

// VerifyToken method
func (manager *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected algorithm: %v", t.Header["alg"])
			}

			return manager.config.SecretKey, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer("devboard"),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}

		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)

	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

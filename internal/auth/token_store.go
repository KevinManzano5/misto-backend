// Package auth contains auth
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRevokedToken error
var ErrRevokedToken = errors.New("revoked token")

// ErrTokenNotFound error
var ErrTokenNotFound = errors.New("token not found")

// TokenStore struct
type TokenStore struct {
	redis *redis.Client
}

// NewTokenStore function
func NewTokenStore(redisClient *redis.Client) *TokenStore {
	return &TokenStore{
		redis: redisClient,
	}
}

// refreshKey function
func refreshKey(tokenID string) string {
	return "refresh:" + tokenID
}

// revokedKey function
func revokedKey(tokenID string) string {
	return "revoked:" + tokenID
}

// StoreRefreshToken method
func (store *TokenStore) StoreRefreshToken(ctx context.Context, tokenID, userID string, ttl time.Duration) error {
	err := store.redis.Set(
		ctx,
		refreshKey(tokenID),
		userID,
		ttl,
	).Err()

	if err != nil {
		return fmt.Errorf("saving refresh token in Redis: %w", err)
	}

	return nil
}

// ValidateRefreshToken method
func (store *TokenStore) ValidateRefreshToken(ctx context.Context, tokenID string) (string, error) {
	revoked, err := store.redis.Exists(ctx, revokedKey(tokenID)).Result()

	if err != nil {
		return "", fmt.Errorf("validating revocation: %w", err)
	}

	if revoked > 0 {
		return "", ErrRevokedToken
	}

	userID, err := store.redis.Get(ctx, refreshKey(tokenID)).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrTokenNotFound
		}

		return "", fmt.Errorf("getting refresh token: %w", err)
	}

	return userID, nil
}

// RevokedRefreshToken method
func (store *TokenStore) RevokedRefreshToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	pipe := store.redis.Pipeline()

	pipe.Set(ctx, revokedKey(tokenID), "1", ttl)
	pipe.Del(ctx, refreshKey(tokenID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens method
func (store *TokenStore) RevokeAllUserTokens(ctx context.Context, userID string, ttl time.Duration) error {
	var cursor uint64

	pattern := "refresh:*"

	for {
		keys, nextCursor, err := store.redis.Scan(ctx, cursor, pattern, 100).Result()

		if err != nil {
			return fmt.Errorf("searching user tokens: %w", err)
		}

		for _, key := range keys {
			storedUserID, err := store.redis.Get(ctx, key).Result()

			if err != nil {
				continue
			}

			if storedUserID == userID {
				tokenID := key[len("refresh:"):]

				_ = store.RevokedRefreshToken(ctx, tokenID, ttl)
			}
		}

		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return nil
}

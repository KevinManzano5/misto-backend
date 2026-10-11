// Package middlewares contains middlewares
package middlewares

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/KevinManzano5/misto-backend/internal/auth"
	"github.com/KevinManzano5/misto-backend/internal/domain"
)

type contextKey string

const (
	claimsKey contextKey = "auth_claims"
)

// extractBearerToken function
func extractBearerToken(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")

	if authHeader == "" {
		return "", domain.ErrUnauthorized
	}

	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", domain.ErrUnauthorized
	}

	token := strings.TrimSpace(parts[1])

	if token == "" {
		return "", domain.ErrUnauthorized
	}

	return token, nil
}

// respondAuthError function
func respondAuthError(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	status int,
	title string,
	detail string,
) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	response := struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
	}{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(
			r.Context(),
			"error trying to write auth response",
			"error", err,
			"path", r.URL.Path,
		)
	}
}

// RequireAuth function
func RequireAuth(jwtManager *auth.JWTManager, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractBearerToken(r)

			if err != nil {
				respondAuthError(w, r, logger, http.StatusUnauthorized, "Unauthorized", "access token is required")

				return
			}

			claims, err := jwtManager.VerifyToken(tokenString)

			if err != nil {
				respondAuthError(w, r, logger, http.StatusUnauthorized, "Unauthorized", "access token is required")

				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims function
func GetClaims(req *http.Request) *auth.Claims {
	claims, _ := req.Context().Value(claimsKey).(*auth.Claims)

	return claims
}

// GetUserID function
func GetUserID(req *http.Request) string {
	claims := GetClaims(req)

	if claims == nil {
		return ""
	}

	return claims.UserID
}

// RequireRole function
func RequireRole(role string, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)

			if claims == nil {
				logger.ErrorContext(r.Context(), "RequireRole used without RequireAuth", slog.String("path", r.URL.Path))

				respondAuthError(w, r, logger, http.StatusUnauthorized, "Unauthorized", "access token is required")

				return
			}

			if claims.Role != role {
				respondAuthError(w, r, logger, http.StatusUnauthorized, "Forbidden", "Forbidden")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

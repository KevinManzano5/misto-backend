// Package middlewares contains middlewares
package middlewares

import (
	"net/http"
	"strings"
)

// SecurityConfig struct
type SecurityConfig struct {
	AllowedOrigins []string
}

// SecurityHeaders function
func SecurityHeaders(config SecurityConfig) Middleware {
	allowedOrigins := make(map[string]bool, len(config.AllowedOrigins))

	for _, origin := range config.AllowedOrigins {
		allowedOrigins[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)

				return
			}

			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")

			if !strings.HasPrefix(r.Host, "localhost") {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000;includeSubdomains")
			}

			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permission-Policy", "camera=(), microphone=(), geolocation=()")

			next.ServeHTTP(w, r)
		})
	}
}

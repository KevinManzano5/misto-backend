// Package middlewares contains middlewares
package middlewares

import (
	"net/http"
)

// Middleware func
type Middleware func(http.Handler) http.Handler

// Chain function
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

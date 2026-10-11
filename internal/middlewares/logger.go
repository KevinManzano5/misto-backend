// Package middlewares contains middlewares used on the app
package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder struct
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseRecorder(writer http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: writer,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader method
func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.written {
		rr.statusCode = code
		rr.written = true
		rr.ResponseWriter.WriteHeader(code)
	}
}

// Write method
func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.written {
		rr.WriteHeader(http.StatusOK)
	}

	return rr.ResponseWriter.Write(b)
}

// Logger function
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			start := time.Now()

			recorder := newResponseRecorder(writer)

			// * Call next handler
			next.ServeHTTP(recorder, request)

			// * Log
			logger.InfoContext(request.Context(), "request completed",
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_address", request.RemoteAddr),
			)
		})
	}
}

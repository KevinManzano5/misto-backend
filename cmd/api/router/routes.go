package router

import (
	"net/http"

	"github.com/KevinManzano5/misto-backend/internal/handler"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/server"
	httpSwagger "github.com/swaggo/http-swagger"
)

// AppHandlers groups all instantiated handlers to facilitate dependency injection.
type AppHandlers struct {
	Health      http.Handler
	User        *handler.UserHandler
	Auth        *handler.AuthHandler
	Store       *handler.StoreHandler
	Appointment *handler.AppointmentHandler
}

// setupRoutes orchestrates the configuration of all system routes, organized by module.
func SetupRoutes(srv *server.Server, h *AppHandlers, authMW middlewares.Middleware) {
	setupSystemRoutes(srv, h.Health)
	setupAuthRoutes(srv, h.Auth, h.User, authMW)
	setupStoreRoutes(srv, h.Store, authMW)
	setupAppointmentRoutes(srv, h.Appointment, authMW)
}

// setupSystemRoutes registers health and documentation endpoints (public)
func setupSystemRoutes(srv *server.Server, health http.Handler) {
	srv.RegisterRoutes("GET /docs/", httpSwagger.WrapHandler)
	srv.RegisterRoutes("GET /health", health)
}

// setupAuthRoutes registers endpoints related to authentication and basic user management (mixed)
func setupAuthRoutes(srv *server.Server, auth *handler.AuthHandler, user *handler.UserHandler, authMW middlewares.Middleware) {
	srv.RegisterRoutes("POST /api/v1/auth/sign_up", http.HandlerFunc(auth.SignUp))
	srv.RegisterRoutes("POST /api/v1/auth/sign_in", http.HandlerFunc(auth.SignIn))
	srv.RegisterRoutes("POST /api/v1/auth/sign_out", http.HandlerFunc(auth.SignOut))
	srv.RegisterRoutes("POST /api/v1/auth/refresh", http.HandlerFunc(auth.Refresh))

	// Private: reaching the handler already proves the token is valid, since authMW rejects
	// a missing/invalid/expired token with 401 before it gets here.
	srv.RegisterRoutes("GET /api/v1/auth/validate", middlewares.Chain(http.HandlerFunc(auth.ValidateToken), authMW))

	// Consider protecting this route with authMW in the future if profiles are private.
	srv.RegisterRoutes("GET /api/v1/users/{id}", http.HandlerFunc(user.Get))
}

// setupStoreRoutes registers endpoints for managing stores and their schedules (mixed).
// A store's weekly schedules are embedded directly in its GET response (storeResponse.Schedules),
// so there is no separate read endpoint for them; only the mutation route remains.
func setupStoreRoutes(srv *server.Server, store *handler.StoreHandler, authMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	// === Store routes ===
	// Public
	srv.RegisterRoutes("GET /api/v1/stores", http.HandlerFunc(store.ListStores))
	srv.RegisterRoutes("GET /api/v1/stores/{id}", http.HandlerFunc(store.GetStoreByID))

	// Private
	srv.RegisterRoutes("POST /api/v1/stores", withAuth(store.Create))
	srv.RegisterRoutes("PATCH /api/v1/stores/{id}", withAuth(store.UpdateStore))
	srv.RegisterRoutes("DELETE /api/v1/stores/{id}", withAuth(store.DeleteStore))

	// === Store schedules routes ===
	// Private
	srv.RegisterRoutes("PUT /api/v1/stores/{storeId}/schedules/{day}", withAuth(store.UpsertSchedule))
}

// setupAppointmentRoutes registers endpoints for appointment management (mixed)
func setupAppointmentRoutes(srv *server.Server, appointment *handler.AppointmentHandler, authMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	// Public
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/availability", http.HandlerFunc(appointment.GetAvailableSlots))

	// Private
	srv.RegisterRoutes("POST /api/v1/appointments", withAuth(appointment.Create))
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/appointments", withAuth(appointment.GetBookedAppointments))
}

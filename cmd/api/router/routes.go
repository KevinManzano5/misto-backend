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
	Health       http.Handler
	User         *handler.UserHandler
	Auth         *handler.AuthHandler
	Store        *handler.StoreHandler
	StoreService *handler.StoreServiceHandler
	Appointment  *handler.AppointmentHandler
}

// setupRoutes orchestrates the configuration of all system routes, organized by module.
func SetupRoutes(srv *server.Server, h *AppHandlers, authMW middlewares.Middleware) {
	setupSystemRoutes(srv, h.Health)
	setupAuthRoutes(srv, h.Auth, h.User)
	setupStoreRoutes(srv, h.Store, h.StoreService, authMW)
	setupAppointmentRoutes(srv, h.Appointment, authMW)
}

// setupSystemRoutes registers health and documentation endpoints (public)
func setupSystemRoutes(srv *server.Server, health http.Handler) {
	srv.RegisterRoutes("GET /docs/", httpSwagger.WrapHandler)
	srv.RegisterRoutes("GET /health", health)
}

// setupAuthRoutes registers endpoints related to authentication and basic user management (public)
func setupAuthRoutes(srv *server.Server, auth *handler.AuthHandler, user *handler.UserHandler) {
	srv.RegisterRoutes("POST /api/v1/auth/sign_up", http.HandlerFunc(auth.SignUp))
	srv.RegisterRoutes("POST /api/v1/auth/sign_in", http.HandlerFunc(auth.SignIn))
	srv.RegisterRoutes("POST /api/v1/auth/sign_out", http.HandlerFunc(auth.SignOut))
	srv.RegisterRoutes("POST /api/v1/auth/refresh", http.HandlerFunc(auth.Refresh))

	// Consider protecting this route with authMW in the future if profiles are private.
	srv.RegisterRoutes("GET /api/v1/users/{id}", http.HandlerFunc(user.Get))
}

// setupStoreRoutes registers endpoints for managing stores and their services (mixed)
func setupStoreRoutes(srv *server.Server, store *handler.StoreHandler, service *handler.StoreServiceHandler, authMW middlewares.Middleware) {
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

	// === Store services routes ===
	// Public
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/services", http.HandlerFunc(service.ListByStoreID))
	srv.RegisterRoutes("GET /api/v1/services/{id}", http.HandlerFunc(service.GetByID))
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/availability", http.HandlerFunc(service.GetAvailability))

	// Private
	srv.RegisterRoutes("POST /api/v1/stores/{storeId}/services", withAuth(service.Create))
	srv.RegisterRoutes("PATCH /api/v1/services/{id}", withAuth(service.Update))
	srv.RegisterRoutes("DELETE /api/v1/services/{id}", withAuth(service.Delete))
}

// setupAppointmentRoutes registers endpoints for appointment management (Private)
func setupAppointmentRoutes(srv *server.Server, appointment *handler.AppointmentHandler, authMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	srv.RegisterRoutes("POST /api/v1/appointments", withAuth(appointment.Create))
}

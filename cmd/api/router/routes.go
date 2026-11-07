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
	Health                http.Handler
	User                  *handler.UserHandler
	Auth                  *handler.AuthHandler
	Store                 *handler.StoreHandler
	Appointment           *handler.AppointmentHandler
	StoreRating           *handler.StoreRatingHandler
	StoreCategory         *handler.StoreCategoryHandler
	StoreService          *handler.StoreServiceHandler
	PaymentMethodSettings *handler.PaymentMethodSettingsHandler
}

// setupRoutes orchestrates the configuration of all system routes, organized by module.
func SetupRoutes(srv *server.Server, h *AppHandlers, authMW, adminMW, optionalAuthMW middlewares.Middleware) {
	setupSystemRoutes(srv, h.Health, h.PaymentMethodSettings, authMW, adminMW)
	setupAuthRoutes(srv, h.Auth, h.User, authMW, adminMW)
	setupStoreRoutes(srv, h.Store, authMW, adminMW, optionalAuthMW)
	setupAppointmentRoutes(srv, h.Appointment, authMW)
	setupStoreRatingRoutes(srv, h.StoreRating, authMW)
	setupStoreCategoryRoutes(srv, h.StoreCategory)
	setupStoreServiceRoutes(srv, h.StoreService, authMW)
}

// setupSystemRoutes registers health/documentation endpoints (public) and platform-wide payment
// method settings (admin-only).
func setupSystemRoutes(srv *server.Server, health http.Handler, paymentMethods *handler.PaymentMethodSettingsHandler, authMW, adminMW middlewares.Middleware) {
	srv.RegisterRoutes("GET /docs/", httpSwagger.WrapHandler)
	srv.RegisterRoutes("GET /health", health)

	// Admin-only. Toggling a method here overrides every store's own per-store setting.
	srv.RegisterRoutes("GET /api/v1/payment-methods", middlewares.Chain(http.HandlerFunc(paymentMethods.Get), authMW, adminMW))
	srv.RegisterRoutes("PATCH /api/v1/payment-methods", middlewares.Chain(http.HandlerFunc(paymentMethods.Update), authMW, adminMW))
}

// setupAuthRoutes registers endpoints related to authentication and basic user management (mixed)
func setupAuthRoutes(srv *server.Server, auth *handler.AuthHandler, user *handler.UserHandler, authMW, adminMW middlewares.Middleware) {
	srv.RegisterRoutes("POST /api/v1/auth/sign_up", http.HandlerFunc(auth.SignUp))
	srv.RegisterRoutes("POST /api/v1/auth/sign_in", http.HandlerFunc(auth.SignIn))
	srv.RegisterRoutes("POST /api/v1/auth/sign_out", http.HandlerFunc(auth.SignOut))
	srv.RegisterRoutes("POST /api/v1/auth/refresh", http.HandlerFunc(auth.Refresh))

	// Private: reaching the handler already proves the token is valid, since authMW rejects
	// a missing/invalid/expired token with 401 before it gets here.
	srv.RegisterRoutes("GET /api/v1/auth/validate", middlewares.Chain(http.HandlerFunc(auth.ValidateToken), authMW))

	// Consider protecting this route with authMW in the future if profiles are private.
	srv.RegisterRoutes("GET /api/v1/users/{id}", http.HandlerFunc(user.Get))

	// Admin-only
	srv.RegisterRoutes("GET /api/v1/users", middlewares.Chain(http.HandlerFunc(user.ListUsers), authMW, adminMW))
}

// setupStoreRoutes registers endpoints for managing stores (mixed).
func setupStoreRoutes(srv *server.Server, store *handler.StoreHandler, authMW, adminMW, optionalAuthMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	// === Store routes ===
	// Public, but reads claims if a token is present so an admin caller also sees the app usage
	// fee in the response (see store.buildStoreResponse).
	srv.RegisterRoutes("GET /api/v1/stores", middlewares.Chain(http.HandlerFunc(store.ListStores), optionalAuthMW))
	srv.RegisterRoutes("GET /api/v1/stores/{id}", middlewares.Chain(http.HandlerFunc(store.GetStoreByID), optionalAuthMW))

	// Private
	srv.RegisterRoutes("POST /api/v1/stores", withAuth(store.Create))
	srv.RegisterRoutes("PATCH /api/v1/stores/{id}", withAuth(store.UpdateStore))
	srv.RegisterRoutes("DELETE /api/v1/stores/{id}", withAuth(store.DeleteStore))

	// === Store categories routes ===
	// Private. A store's assigned categories are embedded directly in its GET response
	// (storeResponse.Categories), so there is no separate read endpoint for them.
	srv.RegisterRoutes("PUT /api/v1/stores/{id}/categories", withAuth(store.ReplaceCategories))

	// === Store app usage fee routes ===
	// Admin-only. The fee the platform charges a store for app usage is never exposed through
	// the store owner's own GET/PATCH flow above.
	srv.RegisterRoutes("GET /api/v1/stores/{id}/fee", middlewares.Chain(http.HandlerFunc(store.GetStoreFee), authMW, adminMW))
	srv.RegisterRoutes("PATCH /api/v1/stores/{id}/fee", middlewares.Chain(http.HandlerFunc(store.UpdateStoreFee), authMW, adminMW))
}

// setupStoreCategoryRoutes registers endpoints for the store category catalog (public).
func setupStoreCategoryRoutes(srv *server.Server, category *handler.StoreCategoryHandler) {
	srv.RegisterRoutes("GET /api/v1/store-categories", http.HandlerFunc(category.ListCategories))
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
	srv.RegisterRoutes("GET /api/v1/appointments", withAuth(appointment.GetMyAppointments))
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/appointments", withAuth(appointment.GetBookedAppointments))
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/appointments/me", withAuth(appointment.GetActiveAppointmentStatus))
	srv.RegisterRoutes("PATCH /api/v1/appointments/{id}/status", withAuth(appointment.UpdateStatus))
	srv.RegisterRoutes("PATCH /api/v1/appointments/{id}/cancel", withAuth(appointment.Cancel))
	srv.RegisterRoutes("PATCH /api/v1/appointments/{id}/confirm-cash-payment", withAuth(appointment.ConfirmCashPayment))
	srv.RegisterRoutes("GET /api/v1/appointments/{id}/transactions", withAuth(appointment.ListPaymentTransactions))
}

// setupStoreRatingRoutes registers endpoints for rating a store's completed appointments (mixed).
// Create is nested under the appointment it rates (identity and authorization both derive from
// it); listing is nested under the store (how a public store page consumes it) — its rating
// average/count summary is served as part of the store's own GET/list responses instead of a
// separate endpoint; update/delete are top-level once the rating's own ID is known.
func setupStoreRatingRoutes(srv *server.Server, rating *handler.StoreRatingHandler, authMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	// Public
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/ratings", http.HandlerFunc(rating.ListByStore))

	// Private
	srv.RegisterRoutes("POST /api/v1/appointments/{appointmentId}/ratings", withAuth(rating.Create))
	srv.RegisterRoutes("PATCH /api/v1/ratings/{id}", withAuth(rating.Update))
	srv.RegisterRoutes("DELETE /api/v1/ratings/{id}", withAuth(rating.Delete))
}

// setupStoreServiceRoutes registers endpoints for a store's services (mixed). Listing and
// creation are nested under the store; update/delete are top-level once the service's own ID is
// known, mirroring setupStoreRatingRoutes. Only the store's owner may create, update, or delete
// a service, enforced in the usecase.
func setupStoreServiceRoutes(srv *server.Server, service *handler.StoreServiceHandler, authMW middlewares.Middleware) {
	// Helper for wrapping protected handlers
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return middlewares.Chain(handlerFunc, authMW)
	}

	// Public
	srv.RegisterRoutes("GET /api/v1/stores/{storeId}/services", http.HandlerFunc(service.ListByStore))

	// Private
	srv.RegisterRoutes("POST /api/v1/stores/{storeId}/services", withAuth(service.Create))
	srv.RegisterRoutes("PATCH /api/v1/services/{id}", withAuth(service.Update))
	srv.RegisterRoutes("DELETE /api/v1/services/{id}", withAuth(service.Delete))
	srv.RegisterRoutes("PUT /api/v1/services/{serviceId}/schedules/{day}", withAuth(service.UpsertSchedule))
}

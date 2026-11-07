// Package server handles main execution
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/KevinManzano5/misto-backend/cmd/api/router"
	_ "github.com/KevinManzano5/misto-backend/docs"
	"github.com/KevinManzano5/misto-backend/internal/auth"
	"github.com/KevinManzano5/misto-backend/internal/cache"
	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/handler"
	"github.com/KevinManzano5/misto-backend/internal/logger"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/payment"
	"github.com/KevinManzano5/misto-backend/internal/repository/postgres"
	"github.com/KevinManzano5/misto-backend/internal/scheduler"
	"github.com/KevinManzano5/misto-backend/internal/server"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
	"github.com/joho/godotenv"
)

// @title Devboard API
// @version 1.0
// @description API REST to handle projects
// @contact.name support
// @host localhost
// @BasePath /api/v1
func main() {
	log := setupLogger()

	if err := godotenv.Load(); err != nil {
		log.Error(".env file not found")
	}

	port := os.Getenv("PORT")
	dbURL := os.Getenv("DB_URL")
	redisAddr := os.Getenv("REDIS_ADDR")

	pool, err := postgres.NewPool(context.Background(), postgres.Config{
		URL:             dbURL,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
	if err != nil {
		log.Error("db connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("db connection success")

	redisClient, err := cache.NewClient(context.Background(), cache.Config{
		Addr:    redisAddr,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Error("redis connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("redis connection cannot be closed", slog.Any("error", err))
		}
	}()
	log.Info("redis connection success")

	jwtSecret := os.Getenv("JWT_SECRET")
	accessTokenDuration := accessTokenDurationFromEnv(log)
	refreshTokenDuration := refreshTokenDurationFromEnv(log)

	jwtManager, err := auth.NewJWTManager(auth.JWTConfig{
		SecretKey:            []byte(jwtSecret),
		AccessTokenDuration:  accessTokenDuration,
		RefreshTokenDuration: refreshTokenDuration,
	})
	if err != nil {
		log.Error("error configuring JWT", slog.Any("error", err))
		os.Exit(1)
	}

	tokenStore := auth.NewTokenStore(redisClient)

	// Instantiation of repositories
	userRepo := postgres.NewUserRepository(pool)
	storeRepo := postgres.NewStoreRepository(pool)
	appointmentRepo := postgres.NewAppointmentRepository(pool)
	storeServiceScheduleRepo := postgres.NewStoreServiceScheduleRepository(pool)
	storeRatingRepo := postgres.NewStoreRatingRepository(pool)
	paymentTransactionRepo := postgres.NewPaymentTransactionRepository(pool)
	storeCategoryRepo := postgres.NewStoreCategoryRepository(pool)
	storeCategoryAssignmentRepo := postgres.NewStoreCategoryAssignmentRepository(pool)
	storeServiceRepo := postgres.NewStoreServiceRepository(pool)

	// Instantiation of use cases
	userUC := usecase.NewUserUseCase(userRepo)
	authUC := usecase.NewAuthUseCase(userRepo, jwtManager, tokenStore, refreshTokenDuration)
	storeUC := usecase.NewStoreUseCase(storeRepo, storeRatingRepo)
	paymentProcessor := payment.NewStubProcessor()
	appointmentUC := usecase.NewAppointmentUseCase(appointmentRepo, storeServiceScheduleRepo, storeRepo, storeServiceRepo, paymentProcessor, paymentTransactionRepo, log)
	storeRatingUC := usecase.NewStoreRatingUseCase(storeRatingRepo, appointmentRepo)
	storeCategoryUC := usecase.NewStoreCategoryUseCase(storeCategoryRepo)
	storeCategoryAssignmentUC := usecase.NewStoreCategoryAssignmentUseCase(storeCategoryAssignmentRepo, storeRepo)
	storeServiceUC := usecase.NewStoreServiceUseCase(storeServiceRepo, storeRepo, storeCategoryAssignmentRepo, storeServiceScheduleRepo)

	// Background jobs
	appointmentExpiryInterval := appointmentExpiryIntervalFromEnv(log)
	appointmentExpirer := scheduler.NewAppointmentExpirer(appointmentUC, appointmentExpiryInterval, log)
	expirerCtx, cancelExpirer := context.WithCancel(context.Background())
	defer cancelExpirer()
	go appointmentExpirer.Start(expirerCtx)

	// Utility configuration
	validate := validator.New()

	// Server configuration
	srv := server.New(port,
		server.WithLogger(log),
		server.WithReadTimeout(15*time.Second),
		server.WithWriteTimeout(30*time.Second),
	)

	// Global middlewares
	srv.UseChain(
		middlewares.Recovery(log),
		middlewares.Logger(log),
		middlewares.SecurityHeaders(middlewares.SecurityConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:5173",
			},
		}),
	)

	// Grouping of handlers
	handlers := &router.AppHandlers{
		Health:        handler.NewHealthHandler(),
		User:          handler.NewUserHandler(userUC, storeUC, validate, log),
		Auth:          handler.NewAuthHandler(authUC, validate, log),
		Store:         handler.NewStoreHandler(storeUC, storeCategoryAssignmentUC, validate, log),
		Appointment:   handler.NewAppointmentHandler(appointmentUC, storeRatingUC, validate, log),
		StoreRating:   handler.NewStoreRatingHandler(storeRatingUC, validate, log),
		StoreCategory: handler.NewStoreCategoryHandler(storeCategoryUC, log),
		StoreService:  handler.NewStoreServiceHandler(storeServiceUC, validate, log),
	}

	// Initialization of specific middlewares
	authMW := middlewares.RequireAuth(jwtManager, log)
	adminMW := middlewares.RequireRole(domain.RoleAdmin, log)
	optionalAuthMW := middlewares.OptionalAuth(jwtManager, log)

	// Routing injection
	router.SetupRoutes(srv, handlers, authMW, adminMW, optionalAuthMW)

	// Server startup
	if err := srv.Start(); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

// appointmentExpiryIntervalFromEnv reads APPOINTMENT_EXPIRY_INTERVAL (a Go duration string, e.g.
// "5m") and falls back to a 5 minute default when unset or invalid.
func appointmentExpiryIntervalFromEnv(log *slog.Logger) time.Duration {
	const defaultInterval = 5 * time.Minute

	return durationFromEnv(log, "APPOINTMENT_EXPIRY_INTERVAL", defaultInterval)
}

// accessTokenDurationFromEnv reads ACCESS_TOKEN_DURATION (a Go duration string, e.g. "15m") and
// falls back to a 15 minute default when unset or invalid.
func accessTokenDurationFromEnv(log *slog.Logger) time.Duration {
	const defaultDuration = 15 * time.Minute

	return durationFromEnv(log, "ACCESS_TOKEN_DURATION", defaultDuration)
}

// refreshTokenDurationFromEnv reads REFRESH_TOKEN_DURATION (a Go duration string, e.g. "168h") and
// falls back to a 7 day default when unset or invalid.
func refreshTokenDurationFromEnv(log *slog.Logger) time.Duration {
	const defaultDuration = 7 * 24 * time.Hour

	return durationFromEnv(log, "REFRESH_TOKEN_DURATION", defaultDuration)
}

// durationFromEnv reads a Go duration string from the given environment variable, falling back to
// defaultDuration when unset or invalid.
func durationFromEnv(log *slog.Logger, key string, defaultDuration time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultDuration
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		log.Error("invalid "+key+", using default", slog.Any("error", err), slog.Duration("default", defaultDuration))
		return defaultDuration
	}

	return duration
}

func setupLogger() *slog.Logger {
	if os.Getenv("GO_ENV") == "production" {
		return logger.New(logger.ProductionConfig())
	}
	return logger.New(logger.DefaultConfig())
}

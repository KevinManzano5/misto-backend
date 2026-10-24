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
	"github.com/KevinManzano5/misto-backend/internal/handler"
	"github.com/KevinManzano5/misto-backend/internal/logger"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/repository/postgres"
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
	refreshTokenDuration := 7 * 24 * time.Hour

	jwtManager, err := auth.NewJWTManager(auth.JWTConfig{
		SecretKey:            []byte(jwtSecret),
		AccessTokenDuration:  15 * time.Minute,
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
	storeServiceRepo := postgres.NewStoreServiceRepository(pool)
	appointmentRepo := postgres.NewAppointmentRepository(pool)

	// Instantiation of use cases
	userUC := usecase.NewUserUseCase(userRepo)
	authUC := usecase.NewAuthUseCase(userRepo, jwtManager, tokenStore, refreshTokenDuration)
	storeUC := usecase.NewStoreUseCase(storeRepo)
	storeServiceUC := usecase.NewStoreServiceUseCase(storeServiceRepo)
	appointmentUC := usecase.NewAppointmentUseCase(appointmentRepo)

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
		Health:       handler.NewHealthHandler(),
		User:         handler.NewUserHandler(userUC, validate, log),
		Auth:         handler.NewAuthHandler(authUC, validate, log),
		Store:        handler.NewStoreHandler(storeUC, validate, log),
		StoreService: handler.NewStoreServiceHandler(storeServiceUC, appointmentUC, validate, log),
		Appointment:  handler.NewAppointmentHandler(appointmentUC, validate, log),
	}

	// Initialization of specific middlewares
	authMW := middlewares.RequireAuth(jwtManager, log)

	// Routing injection
	router.SetupRoutes(srv, handlers, authMW)

	// Server startup
	if err := srv.Start(); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func setupLogger() *slog.Logger {
	if os.Getenv("GO_ENV") == "production" {
		return logger.New(logger.ProductionConfig())
	}
	return logger.New(logger.DefaultConfig())
}

// Package server handles main execution
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	httpSwagger "github.com/swaggo/http-swagger"
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
			log.Error(
				"redis connection cannot be closed",
				slog.Any("error", err),
			)
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

	userRepo := postgres.NewUserRepository(pool)
	storeRepo := postgres.NewStoreRepository(pool)
	storeServiceRepo := postgres.NewStoreServiceRepository(pool)
	appointmentRepo := postgres.NewAppointmentRepository(pool)

	userUC := usecase.NewUserUseCase(userRepo)
	authUC := usecase.NewAuthUseCase(userRepo, jwtManager, tokenStore, refreshTokenDuration)
	storeUC := usecase.NewStoreUseCase(storeRepo)
	storeServiceUC := usecase.NewStoreServiceUseCase(storeServiceRepo)
	appointmentUC := usecase.NewAppointmentUseCase(appointmentRepo)

	validate := validator.New()

	healthHandler := handler.NewHealthHandler()
	userHandler := handler.NewUserHandler(userUC, validate, log)
	authHandler := handler.NewAuthHandler(authUC, validate, log)
	storeHandler := handler.NewStoreHandler(storeUC, validate, log)
	storeServiceHandler := handler.NewStoreServiceHandler(storeServiceUC, validate, log)
	appointmentHandler := handler.NewAppointmentHandler(appointmentUC, validate, log)

	server := server.New(port,
		server.WithLogger(log),
		server.WithReadTimeout(15*time.Second),
		server.WithWriteTimeout(30*time.Second),
	)

	// * Middleware Auth
	authMW := middlewares.RequireAuth(jwtManager, log)

	// * Public routes
	server.RegisterRoutes("GET /docs/", httpSwagger.WrapHandler)
	server.RegisterRoutes("GET /health", healthHandler)
	server.RegisterRoutes("POST /api/v1/auth/sign_up", http.HandlerFunc(authHandler.SignUp))
	server.RegisterRoutes("POST /api/v1/auth/sign_in", http.HandlerFunc(authHandler.SignIn))
	server.RegisterRoutes("POST /api/v1/auth/sign_out", http.HandlerFunc(authHandler.SignOut))
	server.RegisterRoutes("POST /api/v1/auth/refresh", http.HandlerFunc(authHandler.Refresh))
	server.RegisterRoutes("GET /api/v1/users/{id}", http.HandlerFunc(userHandler.Get))
	server.RegisterRoutes("GET /api/v1/stores/{id}", http.HandlerFunc(storeHandler.GetStoreByID))
	server.RegisterRoutes("GET /api/v1/stores", http.HandlerFunc(storeHandler.ListStores))
	server.RegisterRoutes("GET /api/v1/stores/{storeId}/services", http.HandlerFunc(storeServiceHandler.ListByStoreID))
	server.RegisterRoutes("GET /api/v1/services/{id}", http.HandlerFunc(storeServiceHandler.GetByID))

	// * Private routes
	server.RegisterRoutes("POST /api/v1/stores", middlewares.Chain(http.HandlerFunc(storeHandler.Create), authMW))
	server.RegisterRoutes("PATCH /api/v1/stores/{id}", middlewares.Chain(http.HandlerFunc(storeHandler.UpdateStore), authMW))
	server.RegisterRoutes("DELETE /api/v1/stores/{id}", middlewares.Chain(http.HandlerFunc(storeHandler.DeleteStore), authMW))
	server.RegisterRoutes("POST /api/v1/stores/{storeId}/services", middlewares.Chain(http.HandlerFunc(storeServiceHandler.Create), authMW))
	server.RegisterRoutes("PATCH /api/v1/services/{id}", middlewares.Chain(http.HandlerFunc(storeServiceHandler.Update), authMW))
	server.RegisterRoutes("DELETE /api/v1/services/{id}", middlewares.Chain(http.HandlerFunc(storeServiceHandler.Delete), authMW))
	server.RegisterRoutes("POST /api/v1/appointments", middlewares.Chain(http.HandlerFunc(appointmentHandler.Create), authMW))

	// * Protected routes
	server.UseChain(
		middlewares.Recovery(log),
		middlewares.Logger(log),
		middlewares.SecurityHeaders(middlewares.SecurityConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:5173",
			},
		}),
	)

	if err := server.Start(); err != nil {
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

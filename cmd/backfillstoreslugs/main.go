// Package main runs a one-off backfill: computes and persists a slug for every store created
// before the slug column existed (migration 000023), reusing domain.SlugFromName — the exact
// same logic StoreUseCase.Create/Update already use — so backfilled slugs are generated
// identically to new ones. Run with: go run ./cmd/backfillstoreslugs
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/logger"
	"github.com/KevinManzano5/misto-backend/internal/repository/postgres"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/joho/godotenv"
)

func main() {
	log := logger.New(logger.DefaultConfig())

	if err := godotenv.Load(); err != nil {
		log.Error(".env file not found")
	}

	pool, err := postgres.NewPool(context.Background(), postgres.Config{
		URL:             os.Getenv("DB_URL"),
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
	if err != nil {
		log.Error("db connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	storeRepo := postgres.NewStoreRepository(pool)
	scheduleRepo := postgres.NewStoreScheduleRepository(pool)
	storeUC := usecase.NewStoreUseCase(storeRepo, scheduleRepo)

	updated, err := storeUC.BackfillMissingSlugs(context.Background())
	if err != nil {
		log.Error("backfill failed", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("slug backfill complete", slog.Int("stores_updated", updated))
}

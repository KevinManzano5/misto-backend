# Variables
APP_NAME=devboard
MAIN_PATH=./cmd/api
MIGRATE_PATH=./migrations
DB_URL=postgresql://postgres:password@localhost:5432/misto?sslmode=disable
JWT_SECRET=wr356wjUELHijQUc2WI9caKOm019ydb22a

.PHONY: run build test lint migrate-up migrate-down generate tidy help docs

## run: Run app
run:
	GO_ENV=development DATABASE_URL=$(DB_URL) REDIS_ADDR=localhost:6379 JWT_SECRET=$(JWT_SECRET) go run $(MAIN_PATH)/main.go

## build: Create binary
build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

## test: Run all tests with race detector
test:
	go test -race -cover ./...

## lint: Scan all code with golangci-lint
lint:
	golangci-lint run ./...

## migrate-create: creates a new migration
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

## migrate-up: Apply all pending migrations
migrate-up:
	migrate -path $(MIGRATE_PATH) -database "$(DB_URL)" up

## migrate-down: Revert last migration
migrate-down:
	migrate -path $(MIGRATE_PATH) -database "$(DB_URL)" down 1

## generate: Execute go generate in all the project
generate:
	sqlc generate
	go generate ./...

## tidy: Clean and check dependencies
tidy:
	go mod tidy
	go mod verify

## help: Show this menu
help:
	@grep -E '^##' Makefile | sed 's/## //'

## docker-up: Up all development services
docker-up:
	docker compose up -d

## docker-down: Stop all development services
docker-down: 
	docker compose down

## docker-logs: See logs of all services
docker-logs:
	docker compose logs -f

## docs: generate OpenAPI documentation with swaggo
docs:
	swag init -g cmd/api/main.go -o docs
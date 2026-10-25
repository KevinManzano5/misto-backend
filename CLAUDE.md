# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Reglas de desarrollo obligatorias (se cargan siempre):
@.claude/rules/golang-standards.md
@.claude/rules/backend-standards.md

## What this is

A Go REST API backend (module `github.com/KevinManzano5/misto-backend`) for a store/appointment booking system ("devboard"/"misto"): users, stores, store services, store schedules, and appointments, with JWT auth and Redis-backed refresh tokens.

## Common commands

All common tasks are defined in the `Makefile` (run `make help` to list them):

- `make run` — run the API locally (sets `GO_ENV`, `DATABASE_URL`, `REDIS_ADDR`, `JWT_SECRET` from Makefile vars and loads `.env`)
- `make build` — build binary to `bin/devboard`
- `make test` — `go test -race -cover ./...`
- `make lint` — `golangci-lint run ./...` (config in `.golangci.yml`; the generated sqlc code under `internal/repository/postgres/generated` is excluded from all linters)
- `make generate` — runs `sqlc generate` (regenerates `internal/repository/postgres/generated/*`) then `go generate ./...`
- `make docs` — regenerates Swagger docs via `swag init -g cmd/api/main.go -o docs`
- `make tidy` — `go mod tidy && go mod verify`
- `make docker-up` / `make docker-down` / `make docker-logs` — manage local Postgres + Redis (see `docker-compose.yml`)
- `make migrate-create name="..."` — create a new up/down migration pair in `migrations/`
- `make migrate-up` / `make migrate-down` — apply/revert migrations against `DB_URL`

Run a single test package/test the normal Go way, e.g. `go test ./internal/usecase/... -run TestName -race`.

Local infra (Postgres on `5432`, Redis on `6379`) is started via `docker-compose.yml`; real runtime config comes from `.env` (see `.env.example`), not the Makefile's hardcoded defaults.

## Database workflow (sqlc + golang-migrate)

This project hand-writes SQL and generates typed Go from it — there is no ORM.

1. Add/modify a migration pair in `migrations/` (`make migrate-create name="..."`), write the `.up.sql`/`.down.sql`.
2. Apply it locally with `make migrate-up` (if a migration fails partway, `migrate ... force <version>` then fix and re-run — see `md/MIGRATIONS.md`).
3. Write/update queries as `.sql` files in `internal/repository/postgres/queries/` (see `md/QUERIES.md`).
4. Run `make generate` (sqlc) to regenerate code in `internal/repository/postgres/generated/`. **Never hand-edit files in `generated/`** — they're overwritten and excluded from lint.
5. Wire the generated query methods into a repository struct in `internal/repository/postgres/` (not `generated/`), which converts between sqlc's generated types (`pgtype.UUID`, `pgtype.Time`, etc.) and domain types.

sqlc config is in `sqlc.yaml` (schema source: `migrations/`, queries source: `internal/repository/postgres/queries/`, pgx/v5 driver).

## Architecture

Layered/clean-architecture style, wired together manually (no DI framework) in `cmd/api/main.go`:

```
handler  →  usecase  →  repository (interface)  →  postgres (implementation, generated sqlc)
                ↑
             domain (entities, sentinel errors — imported by all layers, depends on none)
```

- **`internal/domain`**: plain structs (`Appointment`, `Store`, `StoreSchedule`, `User`, ...) and shared sentinel errors (`domain.ErrNotFound`, `ErrAlreadyExists`, `ErrUnauthorized`, `ErrForbidden`, `ErrInvalidInput`, `ErrInternal` in `errors.go`). Every layer wraps these with `fmt.Errorf("...: %w", domain.ErrX)` rather than inventing new error types, so `errors.Is` works end-to-end up to the HTTP layer.
- **`internal/usecase`**: business logic. Each usecase defines its own narrow repository interface(s) (e.g. `AppointmentUseCase` depends on `AppointmentRepository` and `StoreScheduleRepository` interfaces declared in `appointment_usecase.go`, not on the concrete postgres struct) — interfaces live next to their consumer, not next to the implementation.
- **`internal/repository/postgres`**: concrete repository structs wrapping a `*pgxpool.Pool` and the sqlc-generated `*Queries`. Responsible for converting sqlc/pgtype values ↔ domain structs (see `toDomainStoreSchedule`, `toPgTime`/`fromPgTime` patterns in `store_schedules.go`).
- **`internal/repository/postgres/generated`**: sqlc output — never edit by hand, regenerate via `make generate`.
- **`internal/handler`**: HTTP handlers (stdlib `net/http`, Go 1.22+ method+pattern mux syntax, e.g. `"POST /api/v1/stores/{id}"`). Handlers validate input (`internal/validator`), call a usecase, and translate results/errors to HTTP via `handler.RespondJSON` / `handler.RespondError`.
- **`internal/server`**: thin wrapper around `*http.Server` + `*http.ServeMux` with graceful shutdown on SIGINT/SIGTERM (`Start()`), `RegisterRoutes`, and middleware chaining (`Use`/`UseChain`).
- **`cmd/api/router`**: `router.SetupRoutes` registers all routes grouped by domain area (`setupSystemRoutes`, `setupAuthRoutes`, `setupStoreRoutes`, `setupAppointmentRoutes`), composing public vs. auth-protected routes via a local `withAuth` helper that wraps a handler with `middlewares.Chain(handlerFunc, authMW)`.
- **`internal/middlewares`**: `Recovery`, `Logger`, `SecurityHeaders` (CORS/security headers, applied globally in `main.go`), and `RequireAuth`/`RequireRole` (JWT-based, applied per-route in the router). `RequireAuth` stores `*auth.Claims` in request context; read it back with `middlewares.GetClaims(r)` / `middlewares.GetUserID(r)`.
- **`internal/auth`**: JWT issuing/verification (`JWTManager`), password hashing, and a Redis-backed refresh `TokenStore`.
- **`internal/cache`**: Redis client setup.

### Error → HTTP mapping

`handler.RespondError` (in `internal/handler/response.go`) is the single place that maps domain sentinel errors to RFC7807-style `application/problem+json` responses (404/409/401/403/400/500). Validation failures use `ValidationErrorContext` wrapping `domain.ErrInvalidInput` to carry field-level `[]validator.ValidationError` details through to the response. Any error not matching a known sentinel is logged and returned as a generic 500 — don't swallow unexpected errors silently in handlers/usecases.

### Adding a new resource/endpoint

The established pattern (follow the shape of the `store_schedule`/`appointment` code):
1. Domain struct in `internal/domain/<name>.go`.
2. Migration + sqlc query file + `make generate`.
3. Repository struct in `internal/repository/postgres/<name>.go` implementing the usecase-defined interface.
4. Usecase in `internal/usecase/<name>_usecase.go`, declaring the repo interface(s) it needs.
5. Handler in `internal/handler/<name>_handler.go` using `validator.New()` + `handler.RespondJSON`/`RespondError`.
6. Wire construction in `cmd/api/main.go` and routes in `cmd/api/router/routes.go` (new `setupXRoutes` function, grouped public/private, using `withAuth` for protected routes).

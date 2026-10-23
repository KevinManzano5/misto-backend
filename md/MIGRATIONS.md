# Migrations

## How to create migrations

1. Create migrations files with `make migrate-create name="migration_name"`
2. Write SQL in both files
3. Execute migrations in database with `make migrate-up`
4. Check changes in database
5. If execution went wrong check SQL and fix migration with `migrate -path ./migrations -database "postgresql://postgres:password@localhost:5432/misto?sslmode=disable" force 2`
6. Execute again `make migrate-up`

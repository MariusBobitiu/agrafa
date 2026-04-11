ifneq (,$(wildcard .env.local))
include .env.local
export
endif

ifneq (,$(wildcard .env))
include .env
export
endif

POSTGRES_MIGRATIONS_URI := $(POSTGRES_URI)$(if $(findstring ?,$(POSTGRES_URI)),&,?)x-migrations-table=%22agrafa_meta%22.%22schema_migrations%22&x-migrations-table-quoted=true
SEED_FILE ?= src/db/seeds/app/001_test_project_node.sql
SWAG ?= swag

.PHONY: run check-postgres-uri migrate-up migrate-down seed migrate-up-app migrate-down-app seed-app sqlc-generate swagger-generate

run:
	go run .

check-postgres-uri:
	@test -n "$(POSTGRES_URI)" || (echo "POSTGRES_URI is required" && exit 1)

migrate-up: check-postgres-uri
	psql "$(POSTGRES_URI)" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA IF NOT EXISTS \"agrafa_meta\""
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path src/db/migrations/app -database "$(POSTGRES_MIGRATIONS_URI)" up

migrate-down: check-postgres-uri
	psql "$(POSTGRES_URI)" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA IF NOT EXISTS \"agrafa_meta\""
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path src/db/migrations/app -database "$(POSTGRES_MIGRATIONS_URI)" down -all
	psql "$(POSTGRES_URI)" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS \"agrafa_meta\" CASCADE"

seed: check-postgres-uri
	psql "$(POSTGRES_URI)" -v ON_ERROR_STOP=1 -f "$(SEED_FILE)"

migrate-up-app: migrate-up

migrate-down-app: migrate-down

seed-app: seed

sqlc-generate:
	sqlc generate

swagger-generate:
	$(SWAG) init --parseInternal -g main.go -o docs

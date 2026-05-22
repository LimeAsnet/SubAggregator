.PHONY: help docker-up docker-up-db docker-down docker-logs docker-logs-app migrate-up migrate-down migrate-version migrate-force migrate-drop swagger test test-integration run build tidy

MIGRATE := go run ./cmd/migrate
APP     := go run ./cmd/app

help:
	@echo "Targets:"
	@echo "  docker-up          Build and start full stack (postgres + migrate + app)"
	@echo "  docker-up-db       Start only PostgreSQL"
	@echo "  docker-down        Stop all containers"
	@echo "  docker-logs        Tail logs (all services)"
	@echo "  docker-logs-app    Tail app logs"
	@echo "  migrate-up         Apply migrations"
	@echo "  migrate-down       Roll back last migration"
	@echo "  migrate-version    Show migration version"
	@echo "  migrate-drop       Drop all tables (destructive)"
	@echo "  swagger            Generate Swagger docs (docs/)"
	@echo "  test               Run unit tests (no integration)"
	@echo "  test-integration   Run integration tests (requires Postgres)"
	@echo "  run                Run API server"
	@echo "  build              Build binaries"
	@echo "  tidy               go mod tidy"
	@echo "  dev                postgres in Docker + migrate + run app locally"

docker-up:
	docker compose up -d --build

docker-up-db:
	docker compose up -d postgres

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-logs-app:
	docker compose logs -f app

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-version:
	$(MIGRATE) version

migrate-force:
	@test -n "$(VERSION)" || (echo "Usage: make migrate-force VERSION=1" && exit 1)
	$(MIGRATE) force $(VERSION)

migrate-drop:
	$(MIGRATE) drop

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs -d cmd/app,internal/handlers,internal/models,internal/service --parseInternal

test:
	go test ./internal/service/... ./internal/handlers/... -count=1

test-integration:
	go test -tags=integration ./internal/handlers/integration/... -count=1 -v

run:
	$(APP)

build:
	go build -o bin/app ./cmd/app
	go build -o bin/migrate ./cmd/migrate

tidy:
	go mod tidy

dev: docker-up-db migrate-up run

# Makefile — echobackend dev shortcuts.
# Windows: run via Git Bash / WSL / Chocolatey make, or use the commands
# from AGENTS.md directly.
#
# Common flow:
#   make up && make migrate-up   # first time: start services + apply schema
#   make dev                     # hot-reload server (needs `air`)

.DEFAULT_GOAL := help

GOOSE := goose

.PHONY: help up down down-clean logs ps \
	dev run build test test-race cover vet fmt lint sec check tidy \
	migrate-up migrate-down migrate-status migrate-create

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## --- Local services (docker compose) ---------------------------------------

up: ## Start Postgres + Redis + RustFS (waits for healthchecks)
	docker compose up -d --wait

down: ## Stop local services
	docker compose down

down-clean: ## Stop services and DELETE all data volumes
	docker compose down -v

logs: ## Tail service logs
	docker compose logs -f

ps: ## Show service status
	docker compose ps

## --- App -------------------------------------------------------------------

dev: ## Run server with hot reload (requires air)
	air

run: ## Run server
	go run cmd/main.go

build: ## Build binary to bin/main
	go build -o bin/main cmd/main.go

## --- Quality gates (mirrors CI) --------------------------------------------

test: ## Run all tests (no DB required)
	go test ./...

test-race: ## Run tests with race detector
	go test -race ./...

cover: ## Run tests with coverage report (coverage.out)
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

vet: ## Static analysis
	go vet ./...

fmt: ## Format code (gofmt + goimports via golangci-lint)
	golangci-lint fmt ./...

lint: ## Lint (golangci-lint)
	golangci-lint run ./...

sec: ## Security scan (requires gosec)
	gosec ./...

check: vet test lint ## Run vet + test + lint (same as CI)

tidy: ## Tidy and verify module deps
	go mod tidy

## --- Migrations (goose, reads GOOSE_* from .env) ---------------------------

migrate-up: ## Apply pending migrations
	$(GOOSE) up

migrate-down: ## Roll back one migration
	$(GOOSE) down

migrate-status: ## Show migration status
	$(GOOSE) status

migrate-create: ## Create a new SQL migration: make migrate-create name=add_index
	$(GOOSE) create $(name) sql

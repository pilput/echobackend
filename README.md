# Echo Backend API

A modern, robust REST API built with Go 1.26, Echo v5, GORM, and PostgreSQL. Designed for scalability with manual dependency injection and high performance via Valkey caching.

## Core Features

- **High Performance**: Built on Echo v5 with optimized middleware.
- **Robust Persistence**: GORM with PostgreSQL (using `pgx/v5` driver).
- **Fast Caching**: Valkey (Redis-compatible) integration for high-traffic endpoints.
- **Secure Auth**: JWT-based authentication with custom middleware.
- **Reliable Storage**: S3/MinIO compatible storage for file management.
- **Validation**: Strict request validation using `go-playground/validator/v10`.
- **Modern Keys**: UUID primary keys across all models (v7 preferred).
- **Rate Limiting**: Global and per-route rate limiting to protect critical endpoints.
- **Graceful Shutdown**: Automatic resource cleanup on server termination.
- **Health Checks**: Dedicated `/health` endpoint for Docker HEALTHCHECK and load balancers.

## Tech Stack

- **Runtime**: Go 1.26
- **Web Framework**: [Echo v5](https://github.com/labstack/echo)
- **ORM**: [GORM](https://gorm.io/) v2
- **Database**: PostgreSQL 18+
- **Cache**: Valkey/Redis
- **Storage**: MinIO / AWS S3
- **Migrations**: [Goose](https://github.com/pressly/goose) (Raw SQL)

## Quick Start

```bash
# 1. Clone and setup environment
cp .env.example .env

# 2. Edit .env with your local configuration
#    (set JWT_SECRET; for local services below also set REDIS_URL=redis://localhost:6379
#     and S3_USE_SSL=false)

# 3. Start local services (Postgres 18, Valkey, MinIO)
docker compose up -d --wait

# 4. Apply database migrations
goose up

# 5. Run with hot reload (requires air)
air

# 6. Or run normally
go run cmd/main.go
```

The server starts at `http://localhost:8080`. A `Makefile` wraps common tasks (`make help`); on Windows use Git Bash/WSL or run the underlying commands directly.

## Commands Reference

| Task | Command |
|------|---------|
| **Build Binary** | `go build -o bin/main cmd/main.go` |
| **Run Server** | `go run cmd/main.go` |
| **Hot Reload** | `air` |
| **Run All Tests** | `go test ./...` |
| **Race Check** | `go test -race ./...` |
| **Test Coverage** | `go test -cover ./...` |
| **Static Analysis**| `go vet ./...` |
| **Format Code** | `go fmt ./...` |
| **Linting** | `golangci-lint run` |
| **Security Scan** | `gosec ./...` |
| **Start Local Services** | `docker compose up -d --wait` (or `make up`) |
| **CI-equivalent Check** | `make check` (vet + test + lint) |

## Environment Variables

The application requires the following mandatory environment variables to start:

| Variable | Fallback Alias | Description |
|----------|----------------|-------------|
| `DATABASE_URL` | - | PostgreSQL connection string (DSN) |
| `JWT_SECRET` | - | Secret key for JWT signing & verification |

Other optional configurations (S3, Valkey, Rate Limiting, SMTP email, etc.) are documented in [`.env.example`](.env.example).

## Architecture

The project follows a modular layered architecture with manual dependency injection:

- **`internal/di/`**: Centralized Dependency Injection container (`container.go`).
- **`internal/handler/`**: Request handling and response formatting via `pkg/response`.
- **`internal/service/`**: Core business logic and service orchestration.
- **`internal/repository/`**: Data access layer using GORM.
- **`internal/model/`**: GORM entities and shared domain models.
- **`internal/platform/`**: App-owned infrastructure adapters for cache, database, email, queue, and storage.
- **`internal/apperror/`**: Shared application error sentinels.
- **`pkg/`**: Reusable helper packages such as market data, response formatting, and validation.

## API Documentation

Full HTTP API reference for frontend integration lives in [`docs/api/openapi.yaml`](docs/api/openapi.yaml) — an OpenAPI 3.1 document, split per-module (auth, users, posts, tags, chat, holdings, exchange rates, bookmarks, notifications, admin reports) into small files under `docs/api/paths/` and `docs/api/schemas/`. See [`docs/api/README.md`](docs/api/README.md) for the layout and how to preview, lint, and generate clients from it.

### Standardized Responses

All handlers use `pkg/response` helpers to ensure a consistent API contract:
```go
return response.Success(c, "Data retrieved", data)
return response.ValidationError(c, "Invalid input", err)
```

## Database Migrations

Managed via [goose](https://github.com/pressly/goose). Configuration is automatically picked up from `.env`.

```bash
# Apply all pending migrations
goose up

# Rollback the last migration
goose down

# Check migration history
goose status

# Create a new migration file
goose create <migration_name> sql
```

## Deployment

CI via GitHub Actions:

1. **Test & lint** (`go vet`, `go test`, `golangci-lint`) on every PR and push to `main`
2. **Docker build & push** (push to `main` only, after tests pass) — tags `latest`, `sha-<12-char-sha>`, and full-commit SHA

Pull a pinned `sha-*` image for reproducible deploys (prefer that over floating `latest`).

```bash
# Local Docker build test
docker build -t cecep31/echobackend .

# Run container locally
docker run -p 8080:8080 --env-file .env cecep31/echobackend
```

## License

MIT

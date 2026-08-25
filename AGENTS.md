# AGENTS.md — echobackend

## Commands

```bash
# Local dev services (Postgres 18 + Valkey + MinIO via docker-compose.yml)
docker compose up -d --wait # Start services (or: make up). Creates the `custom` schema automatically.
docker compose down         # Stop (or: make down; make down-clean also wipes data)
make help                   # All shortcuts: up, dev, test, lint, check, migrate-*, ...

go run cmd/main.go          # Run server (requires .env with DATABASE_URL + JWT_SECRET >= 32 chars)
air                         # Hot reload (reads .env automatically)
go test ./...               # All tests (service + pkg layers only; no DB integration tests)
go test ./internal/service/...  # Service tests only
go test ./pkg/...           # Reusable package tests only
go test -race ./...         # Race checker
go vet ./...                # Static analysis
golangci-lint run           # Lint (default + enabled linters, see .golangci.yml)
golangci-lint fmt ./...     # Format (= make fmt; gofmt + goimports, the canonical formatter)
gosec ./...                 # Security scan

make check                  # vet + test + lint — same gate as CI
```

Make only works under Git Bash/WSL/Chocolatey on Windows — otherwise run the underlying commands directly (`make help` lists all targets).

```bash
# Migrations (requires .env with GOOSE_* vars)
goose up                    # Apply pending
goose down                  # Rollback one
goose status                # Check current
goose create <name> sql     # New migration file

# First-time setup: goose stores version history in the `custom` schema.
# `docker compose up` already creates it via scripts/init-db.sql.
# For an external Postgres, run this once before the first `goose up`:
psql "$DATABASE_URL" -c 'CREATE SCHEMA IF NOT EXISTS custom;'
```

## Architecture

- **Framework**: Echo **v5** (not v4). API differs — handlers receive `*echo.Context` (pointer).
- **Entry point**: `cmd/main.go` — loads config → creates DI container → registers routes → starts server with graceful shutdown.
- **DI**: Manual wiring in `internal/di/container.go`. All handler/service/repo instances created there.
- **Layering**: `handler` → `service` → `repository`. No DI framework.
- **`internal/platform/`**: App-owned infrastructure adapters (`cache`, `database`, `email`, `queue`, `storage`).
- **`pkg/`**: Reusable helper packages (`applog`, `market`, `response`, `validator`).
- **`internal/model/`**: GORM entities (domain models).
- **`internal/repository/`**: Data access layer using GORM.
- **`internal/dto/`**: Request/response structs. `internal/apperror/` for shared app error sentinels.
- **API routes**: All under `/api/*`. A single `Routes` struct (`internal/routes/routes.go`) wires handler dependencies; per-module `setupXxxRoutes` methods live in `*Routes.go` files. Auth-protected routes use `r.authMiddleware.Auth()`.
- **Health**: `GET /health` — pings DB (200/503). Used by Docker HEALTHCHECK and load balancers.
- **Auth gates**: routes apply `r.authMiddleware.Auth()`, and admin routes chain `r.authMiddleware.AuthAdmin()` after it (e.g. `posts.PUT("/:id", h.UpdatePost, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())`).
- **Pagination**: Use `handler.ParsePaginationParams(c, defaultLimit)` — returns `(limit, offset)`, max cap 100. Build response meta with `response.CalculatePaginationMeta(total, offset, limit)` and pass via `response.SuccessWithMeta`.
- **Docs**: endpoint reference lives in `docs/api/`, migration notes in `migrations/README.md`.

## Config & Env

- Config loaded via `config.Load()` — reads `.env` (godotenv) then environment variables.
- **Required**: `DATABASE_URL`, `JWT_SECRET` (must be ≥ 32 chars). App panics if missing/invalid.
- Many keys accept **fallback aliases** (legacy names). First-set key wins. See `config/config.go` for full list.
- `GOOSE_TABLE=custom.goose_migrations` — non-default goose table location; create the `custom` schema (`psql "$DATABASE_URL" -c 'CREATE SCHEMA IF NOT EXISTS custom;'`) once before the first `goose up`.
- Cache primary key is **`REDIS_URL`** (`VALKEY_URL` is a fallback alias) — leave empty to disable. Caching is fail-open: `NewRedisCache` returns nil and the app runs without it.
- External integrations are **disabled by default** when their env keys are empty, not errors: GitHub OAuth (`GITHUB_CLIENT_*` → `ErrOAuthNotConfigured`), OpenRouter AI chat (`OPENROUTER_API_KEY` — stream endpoints echo the user message), SMTP email (`SMTP_HOST`), RapidAPI market data (`RAPIDAPI_KEY`), Asynq queue (`QUEUE_REDIS_URL`).

## Testing

- Tests exist mostly in `internal/service/`, `internal/handler/`, `internal/middleware/`, `config/`, and `pkg/`. No repository or DB integration tests.
- **No external test dependencies** — service tests use hand-written mocks (`internal/service/mocks_test.go`). No mockgen or code-gen.
- No testcontainers or integration test harness. Running `go test ./...` does not require PostgreSQL.
- Test file pattern: `*_test.go` in the same package (white-box).

## Response Format

All handlers use `pkg/response` for consistent JSON:

```go
response.Success(c, "message", data)        // 200
response.Created(c, "message", data)         // 201
response.ValidationError(c, "msg", err)       // 422
response.BadRequest(c, "msg", err)            // 400
response.NotFound(c, "msg", err)              // 404
response.Unauthorized(c, "msg")              // 401
response.Forbidden(c, "msg")                 // 403
response.Conflict(c, "msg", conflictErr)      // 409
response.InternalServerError(c, "msg", err)  // 500 — err logged server-side only, never sent to client
response.FromValidateError(c, err)            // 422 with structured field errors
```

Use `response.TooManyRequests(c, "msg")` for 429 (rate limit). `response.Conflict` takes a string reason, not an `error`.

## CI

`.github/workflows/main.yml` runs on PRs and pushes to `main`:

1. **test** — `go vet ./...`, `go test ./...`, `golangci-lint` (pinned to v2.12.2)
2. **docker** (push to `main` only, after test) — build & push `cecep31/echobackend:latest`, `:sha-<12-char>`, and `:sha-<full>`

Still useful locally before pushing: `go test ./...`, `go vet ./...`, `golangci-lint run`.

## Migrations

- Goose with **raw SQL** files in `migrations/` (`001_init_schema.sql`, `002_add_post_views_and_user_follows.sql`, …).
- Uses PostgreSQL features: triggers for count fields, `uuid_generate_v4()` (v7 preferred), `ON DELETE CASCADE`, soft deletes via `deleted_at`.
- **New migrations**: `goose create <name> sql` (always `sql`, never `go`).

## Deployment

- GitHub Actions on push to `main`: test/lint → Docker build → push to Docker Hub (`cecep31/echobackend:latest` and `:sha-*`).
- Docker image is built with Go 1.26, multi-stage, runs as non-root user. Pull a specific `sha-*` tag for reproducible deploys.

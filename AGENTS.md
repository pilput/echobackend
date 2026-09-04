# AGENTS.md — echobackend

## Commands

```bash
# First-time setup: start services → apply schema → run
docker compose up -d --wait  # or: make up. Postgres 18 + Valkey + RustFS; creates `custom` schema via scripts/init-db.sql
goose up                     # apply migrations (needs GOOSE_* vars from .env)
go run cmd/main.go           # run server (or: air for hot reload; make dev / make run)

go test ./...                # all tests — no DB required (service + pkg layers only, hand-written mocks)
go test ./internal/service/...  # single package
go test -race ./...          # race checker
go vet ./...                 # static analysis
golangci-lint run            # lint (see .golangci.yml)
golangci-lint fmt ./...      # canonical formatter (= make fmt; gofmt + goimports)
make check                   # vet + test + lint — same gate as CI
```

Make only works under Git Bash/WSL/Chocolatey on Windows — otherwise run the underlying commands directly (`make help` lists all targets).

```bash
goose down                    # roll back one
goose status                  # check current
goose create <name> sql       # new migration (always `sql`, never `go`)
# External Postgres only (compose already handles this): create schema once before first `goose up`
psql "$DATABASE_URL" -c 'CREATE SCHEMA IF NOT EXISTS custom;'
```

Local `.env` after first `docker compose up`: set `REDIS_URL=redis://localhost:6379` and `S3_USE_SSL=false` (local RustFS speaks plain HTTP).

## Architecture

- **Framework**: Echo **v5** (not v4) — handlers receive `*echo.Context` (pointer).
- **Entry**: `cmd/main.go` — `config.Load()` → `di.NewContainer` → `container.Routes.Setup(e)` → server with graceful shutdown (10s) + resource cleanup (5s).
- **DI**: manual wiring in `internal/di/container.go`. No DI framework. New handler/service/repo must be wired there and passed to `routes.NewRoutes`.
- **Layering**: `handler` → `service` → `repository`. `internal/model/` = GORM entities, `internal/dto/` = request/response structs, `internal/apperror/` = shared error sentinels.
- **`internal/platform/`** = app-owned infra adapters (`cache`, `database`, `email`, `queue`, `storage`). **`pkg/`** = reusable helpers (`response`, `validator`, `applog`, `market`).
- **Routes**: all under `/api/*`. Single `Routes` struct (`internal/routes/routes.go`); per-module `setupXxxRoutes` live in `*Routes.go` files.
- **Auth**: `r.authMiddleware.Auth()` for login-required; admin routes must chain `Auth()` **first**, then `AuthAdmin()` (`AuthAdmin` reads `c.Get("user")` set by `Auth()` and returns 401 without it). `OptionalAuth()` exists for public-but-personalized endpoints. Claims shortcut `is_super_admin` is a fast path only — DB (`GetAdminByID`) is authoritative.
- **Health**: `GET /health` pings DB (200/503). `GET /` returns Hello World via `response.Success`.
- **Docs**: endpoint reference in `docs/api/`, migration notes in `migrations/README.md`.

## Config & Env

- `config.Load()` reads `.env` (stdlib loader, best-effort) then env vars; system env always wins. Many keys accept fallback aliases (first-set wins) — see `config/config.go`.
- **Required**: `DATABASE_URL`, `JWT_SECRET` (≥ 32 chars). App panics otherwise.
- `GOOSE_TABLE=custom.goose_migrations` — non-default table; `custom` schema must exist before first `goose up`.
- Cache primary key is **`REDIS_URL`** (`VALKEY_URL` alias) — empty disables caching. Fail-open: `NewRedisCache` returns nil and app runs without it. `QUEUE_REDIS_URL` falls back to `REDIS_URL`/`VALKEY_URL` when empty.
- Integrations degrade to disabled/echo when keys are empty (not errors): GitHub OAuth (`GITHUB_CLIENT_*` → `ErrOAuthNotConfigured`), OpenRouter (`OPENROUTER_API_KEY` — streams echo user message), SMTP (`SMTP_HOST` — reset links stay in activity metadata), RapidAPI market data (`RAPIDAPI_KEY`).

## Handler conventions

- All responses via `pkg/response` (`success` envelope, `omitempty` on data/meta/error/errors):
  `Success` (200), `Created` (201), `BadRequest` (400), `Unauthorized`/`Forbidden` (fixed `"Unauthorized access"`/`"Access forbidden"` strings), `NotFound` (404), `Conflict` (409 — takes a **string** reason, not `error`), `ValidationError`/`FromValidateError` (422), `TooManyRequests` (429), `InternalServerError` (500 — logs `err` server-side only, never sends it to client).
- Validate with `c.Validate(dto)` (custom validator set in `main.go`, includes `free_model` tag) → on error `return response.FromValidateError(c, err)`.
- Pagination: `limit, offset := handler.ParsePaginationParams(c, defaultLimit)` (cap 100; `GET /api/posts` is the known exception accepting larger) → `meta := response.CalculatePaginationMeta(total, offset, limit)` → `response.SuccessWithMeta(c, msg, data, meta)`.
- Global body limit 10 MB (larger → 413); server read/write timeouts 60s (see `cmd/main.go` before changing).

## Database gotchas

- Goose + raw SQL in `migrations/`. New PK defaults are **`uuidv7()`** (Postgres 18+); migration 009 switched from v4, 010 dropped `uuid-ossp`.
- Triggers maintain `view/like/bookmark/followers/following_count` — don't update counts by hand.
- Soft deletes via `deleted_at` apply to `users`, `post_views`, `post_comments`, `user_follows`, `files`, `chat_conversations` only. **`posts` and `post_likes` are hard-deleted since 014** (destructive migration that also purges old soft-deleted rows).
- `.Table(...)` opts out of GORM's soft-delete scope — add `deleted_at IS NULL` by hand in raw/`Table` queries (see `post_repository.go`, `report_repository.go`).

## Testing

- `go test ./...` needs no Postgres/Valkey. No testcontainers, no repo/DB integration tests, no mockgen — hand-written mocks in `internal/service/mocks_test.go`.
- White-box `*_test.go` in same package. Tests live mostly in `internal/service/`, `internal/handler/`, `internal/middleware/`, `config/`, `pkg/`.

## CI & deploy

- `.github/workflows/main.yml` on PRs + pushes to `main`: `go vet ./...` → `go test ./...` → `golangci-lint` (**pinned v2.12.2**). Run `make check` locally before pushing.
- Push to `main` only (after test): Docker build → `cecep31/echobackend:latest`, `:sha-<12-char>`, `:sha-<full>`. Multi-stage Go 1.26 image, non-root. Pull a pinned `sha-*` tag for reproducible deploys.

# Echo Backend API

REST API for [pilput](https://pilput.net), built with Go 1.27, Echo v5, GORM, and PostgreSQL. It uses manual dependency injection and Valkey/Redis caching.

## Core Features

- **Blog & community**: posts, comments, tags, likes, views, bookmarks (with folders), follows, notifications, and moderation reports.
- **Guilds**: community spaces with channels, messages, and realtime events over SSE, fanned out through Redis pub/sub.
- **AI chat**: conversations with streaming replies from [OpenRouter](https://openrouter.ai). Leave `OPENROUTER_API_KEY` empty to turn it off.
- **Portfolio tracking**: holdings, monthly summaries and trends, price sync, exchange rates, and an IDX corporate-action calendar (market data from RapidAPI).
- **Auth**: short-lived JWT access tokens plus rotating refresh tokens (sliding + absolute expiry), GitHub OAuth, password reset by email, activity logs, and admin session revocation.
- **Rate limiting**: an optional global token-bucket limiter plus per-route limits on sensitive auth endpoints. The limits are shared across instances when Redis is configured.
- **Storage**: S3-compatible object storage (RustFS locally, AWS S3 in production).
- **Validation**: request validation with `go-playground/validator/v10`.
- **UUID keys**: every model uses a UUID primary key (Postgres `uuidv7()` by default).
- **Operations**: a `/health` endpoint for Docker HEALTHCHECK and load balancers, and graceful shutdown that closes open SSE streams and releases resources.

## Tech Stack

| Layer | Choice |
|-------|--------|
| Runtime | Go 1.27 |
| Web framework | [Echo v5](https://github.com/labstack/echo) |
| ORM | [GORM](https://gorm.io/) v2 (`pgx/v5` driver) |
| Database | PostgreSQL 18+ |
| Cache / pub-sub | Valkey or Redis (optional; in-memory fallbacks where available) |
| Object storage | RustFS (local) / AWS S3 |
| Migrations | [Goose](https://github.com/pressly/goose) (raw SQL) |

## Quick Start

Prerequisites: Go 1.27, Docker, [goose](https://github.com/pressly/goose), and optionally [air](https://github.com/air-verse/air) for hot reload.

```bash
# 1. Create your environment file
cp .env.example .env

# 2. Edit .env:
#    - set JWT_SECRET (at least 32 characters)
#    - for the local services below: REDIS_URL=redis://localhost:6379 and S3_USE_SSL=false

# 3. Start local services: Postgres 18, Redis 8, RustFS (+ creates the bucket)
docker compose up -d --wait        # or: make up

# 4. Apply database migrations
goose up                           # or: make migrate-up

# 5. Run the server
air                                # hot reload (or: make dev)
go run cmd/main.go                 # or run it without hot reload (or: make run)
```

The server listens on `http://localhost:8080`. All API routes are served under `/api`. The RustFS console is at `http://localhost:9001`.

Run `make help` to see all Makefile targets. On Windows, run `make` through Git Bash or WSL, or run the commands below directly.

## Commands Reference

| Task | Command | Make |
|------|---------|------|
| Run server | `go run cmd/main.go` | `make run` |
| Hot reload | `air` | `make dev` |
| Build binary | `go build -o bin/main cmd/main.go` | `make build` |
| Run tests | `go test ./...` | `make test` |
| Race detector | `go test -race ./...` | `make test-race` |
| Coverage | `go test -coverprofile=coverage.out ./...` | `make cover` |
| Static analysis | `go vet ./...` | `make vet` |
| Format | `golangci-lint fmt ./...` | `make fmt` |
| Lint | `golangci-lint run ./...` | `make lint` |
| Security scan | `gosec ./...` | `make sec` |
| CI-equivalent check | vet + test + lint | `make check` |
| Start / stop services | `docker compose up -d --wait` / `docker compose down` | `make up` / `make down` |
| Wipe service data | `docker compose down -v` | `make down-clean` |

The tests don't need a running database.

## Environment Variables

Only two variables are required at startup:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL DSN. Put libpq options such as `connect_timeout` in the DSN |
| `JWT_SECRET` | JWT signing key, at least 32 characters |

Everything else is optional and has sensible defaults. Each variable is documented in [`.env.example`](.env.example), grouped as follows:

- **Auth**: `JWT_EXPIRY`, `REFRESH_TOKEN_*`, `GITHUB_*`, `FRONTEND_*`, `MAIN_DOMAIN`
- **Database pool**: `DB_POOL_*`
- **HTTP**: `HTTP_RATE_LIMIT_*`, `HTTP_TRUST_PROXY`, `HTTP_ALLOW_ORIGINS`
- **Storage**: `S3_*`
- **Cache**: `REDIS_URL` / `VALKEY_URL`, `CACHE_*`
- **AI chat**: `OPENROUTER_*`
- **Email**: `SMTP_*` (leave `SMTP_HOST` empty to skip sending)
- **Market data**: `RAPIDAPI_KEY`
- **Debug**: `APP_DEBUG`

Many variables also accept a legacy alias, such as `MINIO_*` for `S3_*`. The aliases are listed in `.env.example`.

## Architecture

The code is organised in layers, and the wiring is done by hand in a single DI container:

```
cmd/main.go             entrypoint: config, DI container, server, /health, graceful shutdown
config/                 env loading + validation
internal/
  di/                   dependency-injection container (container.go)
  routes/               route registration per module, under /api
  middleware/           auth (JWT, optional auth, admin), rate limiting, recovery/logging
  handler/              HTTP handlers; respond via pkg/response
  dto/                  request/response DTOs and model converters
  service/              business logic
  repository/           data access (GORM)
  model/                GORM entities
  apperror/             shared error sentinels
  platform/             infrastructure adapters:
                        cache (Redis + pub/sub), database, email (SMTP),
                        market (RapidAPI quotes/IDX), openrouter, realtime (SSE hub), storage (S3)
pkg/                    reusable helpers: applog, password, response, slug, uid, validator
migrations/             goose SQL migrations
docs/api/               OpenAPI 3.1 spec
```

### Standardized Responses

All handlers use the `pkg/response` helpers, so every endpoint returns responses in the same shape:

```go
return response.Success(c, "Data retrieved", data)
return response.ValidationError(c, "Invalid input", err)
```

## API Documentation

The full HTTP API reference is [`docs/api/openapi.yaml`](docs/api/openapi.yaml), an OpenAPI 3.1 document split into one set of files per module under `docs/api/paths/` and `docs/api/schemas/`. See [`docs/api/README.md`](docs/api/README.md) for how to preview it, lint it, and generate clients from it.

## Database Migrations

Migrations are managed with [goose](https://github.com/pressly/goose), which reads its settings (`GOOSE_*`) from `.env`. The goose version table is `custom.goose_migrations`. The local Postgres container creates the `custom` schema through `scripts/init-db.sql`. On any other database, create that schema yourself before running goose.

```bash
goose up                      # apply pending migrations   (make migrate-up)
goose down                    # roll back the last one     (make migrate-down)
goose status                  # show migration status      (make migrate-status)
goose create <name> sql       # new SQL migration          (make migrate-create name=<name>)
```

## CI & Deployment

GitHub Actions ([`.github/workflows/main.yml`](.github/workflows/main.yml)) runs on every PR and every push to `main`. Changes that only touch docs are skipped.

1. **Test**: `go mod verify`, `go mod tidy -diff`, `go vet`, build, and `go test -race` with a coverage summary.
2. **Lint**: `golangci-lint`.
3. **Docker**: the image is built on every run. It is pushed to Docker Hub (`cecep31/echobackend`) only on `main`, after test and lint pass, with the tags `latest`, `sha-<12-char>`, and `sha-<full-sha>`.

For reproducible deploys, pin a `sha-*` tag instead of `latest`.

```bash
docker build -t cecep31/echobackend .
docker run -p 8080:8080 --env-file .env cecep31/echobackend
```

## License

MIT

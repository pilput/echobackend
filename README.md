# Echo Backend API

REST API for [pilput](https://pilput.net), built with Go 1.27, Echo v5, GORM, and PostgreSQL 18.

## Features

- **Blog & community**: posts, comments, tags, likes, bookmarks, follows, notifications, reports.
- **Guilds**: Discord-style channels and messages with realtime updates over SSE.
- **AI chat**: streaming replies from [OpenRouter](https://openrouter.ai).
- **Portfolio**: holdings, monthly summaries, price sync, exchange rates, IDX corporate-action calendar.
- **Auth**: JWT with rotating refresh tokens, GitHub OAuth, password reset by email.

Redis, S3, SMTP, OpenRouter, and RapidAPI are optional. The app still runs when they are not configured.

## Quick Start

Prerequisites: Go 1.27, Docker, [goose](https://github.com/pressly/goose), and optionally [air](https://github.com/air-verse/air).

```bash
cp .env.example .env
# Edit .env: set JWT_SECRET (min. 32 chars), REDIS_URL=redis://localhost:6379, S3_USE_SSL=false

docker compose up -d --wait   # Postgres, Redis, RustFS   (make up)
goose up                      # apply migrations          (make migrate-up)
go run cmd/main.go            # or: air for hot reload    (make run / make dev)
```

The server runs on `http://localhost:8080`, with all routes under `/api`. Run `make help` to see the other commands.

## Development

```bash
make check    # vet + test + lint, same as CI
make test     # go test ./... (no database needed)
make fmt      # golangci-lint fmt ./...
```

Only `DATABASE_URL` and `JWT_SECRET` are required. Every other variable is documented in [`.env.example`](.env.example).

## Project Structure

```
cmd/main.go        entrypoint
config/            env loading
internal/          handler → service → repository, plus model, dto, routes, middleware, di, platform
pkg/               shared helpers (response, validator, password, ...)
migrations/        goose SQL migrations
docs/api/          OpenAPI 3.1 spec
```

## API Docs

The API reference is [`docs/api/openapi.yaml`](docs/api/openapi.yaml). See [`docs/api/README.md`](docs/api/README.md) for how to preview and lint it.

## Deployment

Every push to `main` that passes CI publishes the Docker image `cecep31/echobackend` with the tags `latest` and `sha-<commit>`. Use a `sha-*` tag for reproducible deploys.

```bash
docker run -p 8080:8080 --env-file .env cecep31/echobackend
```

## License

MIT

# API Documentation - echobackend

HTTP API reference for frontend integration. All business routes are under the `/api` prefix, except the root health check endpoints.

## Base URL

| Environment | URL |
|-------------|-----|
| Local | `http://localhost:<PORT>` (see `PORT` in `.env`) |
| Production | Your deployment / reverse proxy URL |

Full examples: `GET /api/posts`, `POST /api/auth/login`.

## Authentication

Routes that require login must send this header:

```http
Authorization: Bearer <access_token>
```

Tokens are returned by `POST /api/auth/login`, `POST /api/auth/refresh`, or `POST /api/auth/oauth/exchange` after GitHub OAuth. JWT claims include `user_id` (UUID).

Failed auth middleware responses (missing token, invalid token, or non-super-admin user on admin routes) use the standard `success` envelope below, with `success: false` and a generic `error` string:

| Situation | HTTP | Body |
|-----------|------|------|
| Missing / invalid token | 401 | `{"success":false,"message":"...","error":"Unauthorized access"}` |
| Not a super admin on an admin route | 403 | `{"success":false,"message":"...","error":"Access forbidden"}` |

## Standard Response Format

Most handlers use the `pkg/response` envelope:

```json
{
  "success": true,
  "message": "Human-readable message",
  "data": {},
  "meta": {},
  "error": "...",
  "errors": []
}
```

`data`, `meta`, `error`, and `errors` use `omitempty` — they are omitted from the JSON when empty (for example, success responses have no `error`/`errors` keys, and error responses usually have no `data`/`meta` keys).

| Helper | HTTP | Notes |
|--------|------|-------|
| Success | 200 | `success: true`, optional `data` |
| Created | 201 | Same envelope as success |
| Bad request | 400 | `success: false`, `error` contains details |
| Unauthorized | 401 | `error`: `"Unauthorized access"` |
| Forbidden | 403 | `error`: `"Access forbidden"` |
| Not found | 404 | |
| Conflict | 409 | Duplicate resource |
| Validation | 422 | `errors` contains field errors (`field`, `message`, `value`, `tag`) |
| Server error | 500 | Generic client message; details are logged server-side only |

### Pagination (`meta`)

Paginated lists use `SuccessWithMeta`:

```json
{
  "meta": {
    "total_items": 100,
    "offset": 0,
    "limit": 10,
    "total_pages": 10
  }
}
```

Query: `limit` (default varies by endpoint, **maximum 100** on most endpoints — `GET /api/posts` is an exception and currently accepts larger values), `offset` (default `0`).

## Global Limits

- Request body size: **10 MB** (larger requests return **413**).
- CORS: `HTTP_ALLOW_ORIGINS` (default `*`).
- Global rate limit: enabled when `HTTP_RATE_LIMIT_RPS` > 0.
- Auth-specific rate limits use a fixed window per IP. If `VALKEY_URL` is set, counters are stored in Valkey/Redis and work across instances; otherwise they fall back to in-memory per instance:
  `register`, `login`, and `reset-password` **5 / 5 minutes**;
  `forgot-password` **3 / 5 minutes**;
  `refresh` **30 / minute**;
  `oauth/exchange` **10 / minute**;
  `check-username` **20 / 5 minutes**.

## Health & Root

| Method | Path | Auth | Response |
|--------|------|------|----------|
| GET | `/` | No | Success envelope with welcome message |
| GET | `/health` | No | `200` `{"status":"ok"}` or `503` `{"status":"unhealthy","reason":"database unreachable"}` |

## Modules

| Module | Base path | Document |
|--------|-----------|----------|
| Auth | `/api/auth` | [auth.md](./auth.md) |
| Users & follow | `/api/users` | [users.md](./users.md) |
| Posts (comments, views, likes) | `/api/posts` | [posts.md](./posts.md) |
| Tags | `/api/tags` | [tags.md](./tags.md) |
| Chat | `/api/chat/conversations`, `/api/chat/messages` | [chat.md](./chat.md) |
| Holdings | `/api/holdings`, `/api/holding-types` | [holdings.md](./holdings.md) |
| Exchange rates | `/api/exchange-rates` | [exchange-rates.md](./exchange-rates.md) |
| Bookmarks | `/api/bookmarks` | [bookmarks.md](./bookmarks.md) |
| Notifications | `/api/notifications` | [notifications.md](./notifications.md) |
| Reports (admin) | `/api/reports` | [reports.md](./reports.md) |

Debug routes (`/api/debug/pprof/*`) are registered only when `APP_DEBUG=true`; they are not intended for frontend use.

## Type Conventions

- **UUID**: string primary key for users, posts, comments, and conversations.
- **Time**: ISO 8601 / RFC3339 (`2026-05-12T08:00:00Z`).
- **Nullable**: Go pointer fields may be `null` or omitted (`omitempty`).
- **Financial numbers (holdings)**: decimal strings in JSON (for example `"1500000.00"`), not numbers.

# Auth Module - `/api/auth`

Registration, login, OAuth, refresh tokens, password reset, password change, logout, profile, and activity logs.

## Endpoint Summary

Public auth endpoints use a fixed-window rate limit per IP. If `VALKEY_URL` is enabled, counters are stored in Valkey/Redis and work across instances; otherwise the backend falls back to in-memory counters per instance.

| Method | Path | Auth | Rate limit |
|--------|------|------|------------|
| POST | `/register` | No | 5 / 5 minutes |
| POST | `/login` | No | 5 / 5 minutes |
| POST | `/forgot-password` | No | 3 / 5 minutes |
| POST | `/reset-password` | No | 5 / 5 minutes |
| POST | `/refresh` | No | 30 / minute |
| POST | `/logout` | Bearer | Global |
| GET | `/profile` | Bearer | Global |
| PATCH | `/password` | Bearer | Global |
| GET | `/activity-logs` | Bearer | Global |
| GET | `/activity-logs/recent` | Bearer | Global |
| GET | `/activity-logs/failed-logins` | Bearer + super admin | Global |
| GET | `/oauth/github` | No | Global |
| GET | `/oauth/github/callback` | No | Global |
| POST | `/oauth/exchange` | No | 10 / minute |

---

## POST `/api/auth/register`

Create a new account.

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `email` | string | Yes | email format |
| `username` | string | Yes | 3-30 characters |
| `password` | string | Yes | min 8 characters |

> **Note:** Recommended password strength is at least 8 characters with uppercase, lowercase, number, and special character.

**Success - 201**

```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe"
  }
}
```

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid body |
| 422 | Validation failed |
| 409 | Email or username already in use |
| 500 | Server error |

---

## POST `/api/auth/login`

Login with email **or** username in the `identifier` field.

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `identifier` | string | Yes | - |
| `password` | string | Yes | min 6 |

**Success - 200**

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "pl_...",
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "johndoe"
    }
  }
}
```

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid body |
| 401 | Wrong credentials |
| 429 | Rate limited |
| 500 | Server error |

---

## POST `/api/auth/forgot-password`

Request a password reset. The response is the **same** whether the email is registered or not (anti-enumeration).

If `SMTP_HOST` is set and the Asynq queue is available through `QUEUE_REDIS_URL` or `VALKEY_URL`, the backend enqueues the password reset email and a worker sends it through SMTP. The reset link is built from `FRONTEND_RESET_PASSWORD_URL` with a `token` query parameter. If SMTP/queue is not configured, the token is still created and activity is recorded in dev mode metadata.

**Related env vars**

| Variable | Description |
|----------|-------------|
| `SMTP_HOST` | SMTP host. Empty means email delivery is disabled |
| `SMTP_PORT` | SMTP port, default `587` |
| `SMTP_USERNAME` | SMTP username |
| `SMTP_PASSWORD` | SMTP password |
| `SMTP_FROM` | Sender email, default `noreply@pilput.net` |
| `SMTP_TLS` | `true` for implicit TLS (commonly port 465), `false` for STARTTLS (commonly port 587) |
| `SMTP_TIMEOUT_SECONDS` | SMTP connection timeout, default `10` |
| `QUEUE_REDIS_URL` | Redis/Valkey URL for the Asynq broker. Falls back to `VALKEY_URL` when empty |
| `QUEUE_REDIS_TIMEOUT_MS` | Redis connection/read/write timeout for Asynq, default `5000` |
| `QUEUE_DEFAULT_NAME` | Default queue name for all background jobs, default `default` |
| `QUEUE_CONCURRENCY` | Number of Asynq workers for all jobs, default `1` |
| `QUEUE_MAX_RETRY` | Maximum task retry count, default `5` |
| `FRONTEND_RESET_PASSWORD_URL` | Frontend reset password page URL |

**Body**

| Field | Type | Required |
|-------|------|----------|
| `email` | string | Yes (email) |

**Success - 200**

```json
{
  "success": true,
  "message": "If the email exists, a password reset link has been sent",
  "data": null
}
```

---

## POST `/api/auth/reset-password`

Set a new password using a reset token.

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `token` | string | Yes | - |
| `password` | string | Yes | min 8 |

**Success - 200**

```json
{
  "success": true,
  "message": "Password reset successful",
  "data": null
}
```

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid / expired / already-used token |

---

## POST `/api/auth/refresh`

Extend a session with a refresh token. Returns a new access token; the refresh token is **not** rotated — the same one is echoed back and keeps working until it expires (its lifetime is not extended by calling this endpoint). Once it expires, the user must log in again.

**Body**

| Field | Type | Required |
|-------|------|----------|
| `refresh_token` | string | Yes |

**Success - 200** - `data` has the same shape as login (`access_token`, `refresh_token`, `user`). `refresh_token` is the same value passed in the request.

**Errors**

| HTTP | Condition |
|------|-----------|
| 401 | Invalid / expired refresh token |

---

## POST `/api/auth/logout`

Log out the user by deleting the refresh token session.

**Header:** `Authorization: Bearer <access_token>`

**Body**

| Field | Type | Required |
|-------|------|----------|
| `refresh_token` | string | Yes |

**Success - 200**

```json
{
  "success": true,
  "message": "Logout successful",
  "data": null
}
```

---

## GET `/api/auth/profile`

Get the currently logged-in user's profile. Returns a flat subset of user fields (no `name` or nested `profile` object). For the full current-user payload including `profile`, use `GET /api/users/me`.

**Header:** `Authorization: Bearer <access_token>`

**Success - 200**

```json
{
  "success": true,
  "message": "Profile retrieved successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "image": "https://...",
    "is_super_admin": false,
    "followers_count": 0,
    "following_count": 0
  }
}
```

---

## PATCH `/api/auth/password`

Change the currently logged-in user's password.

**Header:** `Authorization: Bearer <access_token>`

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `current_password` | string | Yes | min 8 |
| `new_password` | string | Yes | min 8 |

**Success - 200** - `data`: `null`.

**Errors**

| HTTP | Condition |
|------|-----------|
| 401 | Not authenticated or old password is wrong |

---

## GET `/api/auth/activity-logs`

Paginated auth activity logs for the currently logged-in user.

**Header:** `Authorization: Bearer <access_token>`

**Query Parameters**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | int | 20 | Max 100 |
| `offset` | int | 0 | - |
| `activity_type` | string | - | Filter: `login`, `login_failed`, `logout`, `register`, `password_change`, `password_reset_request`, `password_reset`, `token_refresh`, `oauth_login`, `oauth_login_failed` |

**Success - 200**

```json
{
  "success": true,
  "message": "Activity logs retrieved successfully",
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "activity_type": "login",
      "ip_address": "127.0.0.1",
      "user_agent": "Mozilla/5.0...",
      "status": "success",
      "error_message": null,
      "metadata": null,
      "created_at": "2026-05-13T07:00:00Z"
    }
  ],
  "meta": {
    "total_items": 50,
    "offset": 0,
    "limit": 20,
    "total_pages": 3
  }
}
```

---

## GET `/api/auth/activity-logs/recent`

Recent activity logs for the currently logged-in user.

**Header:** `Authorization: Bearer <access_token>`

**Query Parameters**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | int | 10 | Max 50 |

**Success - 200**

```json
{
  "success": true,
  "message": "Recent activity retrieved successfully",
  "data": [...]
}
```

---

## GET `/api/auth/activity-logs/failed-logins`

Failed login list (all users, for admin monitoring).

**Header:** `Authorization: Bearer <access_token>`

**Access:** super admin only.

**Query Parameters**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | int | 20 | Max 100 |
| `offset` | int | 0 | - |
| `since_hours` | int | 24 | Last N hours |

**Success - 200**

```json
{
  "success": true,
  "message": "Failed logins retrieved successfully",
  "data": [...],
  "meta": { "total_items": 5, "offset": 0, "limit": 20, "total_pages": 1 }
}
```

---

## GET `/api/auth/oauth/github`

Redirect to GitHub's authorization page. The browser is redirected to:

```text
https://github.com/login/oauth/authorize?client_id=...&redirect_uri=...&scope=user:email&state=...
```

The backend creates an HttpOnly `github_oauth_state` cookie with a 10-minute TTL for CSRF validation on callback.

**Success - 307** Redirect to GitHub.

---

## GET `/api/auth/oauth/github/callback`

Callback from GitHub after user authorization. Exchanges `code` for a GitHub access token, fetches the GitHub profile, then creates/finds the local user.

If GitHub email is unavailable, the endpoint requests email from `https://api.github.com/user/emails`.

**Query Parameters**

| Param | Type | Description |
|-------|------|-------------|
| `code` | string | Authorization code from GitHub |
| `state` | string | CSRF state that must match the `github_oauth_state` cookie |

**Success flow - 307** Redirect to:

```text
{FRONTEND_OAUTH_CALLBACK_URL}?code=oc_...
```

Default: `{FRONTEND_URL}/auth/callback` (`http://localhost:3000/auth/callback`).

`code` is a one-time exchange code with a 2-minute TTL. The frontend must exchange it with `POST /api/auth/oauth/exchange` to get `access_token` and `refresh_token`. If Valkey/Redis is enabled, the code is stored in cache with atomic get-delete; if cache is disabled, the backend uses an in-memory fallback for local/dev.

**Failure flow - 307** Redirect to:

```text
{FRONTEND_OAUTH_CALLBACK_URL}?error=<error_type>
```

| Error type | Condition |
|------------|-----------|
| `missing_code` | Empty `code` parameter |
| `invalid_state` | Missing `state` parameter or cookie mismatch |
| `github_token_failed` | Failed to exchange code with GitHub token |
| `github_user_failed` | Failed to fetch GitHub profile |
| `oauth_login_failed` | Failed to create/login user |
| `oauth_exchange_failed` | Failed to create one-time exchange code |

---

## POST `/api/auth/oauth/exchange`

Exchange the one-time code from the OAuth callback for application tokens. The code can only be used once and expires in 2 minutes.

In production, enable `VALKEY_URL` so exchange codes work across backend instances. Without Valkey/Redis, the in-memory fallback is safe only for single-instance/local dev.

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `code` | string | Yes | One-time code from callback query |

**Example request**

```json
{
  "code": "oc_..."
}
```

**Success - 200**

```json
{
  "success": true,
  "message": "OAuth code exchanged successfully",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "pl_...",
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "johndoe"
    }
  }
}
```

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid body |
| 401 | Invalid, expired, or already-used code |
| 422 | Validation failed |
| 500 | Server error |

---

## Auth Activity Log

All auth operations are recorded in `auth_activity_logs` with these types:

| Type | Description |
|------|-------------|
| `login` | Successful login |
| `login_failed` | Failed login |
| `logout` | Logout |
| `register` | Successful registration |
| `password_change` | Password change |
| `password_reset_request` | Password reset request |
| `password_reset` | Successful password reset |
| `token_refresh` | Token refresh |
| `oauth_login` | OAuth login (GitHub) |
| `oauth_login_failed` | Failed OAuth login |

Each log stores: `user_id`, `activity_type`, `ip_address`, `user_agent`, `status` (success/failure/pending), `error_message`, `metadata` (JSON), `created_at`.

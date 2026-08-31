# Users & Follow Module - `/api/users`

User profiles, admin user lists, follow/unfollow, and social statistics.

## Data Types

### `UserResponse` (admin routes: `GET /`, `GET /:id`)

| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | |
| `email` | string \| null | Omitted on public/general routes; present only on admin routes or `/me` |
| `name` | string | Combined first + last name |
| `username` | string \| null | |
| `image` | string \| null | Avatar URL |
| `first_name` | string \| null | |
| `last_name` | string \| null | |
| `followers_count` | number | |
| `following_count` | number | |
| `is_following` | boolean \| null | Present only on routes with auth context, for example admin `GET /:id` |
| `is_super_admin` | boolean \| null | Present only on admin routes (`GET /`, `GET /:id`) |
| `profile` | object \| null | Not loaded on `GET /` (admin list); available on other routes |
| `created_at` | string (ISO) \| null | |
| `updated_at` | string (ISO) \| null | |
| `deleted_at` | string (ISO) \| null | Admin routes only; set when the user has been soft-deleted |
| `last_logged_at` | string (ISO) \| null | Admin routes only (`GET /`, `GET /:id`); last recorded login timestamp, omitted when never set |

### `CurrentUserResponse` (`GET /me`)

Same fields as `UserResponse`, except:

- Always includes `is_super_admin` (not omitted).
- Always includes `email`.
- Never includes `deleted_at`, `is_following`, or `last_logged_at`.
- Includes `profile` when loaded.

### `PublicUserResponse` (`GET /username/:username`, follow lists)

Public profile shape of `UserResponse`. Omits `email`, `is_super_admin`, `deleted_at`, and `last_logged_at`. `is_following` is always omitted on `GET /username/:username` because the route has no auth middleware.

### `Profile`

| Field | Type |
|-------|------|
| `id` | number |
| `user_id` | string (UUID) |
| `bio` | string \| null |
| `website` | string \| null |
| `phone` | string \| null |
| `location` | string \| null |
| `created_at` | string \| null |
| `updated_at` | string \| null |

---

## Profiles & Admin

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/:id` | Bearer + **super admin** | By UUID |
| GET | `/username/:username` | No | By username |
| GET | `/me` | Bearer | User from token |
| POST | `/me/image` | Bearer | Upload/replace the current user's avatar |
| GET | `` | Bearer + **super admin** | User list (paginated); soft-delete filter via query |
| POST | `` | Bearer + **super admin** | Create a user |
| PUT | `/:id` | Bearer + **super admin** | Update any user's profile fields |
| DELETE | `/:id` | Bearer + **super admin** | Soft-delete user |
| POST | `/:id/restore` | Bearer + **super admin** | Restore a soft-deleted user |

### GET `/api/users/me`

**Success - 200** - `data`: one `CurrentUserResponse`.

### POST `/api/users/me/image`

Upload or replace the current user's avatar. `multipart/form-data` with a single file field named `avatar`.

**Header:** `Authorization: Bearer <access_token>`

**Form field**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `avatar` | file | Yes | JPEG, PNG, GIF, or WebP; max 5 MB (checked by content sniffing, not just extension) |

**Success - 200**

```json
{
  "success": true,
  "message": "Profile picture updated successfully",
  "data": { "image": "users/avatars/<generated-key>.jpg" }
}
```

`image` is the stored object key, in the same format as the `image` field returned elsewhere (`GET /api/auth/profile`, `UserResponse`, etc.) — resolve it the same way the frontend already resolves other user/post image fields.

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | No file / file missing, file over 5 MB, unsupported file type, or storage unavailable |
| 401 | Not authenticated |
| 404 | User not found |
| 500 | Server error |

### GET `/api/users` (admin)

**Query**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | number | 10 | Max 100 |
| `offset` | number | 0 | |
| `deleted` | string | *(empty)* | Soft-delete filter: `false` or empty = active users only; `true` = deleted users only; `all` = all users |

Any `deleted` value other than `true`, `false`, `all`, or empty returns **400 Bad Request**.

**Success - 200** - `data`: `UserResponse[]`, `meta`: pagination.

Examples:

```http
GET /api/users?deleted=true&limit=10&offset=0
GET /api/users?deleted=all
```

### POST `/api/users` (admin)

Create a user account directly (admin action; skips email verification/registration flow).

**Header:** `Authorization: Bearer <access_token>`

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `username` | string | Yes | 3-30 characters |
| `email` | string | Yes | email format |
| `password` | string | Yes | min 8 characters |
| `first_name` | string | No | max 100 characters |
| `last_name` | string | No | max 100 characters |

**Success - 201** - `data`: `UserResponse`.

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid body |
| 401 | Not authenticated |
| 403 | Not a super admin |
| 409 | Email or username already exists |
| 422 | Validation failed |
| 500 | Server error |

### GET `/api/users/:id` (admin)

**Query**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `deleted` | string | *(empty)* | `true` = fetch a soft-deleted user by UUID |

Without `deleted=true`, only **active** users are returned. Deleted users are hidden unless the query above is used.

**Success - 200** - `data`: `UserResponse`. `is_following` is set when the requester is logged in (the admin route already uses auth). `deleted_at` is set when the user has been soft-deleted.

Example:

```http
GET /api/users/<uuid>?deleted=true
```

### PUT `/api/users/:id` (admin)

Update any user's profile fields, including their `is_super_admin` flag. Every field is sent on every request (no partial update semantics) — omitted/empty `username` or `email` fail validation.

**Header:** `Authorization: Bearer <access_token>`

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `username` | string | Yes | 3-30 characters |
| `email` | string | Yes | email format |
| `first_name` | string | No | max 100 characters |
| `last_name` | string | No | max 100 characters |
| `is_super_admin` | boolean | No | defaults to `false` if omitted |

**Success - 200** - `data`: updated `UserResponse`.

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid body |
| 401 | Not authenticated |
| 403 | Not a super admin |
| 404 | User not found |
| 409 | Email or username already taken by another user |
| 422 | Validation failed |
| 500 | Server error |

### GET `/api/users/username/:username`

**Success - 200** - `data`: `UserResponse` (public profile shape).

### DELETE `/api/users/:id` (admin)

Soft-delete a user (sets `deleted_at`; the row is not permanently removed from the database).

**Success - 200** - `data`: `null`.

### POST `/api/users/:id/restore` (admin)

Restore a soft-deleted user (`deleted_at` -> `null`).

**Success - 200** - `data`: reactivated `UserResponse`.

**Errors**

| Status | Condition |
|--------|-----------|
| 404 | User not found or not soft-deleted |
| 409 | Email or username is already used by another active user |

Example:

```http
POST /api/users/<uuid>/restore
```

---

## Follow

| Method | Path | Auth |
|--------|------|------|
| POST | `/follow` | Bearer |
| DELETE | `/:id/follow` | Bearer |
| GET | `/:id/follow-status` | Bearer |
| GET | `/:id/mutual-follows` | Bearer |
| GET | `/:id/followers` | No |
| GET | `/:id/following` | No |
| GET | `/:id/follow-stats` | No |

### POST `/api/users/follow`

**Body**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `user_id` | string | Yes | UUID |

**Success - 200** - `data`:

```json
{
  "is_following": true,
  "message": "Message from service"
}
```

### DELETE `/api/users/:id/follow`

Unfollow the user with the UUID in the path.

**Success - 200** - `data`: `FollowResponse` (same shape as follow).

### GET `/api/users/:id/follow-status`

**Success - 200**

```json
{
  "data": { "is_following": false }
}
```

### GET `/api/users/:id/mutual-follows`

**Success - 200** - `data`: `UserResponse[]`.

### GET `/api/users/:id/followers` and `/:id/following`

**Query:** `limit`, `offset`.

**Success - 200** - `data`: `UserResponse[]`, `meta`: pagination.

### GET `/api/users/:id/follow-stats`

**Success - 200** - `data`:

```json
{
  "user_id": "uuid",
  "followers_count": 10,
  "following_count": 5
}
```

**Note:** Some follow domain errors (user not found, already following, etc.) can currently return **500** from the handler layer instead of a specific 4xx.

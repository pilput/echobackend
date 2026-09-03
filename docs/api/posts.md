# Posts Module - `/api/posts`

Post CRUD, image uploads, feeds, comments, views, and likes. Sub-resources use `:id` as the post UUID.

## Data Type: `PostResponse`

| Field | Type | Description |
|-------|------|-------------|
| `id` | string (UUID) | |
| `title` | string \| null | |
| `photo_url` | string \| null | |
| `body` | string \| null | Feed lists truncate this to about 250 runes + `" ..."` |
| `slug` | string \| null | |
| `view_count` | number | |
| `like_count` | number | |
| `bookmark_count` | number | |
| `published` | boolean \| null | |
| `published_at` | string \| null | |
| `user` | `UserBrief` \| null | Author (see below) |
| `tags` | `TagResponse[]` | `{ id, name }` |
| `created_at` | string \| null | |
| `updated_at` | string \| null | |
| `deleted_at` | string \| null | Soft delete timestamp |

### `TagResponse`

`{ "id": number, "name": string }`

### `UserBrief` (nested author on posts, comments, and likes)

| Field | Type |
|-------|------|
| `id` | string (UUID) |
| `username` | string \| null |
| `image` | string \| null |

---

## CRUD & Listing

| Method | Path | Auth |
|--------|------|------|
| POST | `` | Bearer |
| GET | `` | No |
| GET | `/random` | No |
| GET | `/trending` | No |
| GET | `/me` | Bearer |
| GET | `/me/:id` | Bearer |
| PUT | `/me/:id` | Bearer |
| DELETE | `/me/:id` | Bearer |
| GET | `/me/analytics` | Bearer |
| GET | `/me/analytics/likes-by-month` | Bearer |
| GET | `/feed/for-you` | Bearer |
| POST | `/image` | Bearer |
| GET | `/sitemap` | No |
| GET | `/username/:username` | No |
| GET | `/u/:username/:slug` | No |
| GET | `/tag/:tag` | No |
| GET | `/:id` | Bearer + **super admin** |
| PUT | `/:id` | Bearer + **super admin** |
| DELETE | `/:id` | Bearer + **super admin** |

### POST `/api/posts`

**Body (`CreatePostRequest`)**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `title` | string | Yes | min 7 |
| `slug` | string | Yes | min 7 |
| `body` | string | Yes | min 10 |
| `photo_url` | string | No | |
| `published` | boolean | No | default false |
| `tags` | string[] | No | Tag names |

**Success - 201** - `data`: `{ "id": "uuid" }`.

### GET `/api/posts`

**Query**

| Param | Description |
|-------|-------------|
| `search` | Search text |
| `sort_by` | `id`, `title`, `created_at`, `updated_at`, `view_count`, `like_count` |
| `sort_order` | `asc` / `desc` (default `desc`) |
| `start_date`, `end_date` | Date filters |
| `created_by` | Author UUID |
| `published` | `true` / `false` |
| `tags` | Comma-separated list |
| `limit`, `offset` | Pagination (default limit 10) |

**Success - 200** - `data`: `PostResponse[]`, `meta`: pagination.

### GET `/api/posts/random`

**Query:** `limit` (default 9, max 20).

### GET `/api/posts/trending`

**Query:** `limit` (default 10, max 100).

### GET `/api/posts/username/:username` and `/api/posts/tag/:tag`

Paginated published posts for a user or tag name.

**Query:** `limit`, `offset` (default limit 10, max 100).

### GET `/api/posts/me`

Paginated list of posts owned by the logged-in user, including unpublished drafts. **Auth required.**

**Query**

| Param | Description |
|-------|-------------|
| `limit`, `offset` | Pagination (default limit 10) |
| `search` | Case-insensitive match against `posts.title` (`ILIKE '%search%'`); omit for no filter |
| `published` | `true` / `false`; omit (or send anything else) to include both published and draft posts |

### GET `/api/posts/feed/for-you`

**Query:** `limit`, `offset`. **Auth required.**

### GET / PUT / DELETE `/api/posts/me/:id`

Read, update, or delete a post owned by the logged-in user. **Auth required.**

**PUT success - 200** - `data`: full `PostResponse`.

**Common errors**

| HTTP | Condition |
|------|-----------|
| 400 | Invalid post ID |
| 403 | Not the author |
| 404 | Post not found |

### GET `/api/posts/me/analytics`

Aggregated chart data for posts owned by the logged-in user. **Auth required.**

**Query (optional)**

| Param | Default | Description |
|-------|---------|-------------|
| `start_date` | 30 days ago | Format `YYYY-MM-DD` |
| `end_date` | Today | Format `YYYY-MM-DD` |

**Success - 200** - `data`:

```json
{
  "summary": {
    "total_posts": 12,
    "published_posts": 10,
    "total_views": 1500,
    "total_likes": 230
  },
  "view_trend": [
    { "date": "2026-04-24", "views": 10, "cumulative_views": 100 }
  ],
  "top_posts": [
    {
      "id": "uuid",
      "title": "Post title",
      "slug": "post-title",
      "view_count": 500,
      "like_count": 80
    }
  ]
}
```

- `view_trend`: daily series for line/area charts (one point per day in the range, including days with zero views).
- `top_posts`: up to 5 posts with the highest `view_count` (bar chart / ranking).

### GET `/api/posts/me/analytics/likes-by-month`

Likes **received** by posts owned by the logged-in user, aggregated monthly. **Auth required.**

**Query (optional)**

| Param | Default | Description |
|-------|---------|-------------|
| `months` | 12 | 1-24; last N calendar months, including the current month |

**Success - 200** - `data`:

```json
{
  "months": 12,
  "series": [
    { "month": "2025-06", "likes": 0 },
    { "month": "2025-07", "likes": 14 },
    { "month": "2026-05", "likes": 23 }
  ],
  "total": 230
}
```

- `series`: one point per month (`YYYY-MM`), zero-filled for months without likes.
- `total`: total likes within the `months` range.

### POST `/api/posts/image`

**Content-Type:** `multipart/form-data`

| Field | Required | Limit |
|-------|----------|-------|
| `image` | Yes (file) | Max **1 MiB** / 1,048,576 bytes |

**Success - 200** - `data`: `null` (upload succeeds with a success message only).

**Error:** 400 when the file is empty, exceeds 1 MiB, or storage is unavailable.

### GET `/api/posts/sitemap`

Returns posts formatted for sitemap generation.

**Query (optional)**

| Param | Default | Description |
|-------|---------|-------------|
| `limit` | 50000 | Maximum number of posts to return (max: 50000) |

**Success - 200** - `data`: `SitemapPost[]`:

```json
{ "username": "...", "slug": "...", "created_at": "...", "updated_at": "..." }
```

### GET `/api/posts/u/:username/:slug`

Full detail for one post (body is not truncated).

### GET `/api/posts/:id`

Full detail for one post. **Super admin auth required.**

### PUT `/api/posts/:id`

Update a post by ID. **Super admin auth required.**

**Body (`UpdatePostRequest`)** - all fields are optional; `published` is a boolean pointer.

**Common errors**

| HTTP | Condition |
|------|-----------|
| 404 | Post not found |

### DELETE `/api/posts/:id`

Delete a post by ID. **Super admin auth required.**

**Success - 200** - `data`: `null`.

---

## Comments - `/api/posts/:id/comments`

| Method | Path | Auth |
|--------|------|------|
| GET | `/:id/comments` | No |
| POST | `/:id/comments` | Bearer |
| PUT | `/:id/comments/:comment_id` | Bearer (owner only) |
| DELETE | `/:id/comments/:comment_id` | Bearer (owner, or **super admin**) |

### `CommentResponse`

| Field | Type |
|-------|------|
| `id` | string (UUID) |
| `post_id` | string |
| `parent_comment_id` | string \| null | Read-only in responses; not accepted on create |
| `text` | string |
| `user` | `UserBrief` \| null |
| `created_at` | string \| null |
| `updated_at` | string \| null |

### POST / PUT body

POST and PUT accept the same body:

| Field | Required | Validation |
|-------|----------|------------|
| `text` | Yes | 1-1000 characters |

**Success - 201 (POST) / 200 (PUT)** - `data`: `CommentResponse`.

### DELETE `/api/posts/:id/comments/:comment_id`

Deletes a comment. Requires the requester to own the comment (`created_by`), **or** be a super admin (moderation) — a super admin can delete any user's comment, not just their own. `PUT` (edit) remains owner-only; there is no admin override for editing.

**Success - 200** - `data`: `null`.

---

## Views - `/api/posts/:id/view*`

| Method | Path | Auth |
|--------|------|------|
| POST | `/:id/view` | Bearer |
| GET | `/:id/views` | Bearer |
| GET | `/:id/view-stats` | No |
| GET | `/:id/viewed` | Bearer |

### POST `/api/posts/:id/view`

Record a view for the logged-in user.

### GET `/api/posts/:id/view-stats`

**Success - 200** - `data`:

```json
{
  "post_id": "uuid",
  "total_views": 0,
  "unique_views": 0,
  "anonymous_views": 0,
  "authenticated_views": 0
}
```

### GET `/api/posts/:id/viewed`

**Success - 200** - `data`: `{ "has_viewed": true }`.

### GET `/api/posts/:id/views`

List views (internal model: `id`, `post_id`, `user_id`, `ip_address`, `user_agent`, timestamps) + pagination `meta`.

---

## Likes - `/api/posts/:id/like*`

| Method | Path | Auth |
|--------|------|------|
| POST | `/:id/like` | Bearer |
| DELETE | `/:id/like` | Bearer |
| GET | `/:id/likes` | No |
| GET | `/:id/like-stats` | No |
| GET | `/:id/liked` | Bearer |

### POST / DELETE like

**Error 400** when already liked / not yet liked.

### GET `/api/posts/:id/likes`

**Success - 200** - `data`:

```json
{
  "likes": [ /* PostLikeResponse */ ],
  "total": 0,
  "limit": 10,
  "offset": 0
}
```

`PostLikeResponse`: `id`, `post_id`, `user_id`, `user`, `created_at`.

### GET `/api/posts/:id/like-stats`

`data`: `{ "post_id", "total_likes" }`.

### GET `/api/posts/:id/liked`

`data`: `{ "has_liked", "post_id", "user_id" }`.

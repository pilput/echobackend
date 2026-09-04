# Database Migrations

Managed via [goose](https://github.com/pressly/goose). Env vars configured in `.env`.

## Running Migrations

`GOOSE_TABLE=custom.goose_migrations` stores version history in the `custom` schema. Create that schema once before the first run:

```bash
psql "$DATABASE_URL" -f scripts/bootstrap-goose-schema.sql
goose up
```

On Windows (PowerShell):

```powershell
.\scripts\migrate-up.ps1
```

```bash
# Apply all pending migrations
goose up

# Rollback one migration
goose down

# Check current status
goose status

# Create a new migration
goose create nama_migration sql
```

## Migration Order

| # | File | Description |
|---|------|-------------|
| 001 | `001_init_schema.sql` | Core tables: users, posts, tags, posts_to_tags, profiles, sessions, post_comments, files |
| 002 | `002_add_post_views_and_user_follows.sql` | post_views table, user_follows table, view/follow count triggers |
| 003 | `003_add_post_likes.sql` | post_likes table, like_count trigger |
| 004 | `004_add_bookmark_folders_and_post_bookmarks.sql` | bookmark_folders, post_bookmarks, bookmark_count trigger |
| 005 | `005_add_chat_conversations_and_messages.sql` | chat_conversations, chat_messages |
| 006 | `006_add_holding_types_and_holdings.sql` | holding_types, holdings |
| 007 | `007_add_notifications.sql` | notifications |
| 008 | `008_add_password_reset_tokens_and_auth_activity_logs.sql` | password_reset_tokens, auth_activity_logs |
| 009 | `009_use_uuidv7_default.sql` | Switch UUID primary key defaults to `uuidv7()` |
| 010 | `010_drop_uuid_ossp.sql` | Drop unused `uuid-ossp` extension |
| 011 | `011_recompute_user_follow_counts.sql` | One-time backfill: recompute `followers_count`/`following_count` from `user_follows` (repair double-counted values) |
| 012 | `012_add_corporate_actions.sql` | corporate_actions |
| 013 | `013_add_corporate_action_cum_rec_dates.sql` | `cum_date`/`rec_date` on corporate_actions |
| 014 | `014_drop_soft_delete_posts_and_post_likes.sql` | **Destructive.** Drop `deleted_at` from `posts` and `post_likes`; purges already-soft-deleted rows first and recomputes `like_count` |

## Notes

- All `CREATE TABLE` and `ADD COLUMN` statements use `IF NOT EXISTS` / `IF NOT EXISTS` for idempotency
- Foreign key constraints with `ON DELETE CASCADE` ensure data integrity
- Database triggers automatically maintain count fields (view_count, like_count, bookmark_count, followers_count, following_count)
- Soft deletes (`deleted_at`) apply to `users`, `post_views`, `post_comments`, `user_follows`, `files` and `chat_conversations` only. `posts` and `post_likes` are hard-deleted as of 014, and their children follow through `ON DELETE CASCADE`
- A query that opts out of GORM's model scope with `.Table(...)` does **not** get the `deleted_at IS NULL` predicate for free — write it by hand
- UUID primary keys use `uuidv7()` by default (PostgreSQL 18+)
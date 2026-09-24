# Database Migrations

Managed via [goose](https://github.com/pressly/goose). Env vars configured in `.env`.

## Running Migrations

`GOOSE_TABLE=custom.goose_migrations` stores version history in the `custom` schema. Create that schema once before the first run:

```bash
# External Postgres only (Docker Compose applies scripts/init-db.sql automatically on init):
psql "$DATABASE_URL" -f scripts/init-db.sql
goose up
```

On Windows (PowerShell / Command Prompt):

```powershell
# Ensure GOOSE_* variables are set in .env, then run:
goose up
# Or via make (under Git Bash/WSL):
make migrate-up
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

## Notes

- All `CREATE TABLE` and `ADD COLUMN` statements use `IF NOT EXISTS` / `IF NOT EXISTS` for idempotency
- Foreign key constraints with `ON DELETE CASCADE` ensure data integrity
- Database triggers automatically maintain count fields (view_count, like_count, bookmark_count, followers_count, following_count)
- Soft deletes (`deleted_at`) apply to `users`, `post_views`, `post_comments`, `user_follows`, `files` and `chat_conversations` only. `posts` and `post_likes` are hard-deleted as of 014, and their children follow through `ON DELETE CASCADE`
- A query that opts out of GORM's model scope with `.Table(...)` does **not** get the `deleted_at IS NULL` predicate for free — write it by hand
- UUID primary keys use `uuidv7()` by default (PostgreSQL 18+)
- `guild_channels` and `guild_channel_messages` (20260924090000) are hard-deleted: deleting a guild cascades to its channels, and deleting a channel to its messages. Every guild has a `general` channel — the migration backfills it, and new guilds get it in the same transaction as the owner membership

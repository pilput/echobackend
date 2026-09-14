-- +goose Up
-- Email lookups compare LOWER(email) so addresses differing only in case map to
-- one account; this index keeps those lookups from scanning the users table.
--
-- It is intentionally not UNIQUE: existing rows may already contain emails that
-- differ only in case, which would make this migration fail. New accounts are
-- stored lowercase and checked case-insensitively by the application.
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email)) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_email_lower;

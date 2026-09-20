-- +goose Up
-- ============================================
-- Refresh token rotation with reuse detection
-- ============================================
-- Implements RFC 9700 (OAuth 2.0 Security BCP) §4.14 — rotation plus replay
-- detection — and OWASP ASVS v5.0 V7 7.3.1/7.3.2 — an inactivity timeout and a
-- separate absolute session lifetime — on top of the existing sessions table.
--
-- A rotation chain is a "family": every token minted from one login shares a
-- family_id. Rotated rows are kept instead of deleted, because a replayed token
-- must stay recognisable long enough to revoke the family it belongs to. They
-- are pruned once expires_at passes, at which point the token could no longer
-- have been used anyway.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS id UUID NOT NULL DEFAULT uuidv7();
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS family_id UUID;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS absolute_expires_at TIMESTAMPTZ;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMPTZ;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS replaced_by TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);

-- Backfill. Each pre-existing session becomes the root of its own family and
-- keeps its current deadline as the absolute one, so nobody is logged out by
-- this migration and nobody gains lifetime from it either.
UPDATE sessions SET created_at = now() WHERE created_at IS NULL;
UPDATE sessions SET expires_at = now() + interval '3 days' WHERE expires_at IS NULL;
UPDATE sessions SET family_id = id WHERE family_id IS NULL;
UPDATE sessions SET absolute_expires_at = expires_at WHERE absolute_expires_at IS NULL;

ALTER TABLE sessions ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE sessions ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE sessions ALTER COLUMN expires_at SET NOT NULL;
ALTER TABLE sessions ALTER COLUMN family_id SET NOT NULL;
ALTER TABLE sessions ALTER COLUMN absolute_expires_at SET NOT NULL;

-- refresh_token stops being the primary key so a row keeps a stable identity
-- across rotations and can be pointed at by replaced_by. It stays unique.
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_pkey;
ALTER TABLE sessions ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);
ALTER TABLE sessions ADD CONSTRAINT sessions_refresh_token_key UNIQUE (refresh_token);

-- Revoking a family on replay scans by family_id; pruning scans by deadline.
CREATE INDEX IF NOT EXISTS idx_sessions_family_id ON sessions(family_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires_at ON sessions(user_id, expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_user_expires_at;
DROP INDEX IF EXISTS idx_sessions_family_id;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_refresh_token_key;
ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_pkey;
-- Only unrotated rows are still usable; the rest exist purely for replay
-- detection and have no meaning once refresh_token is the primary key again.
DELETE FROM sessions WHERE rotated_at IS NOT NULL;
ALTER TABLE sessions ADD CONSTRAINT sessions_pkey PRIMARY KEY (refresh_token);
ALTER TABLE sessions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE sessions DROP COLUMN IF EXISTS replaced_by;
ALTER TABLE sessions DROP COLUMN IF EXISTS rotated_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS absolute_expires_at;
ALTER TABLE sessions DROP COLUMN IF EXISTS family_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS id;
ALTER TABLE sessions ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE sessions ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE sessions ALTER COLUMN expires_at DROP NOT NULL;

-- +goose Up
-- One-time tokens emailed to a user to confirm deleting their own account.
-- Only the SHA-256 hash of the token is stored.
CREATE TABLE IF NOT EXISTS account_deletion_tokens (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_deletion_tokens_token ON account_deletion_tokens(token);
CREATE INDEX IF NOT EXISTS idx_account_deletion_tokens_user_id ON account_deletion_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS account_deletion_tokens;

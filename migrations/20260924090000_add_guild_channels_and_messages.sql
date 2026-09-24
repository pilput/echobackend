-- +goose Up
-- ============================================
-- Guild channels: named text channels inside a guild (Discord-style).
-- Hard-deleted; deleting a guild cascades to its channels and their messages.
-- ============================================
CREATE TABLE IF NOT EXISTS guild_channels (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    guild_id UUID NOT NULL,
    -- Normalised to lowercase-with-hyphens by the service, like a Discord channel.
    name VARCHAR(100) NOT NULL,
    topic TEXT,
    position INTEGER NOT NULL DEFAULT 0,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_guild_channels_position CHECK (position >= 0)
);

-- Also serves the per-guild channel list: a guild has a handful of channels,
-- so sorting them by position needs no index of its own.
CREATE UNIQUE INDEX IF NOT EXISTS idx_guild_channels_guild_name
    ON guild_channels(guild_id, name);

ALTER TABLE guild_channels
    ADD CONSTRAINT fk_guild_channels_guild_id
    FOREIGN KEY (guild_id) REFERENCES guilds(id) ON DELETE CASCADE;
ALTER TABLE guild_channels
    ADD CONSTRAINT fk_guild_channels_created_by
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- ============================================
-- Guild channel messages — the hottest write table, so it carries only the
-- indexes its queries and foreign keys need.
-- ============================================
CREATE TABLE IF NOT EXISTS guild_channel_messages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    channel_id UUID NOT NULL,
    author_id UUID NOT NULL,
    content TEXT NOT NULL,
    reply_to_id UUID,
    -- The only mutable column is content; edited_at doubles as updated_at.
    edited_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_guild_channel_messages_content_length
        CHECK (char_length(content) BETWEEN 1 AND 4000)
);

-- History pages walk backwards by id: uuidv7 is time-ordered, so id order is
-- send order and doubles as the pagination cursor. Also backs the channel
-- ON DELETE CASCADE.
CREATE INDEX IF NOT EXISTS idx_guild_channel_messages_channel_id
    ON guild_channel_messages(channel_id, id DESC);
-- Backs the reply_to_id ON DELETE SET NULL: without it every message delete
-- would scan the whole table for replies. Partial, since most messages are
-- not replies.
CREATE INDEX IF NOT EXISTS idx_guild_channel_messages_reply_to_id
    ON guild_channel_messages(reply_to_id)
    WHERE reply_to_id IS NOT NULL;
-- No author_id index: users are soft-deleted, so the users FK cascade only
-- runs on a manual purge, which can afford a scan. Add one if a per-author
-- query ever appears.

ALTER TABLE guild_channel_messages
    ADD CONSTRAINT fk_guild_channel_messages_channel_id
    FOREIGN KEY (channel_id) REFERENCES guild_channels(id) ON DELETE CASCADE;
ALTER TABLE guild_channel_messages
    ADD CONSTRAINT fk_guild_channel_messages_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
-- A deleted parent leaves the reply standing, just without its quote.
ALTER TABLE guild_channel_messages
    ADD CONSTRAINT fk_guild_channel_messages_reply_to_id
    FOREIGN KEY (reply_to_id) REFERENCES guild_channel_messages(id) ON DELETE SET NULL;

-- Every existing guild gets the #general channel new guilds are created with.
INSERT INTO guild_channels (guild_id, name, created_by)
SELECT id, 'general', owner_id FROM guilds
ON CONFLICT (guild_id, name) DO NOTHING;

-- +goose Down
ALTER TABLE guild_channel_messages DROP CONSTRAINT IF EXISTS fk_guild_channel_messages_reply_to_id;
ALTER TABLE guild_channel_messages DROP CONSTRAINT IF EXISTS fk_guild_channel_messages_author_id;
ALTER TABLE guild_channel_messages DROP CONSTRAINT IF EXISTS fk_guild_channel_messages_channel_id;
ALTER TABLE guild_channels DROP CONSTRAINT IF EXISTS fk_guild_channels_created_by;
ALTER TABLE guild_channels DROP CONSTRAINT IF EXISTS fk_guild_channels_guild_id;

DROP TABLE IF EXISTS guild_channel_messages;
DROP TABLE IF EXISTS guild_channels;

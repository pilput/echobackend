-- +goose Up
-- ============================================
-- Guilds: the top-level community entity.
-- Everything added later (posts, events, jobs) hangs off guild_id.
-- ============================================
CREATE TABLE IF NOT EXISTS guilds (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    avatar_url TEXT,
    -- Public guilds are listed in the directory and anyone may join.
    -- Private guilds stay unlisted; gated join lands in a later migration.
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    member_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_guilds_member_count_positive CHECK (member_count >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_guilds_slug ON guilds(slug);
CREATE INDEX IF NOT EXISTS idx_guilds_owner_id ON guilds(owner_id);
-- Directory listing: public guilds ordered by size.
CREATE INDEX IF NOT EXISTS idx_guilds_public_member_count
    ON guilds(member_count DESC, created_at DESC)
    WHERE is_public = TRUE;

ALTER TABLE guilds
    ADD CONSTRAINT fk_guilds_owner_id
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;

-- ============================================
-- Guild members
-- ============================================
CREATE TABLE IF NOT EXISTS guild_members (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    guild_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_guild_members_role CHECK (role IN ('owner', 'admin', 'member'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_guild_members_unique
    ON guild_members(guild_id, user_id);
CREATE INDEX IF NOT EXISTS idx_guild_members_user_id ON guild_members(user_id);
-- Member list, newest first.
CREATE INDEX IF NOT EXISTS idx_guild_members_guild_created
    ON guild_members(guild_id, created_at DESC);

ALTER TABLE guild_members
    ADD CONSTRAINT fk_guild_members_guild_id
    FOREIGN KEY (guild_id) REFERENCES guilds(id) ON DELETE CASCADE;
ALTER TABLE guild_members
    ADD CONSTRAINT fk_guild_members_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ============================================
-- Trigger: auto-update guilds.member_count
-- ============================================
-- Goose splits a migration on semicolons, so the function body — which has
-- several of its own — has to be handed over as one statement.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_guild_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE guilds SET member_count = member_count + 1 WHERE id = NEW.guild_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE guilds SET member_count = member_count - 1 WHERE id = OLD.guild_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trigger_update_guild_member_count_insert ON guild_members;
CREATE TRIGGER trigger_update_guild_member_count_insert
    AFTER INSERT ON guild_members
    FOR EACH ROW
    EXECUTE FUNCTION update_guild_member_count();

DROP TRIGGER IF EXISTS trigger_update_guild_member_count_delete ON guild_members;
CREATE TRIGGER trigger_update_guild_member_count_delete
    AFTER DELETE ON guild_members
    FOR EACH ROW
    EXECUTE FUNCTION update_guild_member_count();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_update_guild_member_count_delete ON guild_members;
DROP TRIGGER IF EXISTS trigger_update_guild_member_count_insert ON guild_members;
DROP FUNCTION IF EXISTS update_guild_member_count();

ALTER TABLE guild_members DROP CONSTRAINT IF EXISTS fk_guild_members_user_id;
ALTER TABLE guild_members DROP CONSTRAINT IF EXISTS fk_guild_members_guild_id;
ALTER TABLE guilds DROP CONSTRAINT IF EXISTS fk_guilds_owner_id;

DROP TABLE IF EXISTS guild_members;
DROP TABLE IF EXISTS guilds;

-- +goose Up
-- ============================================
-- Moderation: hide a post without deleting it (reversible, keeps the evidence).
-- A hidden post is excluded from every public listing and cannot be liked,
-- commented on or bookmarked; its author and admins can still open it.
-- NULL = visible.
-- ============================================
ALTER TABLE posts ADD COLUMN IF NOT EXISTS hidden_at TIMESTAMPTZ;

-- ============================================
-- Moderation audit log: one row per admin decision, so a decision is stored
-- once however many reports it closes, and unhide leaves a trace too.
-- post_id / post_author_id survive the post (SET NULL) so "how often has this
-- author been moderated" outlives a deleted post.
-- ============================================
CREATE TABLE IF NOT EXISTS post_moderation_actions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    post_id UUID,
    post_author_id UUID,
    admin_id UUID,
    action VARCHAR(16) NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_post_moderation_actions_action CHECK (action IN ('hide', 'unhide', 'dismiss')),
    CONSTRAINT chk_post_moderation_actions_note_length CHECK (note IS NULL OR char_length(note) <= 1000)
);

CREATE INDEX IF NOT EXISTS idx_post_moderation_actions_post
    ON post_moderation_actions(post_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_post_moderation_actions_author
    ON post_moderation_actions(post_author_id);

ALTER TABLE post_moderation_actions
    ADD CONSTRAINT fk_post_moderation_actions_post_id
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE SET NULL;
ALTER TABLE post_moderation_actions
    ADD CONSTRAINT fk_post_moderation_actions_post_author_id
    FOREIGN KEY (post_author_id) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE post_moderation_actions
    ADD CONSTRAINT fk_post_moderation_actions_admin_id
    FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE SET NULL;

-- ============================================
-- Post reports: a user's report of a post. Reports die with their post (posts
-- are hard-deleted, see 014); the audit log above is what outlives it.
-- ============================================
CREATE TABLE IF NOT EXISTS post_reports (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    post_id UUID NOT NULL,
    reporter_id UUID NOT NULL,
    reason VARCHAR(32) NOT NULL,
    details TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    -- The decision that closed this report; NULL while pending.
    action_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_post_reports_reason CHECK (reason IN ('spam', 'harassment', 'hate', 'nsfw', 'misinformation', 'other')),
    CONSTRAINT chk_post_reports_status CHECK (status IN ('pending', 'resolved', 'dismissed')),
    CONSTRAINT chk_post_reports_details_length CHECK (details IS NULL OR char_length(details) <= 1000)
);

-- One *open* report per user per post. A closed report does not block a new
-- one, so a dismissed reporter can report the post again if it changes.
CREATE UNIQUE INDEX IF NOT EXISTS idx_post_reports_pending_post_reporter
    ON post_reports(post_id, reporter_id) WHERE status = 'pending';
-- "Reports of this post" and the ON DELETE CASCADE lookup.
CREATE INDEX IF NOT EXISTS idx_post_reports_post_id
    ON post_reports(post_id);
-- Admin queue: reports by status, newest first.
CREATE INDEX IF NOT EXISTS idx_post_reports_status_created
    ON post_reports(status, created_at DESC);
-- Cascade lookup when a reporter is deleted.
CREATE INDEX IF NOT EXISTS idx_post_reports_reporter_id
    ON post_reports(reporter_id);

ALTER TABLE post_reports
    ADD CONSTRAINT fk_post_reports_post_id
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE;
ALTER TABLE post_reports
    ADD CONSTRAINT fk_post_reports_reporter_id
    FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE post_reports
    ADD CONSTRAINT fk_post_reports_action_id
    FOREIGN KEY (action_id) REFERENCES post_moderation_actions(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE post_reports DROP CONSTRAINT IF EXISTS fk_post_reports_action_id;
ALTER TABLE post_reports DROP CONSTRAINT IF EXISTS fk_post_reports_reporter_id;
ALTER TABLE post_reports DROP CONSTRAINT IF EXISTS fk_post_reports_post_id;
DROP TABLE IF EXISTS post_reports;
ALTER TABLE post_moderation_actions DROP CONSTRAINT IF EXISTS fk_post_moderation_actions_admin_id;
ALTER TABLE post_moderation_actions DROP CONSTRAINT IF EXISTS fk_post_moderation_actions_post_author_id;
ALTER TABLE post_moderation_actions DROP CONSTRAINT IF EXISTS fk_post_moderation_actions_post_id;
DROP TABLE IF EXISTS post_moderation_actions;
ALTER TABLE posts DROP COLUMN IF EXISTS hidden_at;

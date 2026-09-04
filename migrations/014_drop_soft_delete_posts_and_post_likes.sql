-- +goose Up

-- ============================================
-- posts: drop soft delete
-- ============================================
-- Rows with deleted_at set are already invisible to the application. They MUST be
-- purged before the column goes away, otherwise dropping it would silently
-- resurrect every previously deleted post as a live one.
--
-- Their children (post_comments, post_views, post_likes, post_bookmarks,
-- posts_to_tags) follow via the existing ON DELETE CASCADE foreign keys. The
-- count triggers on those child tables fire and try to decrement posts.view_count
-- / like_count / bookmark_count, but the parent row is gone within the same
-- statement, so those updates match nothing. Harmless.
--
-- The column-existence guard is not paranoia: some environments have tables that
-- predate these migrations (created by GORM AutoMigrate, which only ever emitted
-- the columns the models declared), so deleted_at may already be absent. A plain
-- DELETE would abort the whole migration there. Dynamic SQL keeps the reference
-- from being parsed when the branch is skipped.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'posts'::regclass
          AND attname = 'deleted_at'
          AND NOT attisdropped
    ) THEN
        EXECUTE 'DELETE FROM posts WHERE deleted_at IS NOT NULL';
    END IF;
END $$;
-- +goose StatementEnd

-- creator_and_slug_unique and idx_posts_deleted_at both depend on the column and
-- would be dropped implicitly by DROP COLUMN. Replace the uniqueness guarantee
-- explicitly: with no soft-deleted rows left, the partial predicate is redundant
-- and a plain unique index enforces the same rule.
DROP INDEX IF EXISTS creator_and_slug_unique;
CREATE UNIQUE INDEX IF NOT EXISTS creator_and_slug_unique ON posts(created_by, slug);
DROP INDEX IF EXISTS idx_posts_deleted_at;

ALTER TABLE posts DROP COLUMN IF EXISTS deleted_at;

-- ============================================
-- post_likes: drop soft delete
-- ============================================
-- model.PostLike never mapped this column, so DeleteLike has always issued a hard
-- DELETE and the UPDATE branch of update_post_like_count() was unreachable. Align
-- the schema with the behaviour the code has always had.
--
-- Depending on how the environment was built the column may not be there at all,
-- so every step below is written to be a no-op in that case.
DROP TRIGGER IF EXISTS trigger_update_post_like_count_update ON post_likes;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_attribute
        WHERE attrelid = 'post_likes'::regclass
          AND attname = 'deleted_at'
          AND NOT attisdropped
    ) THEN
        EXECUTE 'DELETE FROM post_likes WHERE deleted_at IS NOT NULL';
    END IF;
END $$;
-- +goose StatementEnd

-- The AFTER DELETE trigger decrements like_count once per purged row, but a row
-- that had been soft-deleted was already decremented back when deleted_at was
-- set. Rather than reason about which case applies to historical data, recompute
-- every counter from the rows that survive.
UPDATE posts p
SET like_count = sub.cnt
FROM (
    SELECT p2.id, COUNT(pl.id) AS cnt
    FROM posts p2
    LEFT JOIN post_likes pl ON pl.post_id = p2.id
    GROUP BY p2.id
) sub
WHERE p.id = sub.id AND p.like_count IS DISTINCT FROM sub.cnt;

DROP INDEX IF EXISTS idx_post_likes_unique_user_post;
CREATE UNIQUE INDEX IF NOT EXISTS idx_post_likes_unique_user_post ON post_likes(post_id, user_id);
DROP INDEX IF EXISTS idx_post_likes_deleted_at;

ALTER TABLE post_likes DROP COLUMN IF EXISTS deleted_at;

-- +goose Down

-- NOTE: rows purged by the Up migration are gone for good; this only restores the
-- columns, indexes and trigger, not the data.

ALTER TABLE post_likes ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_post_likes_deleted_at ON post_likes(deleted_at);
DROP INDEX IF EXISTS idx_post_likes_unique_user_post;
CREATE UNIQUE INDEX IF NOT EXISTS idx_post_likes_unique_user_post
ON post_likes(post_id, user_id)
WHERE deleted_at IS NULL;

-- Recreated only where update_post_like_count() is actually present, and dropped
-- first so a repeated rollback does not trip over an existing trigger.
DROP TRIGGER IF EXISTS trigger_update_post_like_count_update ON post_likes;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_post_like_count') THEN
        EXECUTE 'CREATE TRIGGER trigger_update_post_like_count_update'
             || ' AFTER UPDATE ON post_likes FOR EACH ROW'
             || ' EXECUTE FUNCTION update_post_like_count()';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE posts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_posts_deleted_at ON posts(deleted_at);
DROP INDEX IF EXISTS creator_and_slug_unique;
CREATE UNIQUE INDEX IF NOT EXISTS creator_and_slug_unique
ON posts(created_by, slug)
WHERE deleted_at IS NULL;

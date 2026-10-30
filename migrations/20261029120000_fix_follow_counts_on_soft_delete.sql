-- +goose Up
-- ============================================
-- Keep users.followers_count / following_count correct across soft deletes.
--
-- Migration 002 attached update_user_follow_counts() only AFTER INSERT and
-- AFTER DELETE on user_follows. Every backend unfollows with a soft delete
-- (UPDATE user_follows SET deleted_at = now()), which neither trigger sees, so
-- the stored counts never went down and a re-follow (a fresh row, allowed by
-- the partial unique index) added 1 on top. This migration:
--   1. makes the function count only live rows (deleted_at IS NULL) and adds
--      an UPDATE branch for deleted_at transitions (soft delete / restore),
--   2. attaches it AFTER UPDATE OF deleted_at,
--   3. recomputes both columns from the live rows (same as 011).
-- ============================================
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_user_follow_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.deleted_at IS NULL THEN
            UPDATE users SET following_count = following_count + 1 WHERE id = NEW.follower_id;
            UPDATE users SET followers_count = followers_count + 1 WHERE id = NEW.following_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- A row that was already soft-deleted was already subtracted.
        IF OLD.deleted_at IS NULL THEN
            UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = OLD.follower_id;
            UPDATE users SET followers_count = GREATEST(followers_count - 1, 0) WHERE id = OLD.following_id;
        END IF;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            -- Soft delete (unfollow).
            UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = NEW.follower_id;
            UPDATE users SET followers_count = GREATEST(followers_count - 1, 0) WHERE id = NEW.following_id;
        ELSIF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
            -- Restore of a soft-deleted follow.
            UPDATE users SET following_count = following_count + 1 WHERE id = NEW.follower_id;
            UPDATE users SET followers_count = followers_count + 1 WHERE id = NEW.following_id;
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trigger_update_follow_counts_update ON user_follows;
CREATE TRIGGER trigger_update_follow_counts_update
    AFTER UPDATE OF deleted_at ON user_follows
    FOR EACH ROW
    EXECUTE FUNCTION update_user_follow_counts();

-- Repair the counts that drifted before this trigger existed (idempotent).
UPDATE users u
SET following_count = COALESCE(c.cnt, 0)
FROM (
    SELECT u2.id, COUNT(uf.follower_id)::bigint AS cnt
    FROM users u2
    LEFT JOIN user_follows uf
      ON uf.follower_id = u2.id AND uf.deleted_at IS NULL
    GROUP BY u2.id
) c
WHERE u.id = c.id;

UPDATE users u
SET followers_count = COALESCE(c.cnt, 0)
FROM (
    SELECT u2.id, COUNT(uf.following_id)::bigint AS cnt
    FROM users u2
    LEFT JOIN user_follows uf
      ON uf.following_id = u2.id AND uf.deleted_at IS NULL
    GROUP BY u2.id
) c
WHERE u.id = c.id;

-- +goose Down
DROP TRIGGER IF EXISTS trigger_update_follow_counts_update ON user_follows;

-- Restore the 002 function body. The recomputed counts stay in place.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_user_follow_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Increment follower's following count
        UPDATE users SET following_count = following_count + 1 WHERE id = NEW.follower_id;
        -- Increment following user's followers count
        UPDATE users SET followers_count = followers_count + 1 WHERE id = NEW.following_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- Decrement follower's following count
        UPDATE users SET following_count = following_count - 1 WHERE id = OLD.follower_id;
        -- Decrement following user's followers count
        UPDATE users SET followers_count = followers_count - 1 WHERE id = OLD.following_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

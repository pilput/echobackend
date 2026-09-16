-- +goose Up
-- ============================================
-- 1. Many-to-Many posts_to_tags optimization
-- ============================================
-- Primary key is (post_id, tag_id). Lookups/joins by tag_id (e.g. GetPostsByTag,
-- tag filters, trending tags) cannot use the primary key index and would otherwise
-- force a full table scan on posts_to_tags.
CREATE INDEX IF NOT EXISTS idx_posts_to_tags_tag_id ON posts_to_tags(tag_id);

-- ============================================
-- 2. Posts public feed optimization
-- ============================================
-- Public post feed (GetPosts, GetPostsFiltered, GetPostsForYou) filters published posts
-- ordered by created_at DESC.
CREATE INDEX IF NOT EXISTS idx_posts_published_created_at ON posts(published, created_at DESC);

-- ============================================
-- 3. Post comments pagination optimization
-- ============================================
-- GetCommentsByPostID filters by post_id and deleted_at IS NULL, ordered by created_at DESC.
CREATE INDEX IF NOT EXISTS idx_post_comments_post_created_at 
ON post_comments(post_id, created_at DESC) 
WHERE deleted_at IS NULL;

-- ============================================
-- 4. Author posts listing optimization
-- ============================================
-- GetPostsByCreatedBy (/posts/me and author profiles) retrieves author's posts ordered by created_at DESC.
CREATE INDEX IF NOT EXISTS idx_posts_created_by_created_at ON posts(created_by, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_posts_created_by_created_at;
DROP INDEX IF EXISTS idx_post_comments_post_created_at;
DROP INDEX IF EXISTS idx_posts_published_created_at;
DROP INDEX IF EXISTS idx_posts_to_tags_tag_id;


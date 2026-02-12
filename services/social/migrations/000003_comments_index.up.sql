-- Create composite index for top-level comments by score and recency
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comments_anime_top ON comments(anime_id, score DESC, created_at DESC) WHERE parent_id IS NULL AND deleted_at IS NULL;

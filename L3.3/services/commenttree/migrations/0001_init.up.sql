CREATE TABLE IF NOT EXISTS comments (
  id         BIGSERIAL PRIMARY KEY,
  parent_id  BIGINT NULL REFERENCES comments(id) ON DELETE CASCADE,
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);

CREATE INDEX IF NOT EXISTS idx_comments_root_created
ON comments(created_at DESC)
WHERE parent_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_body_fts
ON comments
USING GIN (to_tsvector('russian', body));


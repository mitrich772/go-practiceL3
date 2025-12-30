CREATE TABLE IF NOT EXISTS links (
    short TEXT PRIMARY KEY,
    original TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_links_created_at
    ON links (created_at);

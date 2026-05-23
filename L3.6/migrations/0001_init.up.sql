CREATE TABLE IF NOT EXISTS items (
    id          BIGSERIAL PRIMARY KEY,
    type        TEXT        NOT NULL CHECK (type IN ('income', 'expense')),
    amount      NUMERIC(14, 2) NOT NULL CHECK (amount >= 0),
    category    TEXT        NOT NULL CHECK (length(category) > 0 AND length(category) <= 64),
    note        TEXT        NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_items_occurred_at ON items (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_category    ON items (category);
CREATE INDEX IF NOT EXISTS idx_items_type        ON items (type);

CREATE TABLE clicks (
    id          BIGSERIAL PRIMARY KEY,
    short       TEXT NOT NULL,
    clicked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip          INET,
    user_agent  TEXT,
    referer     TEXT
);
CREATE INDEX idx_clicks_short ON clicks (short);
CREATE INDEX idx_clicks_clicked_at ON clicks (clicked_at);
CREATE INDEX idx_clicks_short_time ON clicks (short, clicked_at);

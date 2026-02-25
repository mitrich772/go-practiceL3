-- Таблица для хранения статуса обработки изображений

CREATE TABLE IF NOT EXISTS images (
    id            TEXT PRIMARY KEY,
    status        TEXT NOT NULL CHECK (status IN ('processing', 'ready', 'failed')),
    original_path TEXT NOT NULL,
    processed_path TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы под частые запросы
CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at);

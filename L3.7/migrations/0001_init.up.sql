CREATE TABLE IF NOT EXISTS users (
    id       BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    role     TEXT NOT NULL CHECK (role IN ('admin', 'manager', 'viewer'))
);

CREATE TABLE IF NOT EXISTS items (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT           NOT NULL CHECK (length(name) > 0 AND length(name) <= 120),
    sku         TEXT           NOT NULL UNIQUE CHECK (length(sku) > 0 AND length(sku) <= 64),
    quantity    INTEGER        NOT NULL CHECK (quantity >= 0),
    location    TEXT           NOT NULL DEFAULT '' CHECK (length(location) <= 120),
    description TEXT           NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS item_history (
    id          BIGSERIAL PRIMARY KEY,
    item_id     BIGINT,
    action      TEXT        NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    actor       TEXT        NOT NULL DEFAULT 'system',
    actor_role  TEXT        NOT NULL DEFAULT 'system',
    old_data    JSONB,
    new_data    JSONB,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (username, role) VALUES
    ('admin', 'admin'),
    ('manager', 'manager'),
    ('viewer', 'viewer')
ON CONFLICT (username) DO UPDATE SET role = EXCLUDED.role;

CREATE INDEX IF NOT EXISTS idx_items_name        ON items (name);
CREATE INDEX IF NOT EXISTS idx_items_sku         ON items (sku);
CREATE INDEX IF NOT EXISTS idx_item_history_item ON item_history (item_id, changed_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION audit_item_changes()
RETURNS trigger AS $$
DECLARE
    v_actor text := COALESCE(NULLIF(current_setting('app.actor', true), ''), 'system');
    v_role  text := COALESCE(NULLIF(current_setting('app.role', true), ''), 'system');
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO item_history (item_id, action, actor, actor_role, old_data, new_data)
        VALUES (NEW.id, TG_OP, v_actor, v_role, NULL, to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO item_history (item_id, action, actor, actor_role, old_data, new_data)
        VALUES (NEW.id, TG_OP, v_actor, v_role, to_jsonb(OLD), to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO item_history (item_id, action, actor, actor_role, old_data, new_data)
        VALUES (OLD.id, TG_OP, v_actor, v_role, to_jsonb(OLD), NULL);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_items_updated_at ON items;
CREATE TRIGGER trg_items_updated_at
BEFORE UPDATE ON items
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_items_audit ON items;
CREATE TRIGGER trg_items_audit
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW EXECUTE FUNCTION audit_item_changes();

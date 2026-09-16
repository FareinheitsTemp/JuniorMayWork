-- 006_entity_integrity.sql — Фаза A рендизайну БД v2 (адитивна, без ламання коду).
-- Джерело дизайну: затверджений користувачем DB_DESIGN_V2.
-- Типи FK-колонок підігнані під існуючі типи v1 (orders.id INTEGER,
-- reports.id BIGINT, sources.id INTEGER); нові таблиці — BIGINT GENERATED
-- ALWAYS AS IDENTITY за дизайном. Ідемпотентно: IF NOT EXISTS / ON CONFLICT.

-- ---------- ДОВІДНИКИ ----------

CREATE TABLE IF NOT EXISTS order_statuses (
  id          SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  key         TEXT NOT NULL UNIQUE,
  label       TEXT NOT NULL,
  color       TEXT NOT NULL DEFAULT '',
  sort        SMALLINT NOT NULL DEFAULT 0,
  is_terminal BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO order_statuses (key, label, color, sort, is_terminal) VALUES
  ('new',       'Нове',          '#3fcf8e', 1, FALSE),
  ('seen',      'Переглянуто',   '#ffb547', 2, FALSE),
  ('applied',   'Подана заявка', '#5b8cff', 3, FALSE),
  ('interview', 'Інтервʼю',      '#a371f7', 4, FALSE),
  ('offer',     'Офлер',         '#2ea87a', 5, FALSE),
  ('rejected',  'Відмова',       '#ff6b7a', 6, TRUE),
  ('archived',  'Архів',         '#8f9cb2', 7, TRUE)
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS application_results (
  id    SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  key   TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL
);

INSERT INTO application_results (key, label) VALUES
  ('pending',   'Очікує'),
  ('accepted',  'Прийнято'),
  ('declined',  'Відмовлено')
ON CONFLICT (key) DO NOTHING;

-- Словники для фази B (нормалізація навичок і тегів) — створюються заздалегідь.
CREATE TABLE IF NOT EXISTS skills (
  id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  slug TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS tags (
  id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

-- ---------- STATUS_ID + BACKFILL ----------

ALTER TABLE orders ADD COLUMN IF NOT EXISTS status_id SMALLINT REFERENCES order_statuses(id) ON DELETE RESTRICT;

-- Легасі-ключі won/lost мапляться на offer/rejected.
UPDATE orders o
SET    status_id = s.id
FROM   order_statuses s
WHERE  o.status_id IS NULL
  AND  s.key = CASE o.status WHEN 'won' THEN 'offer' WHEN 'lost' THEN 'rejected' ELSE o.status END;

-- ---------- ЖИТТЄВИЙ ЦИКЛ ----------

CREATE TABLE IF NOT EXISTS order_status_history (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  order_id       Integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  from_status_id SMALLINT REFERENCES order_statuses(id) ON DELETE SET NULL,
  to_status_id   SMALLINT NOT NULL REFERENCES order_statuses(id) ON DELETE RESTRICT,
  actor          TEXT,
  changed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_osh_order ON order_status_history (order_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS order_notes (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  order_id   INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  body       TEXT NOT NULL CHECK (length(trim(body)) > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_order_notes_order ON order_notes (order_id, created_at DESC);

-- ---------- АУДИТ І ТЕЛЕМЕТРІЯ ----------

CREATE TABLE IF NOT EXISTS audit_log (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  entity     TEXT NOT NULL,
  entity_id  BIGINT NOT NULL,
  action     TEXT NOT NULL,
  actor      TEXT,
  payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log (entity, entity_id, created_at DESC);

CREATE TABLE IF NOT EXISTS report_downloads (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  report_id     BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  downloaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_agent    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_report_downloads_report ON report_downloads (report_id);

CREATE TABLE IF NOT EXISTS search_run_daily_stats (
  run_date     DATE NOT NULL,
  source_id    INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  orders_found INTEGER NOT NULL DEFAULT 0 CHECK (orders_found >= 0),
  runs_count   INTEGER NOT NULL DEFAULT 0 CHECK (runs_count >= 0),
  PRIMARY KEY (run_date, source_id)
);

-- ---------- ВІДСУТНІ FK-ІНДЕКСИ ----------

CREATE INDEX IF NOT EXISTS idx_events_order         ON events (order_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_type          ON events (type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_applications_order   ON applications (order_id);
CREATE INDEX IF NOT EXISTS idx_applications_applied ON applications (applied_at DESC);
CREATE INDEX IF NOT EXISTS idx_source_runs_source   ON source_runs (source_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_run          ON reports (search_run_id);
CREATE INDEX IF NOT EXISTS idx_orders_status_id     ON orders (status_id);
CREATE INDEX IF NOT EXISTS idx_orders_duplicate    ON orders (duplicate_of_id) WHERE duplicate_of_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_spt_profile          ON search_profile_tags (profile_id);
CREATE INDEX IF NOT EXISTS idx_sps_profile          ON search_profile_sources (profile_id);

-- ---------- updated_at ----------

ALTER TABLE orders          ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE sources         ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE branches        ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE search_profiles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE search_runs     ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_touch_orders ON orders;
CREATE TRIGGER trg_touch_orders BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_touch_sources ON sources;
CREATE TRIGGER trg_touch_sources BEFORE UPDATE ON sources
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_touch_branches ON branches;
CREATE TRIGGER trg_touch_branches BEFORE UPDATE ON branches
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_touch_search_profiles ON search_profiles;
CREATE TRIGGER trg_touch_search_profiles BEFORE UPDATE ON search_profiles
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_touch_search_runs ON search_runs;
CREATE TRIGGER trg_touch_search_runs BEFORE UPDATE ON search_runs
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ---------- VIEW ДЛЯ UI ----------

CREATE OR REPLACE VIEW v_order_board AS
SELECT o.id, o.title, o.url, o.budget_cents, o.currency, o.priority_score,
       o.first_seen_at, o.branch_id, b.name AS branch_name,
       s.key AS status_key, s.label AS status_label, s.color AS status_color
FROM orders o
JOIN order_statuses s ON s.id = o.status_id
LEFT JOIN branches b ON b.id = o.branch_id
WHERE NOT o.is_dismissed;

CREATE OR REPLACE VIEW v_source_health AS
SELECT so.id, so.key, so.name, so.enabled, so.last_success_at,
       sr.started_at AS last_run_at, sr.outcome AS last_outcome,
       sr.discovered_count AS last_discovered
FROM sources so
LEFT JOIN LATERAL (
  SELECT * FROM source_runs sr
  WHERE sr.source_id = so.id ORDER BY sr.started_at DESC LIMIT 1
) sr ON TRUE;

CREATE OR REPLACE VIEW v_daily_intake AS
SELECT date_trunc('day', first_seen_at)::date AS day,
       count(*) AS orders_total,
       count(*) FILTER (WHERE duplicate_of_id IS NOT NULL) AS duplicates
FROM orders GROUP BY 1 ORDER BY 1 DESC;

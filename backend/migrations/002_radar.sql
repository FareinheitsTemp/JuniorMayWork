-- JuniorMayWork Radar: health джерел, свіжість, дедуплікація та карта БД.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS source_published_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS content_fingerprint TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS freshness_score NUMERIC(8,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS relevance_score NUMERIC(8,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS priority_score NUMERIC(8,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS is_seen_by_user BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS is_dismissed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS first_user_seen_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS duplicate_of_id INTEGER REFERENCES orders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_orders_priority ON orders(priority_score DESC, first_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_fresh_unseen ON orders(is_seen_by_user, is_dismissed, priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_orders_fingerprint ON orders(content_fingerprint, first_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_published ON orders(source_published_at DESC);

-- Налаштування і здоров'я кожного каналу збору.
CREATE TABLE IF NOT EXISTS sources (
  id                    SERIAL PRIMARY KEY,
  key                   TEXT NOT NULL UNIQUE,
  name                  TEXT NOT NULL,
  kind                  TEXT NOT NULL CHECK (kind IN ('api', 'telegram', 'rss', 'html')),
  base_url              TEXT NOT NULL DEFAULT '',
  enabled               BOOLEAN NOT NULL DEFAULT TRUE,
  poll_interval_seconds INTEGER NOT NULL DEFAULT 300 CHECK (poll_interval_seconds >= 30),
  last_success_at       TIMESTAMPTZ,
  last_failure_at       TIMESTAMPTZ,
  last_error            TEXT NOT NULL DEFAULT '',
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Окремі канали Telegram, якими можна керувати без зміни коду чи .env.
CREATE TABLE IF NOT EXISTS source_channels (
  id         SERIAL PRIMARY KEY,
  source_id  INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  handle     TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL DEFAULT '',
  enabled    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Кожна спроба збору: для pipeline, health і чесної діагностики.
CREATE TABLE IF NOT EXISTS source_runs (
  id               SERIAL PRIMARY KEY,
  source_id        INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  started_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at      TIMESTAMPTZ,
  outcome          TEXT NOT NULL DEFAULT 'running'
                   CHECK (outcome IN ('running', 'success', 'warning', 'failure')),
  discovered_count INTEGER NOT NULL DEFAULT 0,
  inserted_count   INTEGER NOT NULL DEFAULT 0,
  updated_count    INTEGER NOT NULL DEFAULT 0,
  duplicate_count  INTEGER NOT NULL DEFAULT 0,
  http_status      INTEGER,
  error_message    TEXT NOT NULL DEFAULT '',
  meta             JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_source_runs_source_started ON source_runs(source_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_source_runs_outcome ON source_runs(outcome, started_at DESC);

-- Персональний layout ERD-мапи. Ноди зберігають координати, розмір і стан.
CREATE TABLE IF NOT EXISTS schema_layouts (
  id         SERIAL PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  viewport   JSONB NOT NULL DEFAULT '{"x":0,"y":0,"zoom":1}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS schema_nodes (
  id          SERIAL PRIMARY KEY,
  layout_id   INTEGER NOT NULL REFERENCES schema_layouts(id) ON DELETE CASCADE,
  table_key   TEXT NOT NULL,
  x           NUMERIC(10,2) NOT NULL DEFAULT 0,
  y           NUMERIC(10,2) NOT NULL DEFAULT 0,
  width       NUMERIC(10,2) NOT NULL DEFAULT 240,
  height      NUMERIC(10,2) NOT NULL DEFAULT 160,
  color       TEXT NOT NULL DEFAULT '#5b8cff',
  collapsed   BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (layout_id, table_key)
);

-- Українські джерела, якими керує Radar.
INSERT INTO sources (key, name, kind, base_url, poll_interval_seconds)
VALUES
  ('freelancehunt', 'Freelancehunt API', 'api', 'https://api.freelancehunt.com/v2/projects', 120),
  ('telegram', 'Telegram — український фріланс', 'telegram', 'https://t.me/s/', 300),
  ('weblancer', 'Weblancer', 'html', 'https://www.weblancer.net/jobs/', 900)
ON CONFLICT (key) DO NOTHING;

INSERT INTO source_channels (source_id, handle, name)
SELECT id, 'freelance_for_ukraine', 'Український Freelance'
FROM sources WHERE key = 'telegram'
ON CONFLICT (handle) DO NOTHING;

INSERT INTO schema_layouts (name)
VALUES ('Основна карта')
ON CONFLICT (name) DO NOTHING;

INSERT INTO schema_nodes (layout_id, table_key, x, y, width, height, color)
SELECT l.id, v.table_key, v.x, v.y, v.width, v.height, v.color
FROM schema_layouts l
CROSS JOIN (VALUES
  ('sources', 80::numeric, 100::numeric, 240::numeric, 170::numeric, '#5b8cff'),
  ('source_channels', 80::numeric, 360::numeric, 240::numeric, 145::numeric, '#8b5cf6'),
  ('source_runs', 420::numeric, 100::numeric, 240::numeric, 170::numeric, '#f59e0b'),
  ('branches', 420::numeric, 430::numeric, 220::numeric, 145::numeric, '#a855f7'),
  ('orders', 780::numeric, 250::numeric, 270::numeric, 210::numeric, '#3fcf8e'),
  ('applications', 1160::numeric, 120::numeric, 240::numeric, 160::numeric, '#5b8cff'),
  ('events', 1160::numeric, 410::numeric, 240::numeric, 170::numeric, '#f97316'),
  ('schema_nodes', 780::numeric, 620::numeric, 270::numeric, 145::numeric, '#64748b')
) AS v(table_key, x, y, width, height, color)
WHERE l.name = 'Основна карта'
ON CONFLICT (layout_id, table_key) DO NOTHING;

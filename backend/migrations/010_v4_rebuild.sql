-- 010_v4_rebuild.sql — destructive rebuild: схема v4 замінює v2 (міграції 001–009) повністю.
-- Дані не переносяться (дев-БД вбудована, заповнюється скрейпером).
-- Джерело істини дизайну: docs/DB_DESIGN_V4.md.

-- ============================================================
-- 1. Скидаємо схему v2.
--    schema_migrations НЕ чіпаємо: записи 001–009 лишаються «applied»
--    і не виконаються повторно.
-- ============================================================
DROP TABLE IF EXISTS order_discoveries      CASCADE;
DROP TABLE IF EXISTS search_runs            CASCADE;
DROP TABLE IF EXISTS search_profile_sources CASCADE;
DROP TABLE IF EXISTS search_profile_tags    CASCADE;
DROP TABLE IF EXISTS search_profiles        CASCADE;
DROP TABLE IF EXISTS report_exports         CASCADE;
DROP TABLE IF EXISTS report_downloads       CASCADE;
DROP TABLE IF EXISTS reports                CASCADE;
DROP TABLE IF EXISTS order_notes            CASCADE;
DROP TABLE IF EXISTS order_status_history   CASCADE;
DROP TABLE IF EXISTS order_skills           CASCADE;
DROP TABLE IF EXISTS skills                 CASCADE;
DROP TABLE IF EXISTS order_statuses         CASCADE;
DROP TABLE IF EXISTS events                CASCADE;
DROP TABLE IF EXISTS applications          CASCADE;
DROP TABLE IF EXISTS orders                CASCADE;
DROP TABLE IF EXISTS source_runs            CASCADE;
DROP TABLE IF EXISTS source_channels        CASCADE;
DROP TABLE IF EXISTS sources               CASCADE;
DROP TABLE IF EXISTS branches              CASCADE;
DROP TABLE IF EXISTS schema_nodes           CASCADE;

-- Залишки v2 з міграцій 002/006, які не покриває CASCADE вище
-- (FK-обмеження дропаються разом із цільовою таблицею, а не навпаки):
DROP TABLE IF EXISTS search_run_daily_stats CASCADE;
DROP TABLE IF EXISTS application_results    CASCADE;
DROP TABLE IF EXISTS audit_log              CASCADE;
DROP TABLE IF EXISTS tags                   CASCADE;
DROP TABLE IF EXISTS schema_layouts         CASCADE;

-- Старі view-и v2 (міграція 006):
DROP VIEW IF EXISTS v_order_board;
DROP VIEW IF EXISTS v_source_health;
DROP VIEW IF EXISTS v_daily_intake;

-- ============================================================
-- 2. ДОВІДНИКИ
-- ============================================================

-- Джерела збору замовлень.
CREATE TABLE sources (
  id      INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  key     TEXT NOT NULL UNIQUE,              -- 'kabanchik', 'djinni' ...
  name    TEXT NOT NULL,
  kind    TEXT NOT NULL CHECK (kind IN ('api','telegram','rss','html')),
  enabled BOOLEAN NOT NULL DEFAULT TRUE
);

-- Окремі Telegram-канали всередині джерела.
CREATE TABLE source_channels (
  id        INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  handle    TEXT NOT NULL UNIQUE,
  title     TEXT NOT NULL DEFAULT '',
  enabled   BOOLEAN NOT NULL DEFAULT TRUE
);

-- Довідник навичок (поповнюється скрейпером).
CREATE TABLE skills (
  id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

-- ============================================================
-- 3. ЯДРО: ЗАМОВЛЕННЯ
-- ============================================================

CREATE TABLE orders (
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_id         INTEGER NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
  source_message_id TEXT,                    -- ID постингу на джерелі (дедуп)
  title             TEXT NOT NULL,
  status            TEXT NOT NULL DEFAULT 'new'
                    CHECK (status IN ('new','seen','won','lost','archived')),
  budget_cents      INTEGER CHECK (budget_cents IS NULL OR budget_cents >= 0),
  external_url      TEXT NOT NULL DEFAULT '',
  published_at      TIMESTAMPTZ,             -- дата публікації на джерелі
  first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),  -- для пошуку зниклих
  UNIQUE (source_id, source_message_id)      -- дедуп: той самий постинг двічі
);

-- M:N: які навички вимагає замовлення.
CREATE TABLE order_skills (
  order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
  PRIMARY KEY (order_id, skill_id)
);

-- ============================================================
-- 4. ПРОФІЛІ ПОШУКУ (замість віток)
-- ============================================================

CREATE TABLE profiles (
  id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name             TEXT NOT NULL UNIQUE,
  max_budget_cents INTEGER CHECK (max_budget_cents IS NULL OR max_budget_cents >= 0),
  date_from        DATE,                      -- період публікації замовлення
  date_to          DATE,
  results_limit    INTEGER NOT NULL DEFAULT 50
                   CHECK (results_limit BETWEEN 1 AND 1000),
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (date_from IS NULL OR date_to IS NULL OR date_from <= date_to)
);

-- M:N: які навички/мови шукає профіль (React АБО Go АБО Node).
CREATE TABLE profile_skills (
  profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  skill_id   BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  PRIMARY KEY (profile_id, skill_id)
);

-- M:N: у яких джерелах профіль шукає.
CREATE TABLE profile_sources (
  profile_id BIGINT  NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  source_id  INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  PRIMARY KEY (profile_id, source_id)
);

-- ============================================================
-- 5. ЗАПУСКИ (виконання профілю)
-- ============================================================

CREATE TABLE scrape_runs (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  profile_id  BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ,
  outcome     TEXT NOT NULL DEFAULT 'ok' CHECK (outcome IN ('ok','error')),
  error_text  TEXT,
  stats       JSONB NOT NULL DEFAULT '{}'::jsonb,   -- {fetched, matched, new}
  CHECK (finished_at IS NULL OR finished_at >= started_at)
);

-- M:N: що конкретний запуск знайшов.
CREATE TABLE order_matches (
  run_id          BIGINT NOT NULL REFERENCES scrape_runs(id) ON DELETE CASCADE,
  order_id        BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  relevance_score INTEGER NOT NULL DEFAULT 0,
  captured_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (run_id, order_id)
);

-- ============================================================
-- 6. АРХІВ ЗНИКЛИХ ЗАМОВЛЕНЬ
-- ============================================================

-- Снапшот: source зберігається ключем (без FK) — джерело може бути
-- видалене вже після архівації замовлення, історія мусить вижити.
CREATE TABLE archived_orders (
  id                BIGINT PRIMARY KEY,           -- id перенесений з orders
  source_key        TEXT NOT NULL,
  title             TEXT NOT NULL,
  status            TEXT NOT NULL,
  budget_cents      INTEGER,
  external_url      TEXT NOT NULL DEFAULT '',
  snapshot          JSONB NOT NULL,               -- повна копія рядка orders
  reason            TEXT NOT NULL DEFAULT 'removed'
                    CHECK (reason IN ('removed','manual')),
  archived_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- 7. НАЛАШТУВАННЯ ПРОГИ
-- ============================================================

CREATE TABLE app_settings (
  key   TEXT PRIMARY KEY,
  value JSONB NOT NULL
);

-- ============================================================
-- 8. ІНДЕКСИ
-- ============================================================

-- UNIQUE (source_id, source_message_id) уже індексує пошук по source_id.
CREATE INDEX idx_orders_status      ON orders (status);
CREATE INDEX idx_orders_first_seen  ON orders (first_seen_at DESC);
CREATE INDEX idx_orders_published   ON orders (published_at DESC);
CREATE INDEX idx_orders_last_seen   ON orders (last_seen_at);       -- пошук stale

CREATE INDEX idx_channels_source    ON source_channels (source_id);
CREATE INDEX idx_order_skills_skill ON order_skills (skill_id);      -- обернений бік M:N
CREATE INDEX idx_pskills_skill       ON profile_skills (skill_id);
CREATE INDEX idx_psources_source     ON profile_sources (source_id);
CREATE INDEX idx_runs_profile        ON scrape_runs (profile_id, started_at DESC);
CREATE INDEX idx_matches_order       ON order_matches (order_id);   -- «де це замовлення знаходили"

-- ============================================================
-- 9. ЗВІТНИЙ ШАР (view-и — «вихідні таблиці»)
-- ============================================================

-- Широка картка замовлення: усе про нього в одному рядку (експорт, PDF).
CREATE VIEW v_orders_full AS
SELECT
    o.id,
    o.title,
    o.status,
    o.budget_cents,
    s.key  AS source_key,
    s.name AS source_name,
    o.external_url,
    o.published_at,
    o.first_seen_at,
    array_agg(sk.name ORDER BY sk.name) AS skills
FROM orders o
JOIN sources s            ON s.id = o.source_id
LEFT JOIN order_skills os  ON os.order_id = o.id
LEFT JOIN skills sk       ON sk.id = os.skill_id
GROUP BY o.id, s.key, s.name;

-- Донат: замовлення за статусами.
CREATE VIEW v_status_summary AS
SELECT status, count(*) AS orders_count
FROM orders
GROUP BY status;

-- Попит за навичками (бар + середній бюджет).
CREATE VIEW v_skill_demand AS
SELECT
    sk.name AS skill,
    count(*) AS orders_count,
    round(avg(o.budget_cents) / 100.0, 2) AS avg_budget_usd
FROM skills sk
JOIN order_skills os ON os.skill_id = sk.id
JOIN orders o        ON o.id = os.order_id
GROUP BY sk.name;

-- Динаміка по днях (лінійні графіки).
CREATE VIEW v_daily_dynamics AS
SELECT
    first_seen_at::date AS day,
    count(*)                                AS new_orders,
    count(*) FILTER (WHERE status = 'won')  AS won_orders,
    count(*) FILTER (WHERE status = 'lost') AS lost_orders
FROM orders
GROUP BY day;

-- Ефективність профілів.
CREATE VIEW v_profile_stats AS
SELECT
    p.id,
    p.name,
    count(DISTINCT r.id)       AS total_runs,
    count(DISTINCT m.order_id) AS unique_orders_found,
    max(r.started_at)          AS last_run_at
FROM profiles p
LEFT JOIN scrape_runs r   ON r.profile_id = p.id
LEFT JOIN order_matches m ON m.run_id = r.id
GROUP BY p.id, p.name;

-- Історія зникнень (архів).
CREATE VIEW v_archive_history AS
SELECT
    archived_at::date AS day,
    source_key,
    count(*)          AS disappeared,
    reason
FROM archived_orders
GROUP BY day, source_key, reason;

-- 008_normalization.sql — Фаза B дизайну v2: нормалізація даних.
-- orders.source_id (FK) + backfill; skills/order_skills M:N + backfill
-- з orders.skills TEXT[]; criteria JSONB для додаткових фільтрів.
-- Ідемпотентно: IF NOT EXISTS / ON CONFLICT DO NOTHING.

-- ---------- ORDERS.SOURCE_ID ----------

-- Гарантія: кожне значення orders.source має рядок у sources.
INSERT INTO sources (key, name, kind, base_url, poll_interval_seconds)
SELECT DISTINCT o.source, o.source, 'html', '', 900
FROM orders o
WHERE o.source <> ''
ON CONFLICT (key) DO NOTHING;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS source_id INTEGER REFERENCES sources(id) ON DELETE RESTRICT;

UPDATE orders o
SET    source_id = s.id
FROM   sources s
WHERE  o.source_id IS NULL
  AND  o.source <> ''
  AND  s.key = o.source;

CREATE INDEX IF NOT EXISTS idx_orders_source_id ON orders (source_id);

-- ---------- SKILLS / ORDER_SKILLS (M:N) ----------

CREATE TABLE IF NOT EXISTS order_skills (
  order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  skill_id BIGINT  NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
  PRIMARY KEY (order_id, skill_id)
);
CREATE INDEX IF NOT EXISTS idx_order_skills_skill ON order_skills (skill_id);

-- Словник навичок з існуючих масивів (slug — для майбутніх фільтрів).
INSERT INTO skills (name, slug)
SELECT DISTINCT trim(sk), lower(regexp_replace(trim(sk), '\s+', '-', 'g'))
FROM orders o
CROSS JOIN LATERAL unnest(o.skills) AS sk
WHERE trim(sk) <> ''
ON CONFLICT DO NOTHING;

INSERT INTO order_skills (order_id, skill_id)
SELECT o.id, s2.id
FROM orders o
CROSS JOIN LATERAL unnest(o.skills) AS sk
JOIN skills s2 ON s2.name = trim(sk)
WHERE trim(sk) <> ''
ON CONFLICT DO NOTHING;

-- ---------- SEARCH_PROFILES: CRITERIA JSONB ----------
-- У 003_search_profiles.sql скалярні колонки (interval_minutes, min/max_budget_cents,
-- max_age_hours) вже створено первинно. Тут лише додаємо criteria JSONB як
-- гнучке сховище надлишкових фільтрів з дефолтом '{}'.
ALTER TABLE search_profiles ADD COLUMN IF NOT EXISTS criteria JSONB NOT NULL DEFAULT '{}'::jsonb;

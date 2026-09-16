-- 008_normalization.sql — Фаза B дизайну v2: нормалізація даних.
-- orders.source_id (FK) + backfill; skills/order_skills M:N + backfill
-- з orders.skills TEXT[]; скалярні критерії search_profiles + backfill
-- з criteria JSONB.
-- Старі колонки (orders.source, orders.skills, search_profiles.criteria)
-- зберігаються до повного переходу Go-коду на нові структури —
-- тому міграція не ламає жоден існуючий INSERT/SELECT.
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

-- ---------- SEARCH_PROFILES: СКАЛЯРНІ КРИТЕРІЇ ----------

ALTER TABLE search_profiles
  ADD COLUMN IF NOT EXISTS interval_minutes INTEGER CHECK (interval_minutes IS NULL OR interval_minutes > 0),
  ADD COLUMN IF NOT EXISTS min_budget_cents INTEGER CHECK (min_budget_cents IS NULL OR min_budget_cents >= 0),
  ADD COLUMN IF NOT EXISTS max_budget_cents INTEGER CHECK (max_budget_cents IS NULL OR max_budget_cents >= 0),
  ADD COLUMN IF NOT EXISTS max_age_hours    INTEGER CHECK (max_age_hours IS NULL OR max_age_hours > 0);

UPDATE search_profiles
SET    interval_minutes = CASE WHEN criteria->>'interval_minutes' ~ '^[0-9]+$' THEN (criteria->>'interval_minutes')::int ELSE 15 END,
       min_budget_cents = CASE WHEN criteria->>'min_budget_cents' ~ '^[0-9]+$' THEN (criteria->>'min_budget_cents')::int ELSE NULL END,
       max_budget_cents = CASE WHEN criteria->>'max_budget_cents' ~ '^[0-9]+$' THEN (criteria->>'max_budget_cents')::int ELSE NULL END,
       max_age_hours    = CASE WHEN criteria->>'max_age_hours' ~ '^[0-9]+$' THEN (criteria->>'max_age_hours')::int ELSE NULL END
WHERE  interval_minutes IS NULL
  AND  min_budget_cents IS NULL
  AND  max_budget_cents IS NULL
  AND  max_age_hours IS NULL;

-- Примітка: перенесення тегів у lookup-схему (search_profile_tags має ще й
-- tag_type, який враховано в дизайні) і прибрання applications.order_title
-- відкладено до відповідної правки store/search_profiles.go — теперішня
-- структура коду пише туди напряму, і зміна без неї зламає збірку.

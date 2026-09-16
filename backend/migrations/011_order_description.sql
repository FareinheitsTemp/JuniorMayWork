-- 011_order_description.sql — повертаємо опис замовлення у v4.
-- Причина: скрейпери витягують опис, UI показує, пошук/підбір за ключовими
-- словами (title + description) точніший, ніж лише за заголовком.

ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

-- v_orders_full тепер повертає і опис:
DROP VIEW IF EXISTS v_orders_full;
CREATE VIEW v_orders_full AS
SELECT
    o.id,
    o.title,
    o.description,
    o.status,
    o.budget_cents,
    s.key  AS source_key,
    s.name AS source_name,
    o.external_url,
    o.published_at,
    o.first_seen_at,
    array_agg(sk.name ORDER BY sk.name) AS skills
FROM orders o
JOIN sources s             ON s.id = o.source_id
LEFT JOIN order_skills os  ON os.order_id = o.id
LEFT JOIN skills sk        ON sk.id = os.skill_id
GROUP BY o.id, s.key, s.name;

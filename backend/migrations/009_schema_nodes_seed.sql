-- 009_schema_nodes_seed.sql — нові таблиці v2 на ERD-мапі (як 004 для пошукових).
-- Ідемпотентно: не додає ноди, якщо вони вже існують у даному layout.
INSERT INTO schema_nodes (layout_id, table_key, x, y, width, height, color)
SELECT l.id, v.table_key, v.x, v.y, v.width, v.height, v.color
FROM schema_layouts AS l
CROSS JOIN (VALUES
  ('order_statuses',         1290,   40, 220, 200, '#7aa2f7'),
  ('order_status_history',   1290,  270, 220, 220, '#7dcfff'),
  ('order_notes',            1290,  520, 220, 180, '#7dcfff'),
  ('order_skills',           1290,  730, 220, 180, '#bb9af7'),
  ('skills',                 1530,  730, 220, 180, '#bb9af7'),
  ('application_results',    1530,   40, 220, 180, '#7aa2f7'),
  ('tags',                   1530,  270, 220, 160, '#bb9af7'),
  ('audit_log',              1530,  470, 220, 200, '#9ece6a'),
  ('report_downloads',       1530,  710, 220, 180, '#9ece6a'),
  ('search_run_daily_stats', 1770,   40, 240, 200, '#9ece6a')
) AS v(table_key, x, y, width, height, color)
WHERE l.name = 'Основна карта'
  AND NOT EXISTS (
    SELECT 1 FROM schema_nodes n
    WHERE n.layout_id = l.id AND n.table_key = v.table_key
  );

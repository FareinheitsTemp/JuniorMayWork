-- Нові таблиці пошуку (міграція 003) на ERD-мапі: search_profiles, search_runs, reports.
-- Ідемпотентно: не додає ноди, якщо вони вже існують у даному layout.
INSERT INTO schema_nodes (layout_id, table_key, x, y, width, height, color)
SELECT l.id, v.table_key, v.x, v.y, v.width, v.height, v.color
FROM schema_layouts AS l
CROSS JOIN (VALUES
  ('search_profiles', 1010, 40,  220, 240, '#7aa2f7'),
  ('search_runs',      1010, 310, 220, 240, '#7aa2f7'),
  ('reports',          1010, 580, 220, 220, '#7dcfff')
) AS v(table_key, x, y, width, height, color)
WHERE l.name = 'Основна карта'
  AND NOT EXISTS (
    SELECT 1 FROM schema_nodes n
    WHERE n.layout_id = l.id AND n.table_key = v.table_key
  );

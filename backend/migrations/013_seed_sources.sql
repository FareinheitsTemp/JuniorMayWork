-- 013_seed_sources.sql — стартові джерела для локального скрейпера v4.
-- UpsertListing шукає sources.id за key, тому записи мають існувати до першого запуску.

INSERT INTO sources (key, name, kind, enabled) VALUES
  ('freelancehunt', 'Freelancehunt API', 'api', TRUE),
  ('telegram',      'Telegram — український фріланс', 'telegram', TRUE),
  ('dou',           'DOU — вакансії для початківців', 'html', TRUE),
  ('djinni',        'Djinni — Junior вакансії', 'html', TRUE)
ON CONFLICT (key) DO NOTHING;

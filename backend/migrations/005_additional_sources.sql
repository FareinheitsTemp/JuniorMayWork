-- Додавання нових джерел вакансій/замовлень: DOU та Djinni
INSERT INTO sources (key, name, kind, base_url, poll_interval_seconds)
VALUES
  ('dou', 'DOU — вакансії для початківців', 'html', 'https://jobs.dou.ua/vacancies/?exp=0-1', 900),
  ('djinni', 'Djinni — Junior вакансії', 'html', 'https://djinni.co/jobs/keyword-junior/', 900)
ON CONFLICT (key) DO NOTHING;

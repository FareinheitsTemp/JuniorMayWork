-- 012_source_runs.sql — телеметрія окремих джерел збору.
-- source_runs відрізняється від scrape_runs:
-- source_runs = технічне здоров'я конкретного джерела (DOU, Djinni, Telegram);
-- scrape_runs = результат виконання профілю користувача.

CREATE TABLE source_runs (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_id   INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ,
  outcome     TEXT NOT NULL DEFAULT 'running'
              CHECK (outcome IN ('running','ok','error')),
  error_text  TEXT NOT NULL DEFAULT '',
  stats       JSONB NOT NULL DEFAULT '{}'::jsonb,
  CHECK (finished_at IS NULL OR finished_at >= started_at)
);

CREATE INDEX idx_source_runs_source_started
  ON source_runs (source_id, started_at DESC);

CREATE VIEW v_source_health AS
SELECT
  s.id,
  s.key,
  s.name,
  s.kind,
  s.enabled,
  r.started_at AS last_started_at,
  r.finished_at AS last_finished_at,
  r.outcome AS last_outcome,
  r.error_text AS last_error,
  r.stats AS last_stats
FROM sources s
LEFT JOIN LATERAL (
  SELECT started_at, finished_at, outcome, error_text, stats
  FROM source_runs
  WHERE source_id = s.id
  ORDER BY started_at DESC
  LIMIT 1
) r ON TRUE;

-- 007_report_type.sql — тип звіту згідно з дизайном v2 (run | summary).
ALTER TABLE reports ADD COLUMN IF NOT EXISTS report_type TEXT NOT NULL DEFAULT 'run'
  CHECK (report_type IN ('run', 'summary'));

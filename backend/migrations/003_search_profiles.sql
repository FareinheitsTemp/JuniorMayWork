-- Guided search profiles, their scheduled/manual runs, provenance of discovered
-- orders, and immutable report-export metadata. This migration is additive:
-- existing Radar tables and orders remain the canonical source of current data.

CREATE TABLE IF NOT EXISTS search_profiles (
    id                  BIGSERIAL PRIMARY KEY,
    name                TEXT NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    interval_minutes    INTEGER NOT NULL DEFAULT 15 CHECK (interval_minutes BETWEEN 5 AND 1440),
    min_budget_cents    INTEGER CHECK (min_budget_cents IS NULL OR min_budget_cents >= 0),
    max_budget_cents    INTEGER CHECK (max_budget_cents IS NULL OR max_budget_cents >= 0),
    max_age_hours       INTEGER CHECK (max_age_hours IS NULL OR max_age_hours BETWEEN 1 AND 720),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (max_budget_cents IS NULL OR min_budget_cents IS NULL OR max_budget_cents >= min_budget_cents)
);

CREATE TABLE IF NOT EXISTS search_profile_tags (
    id                  BIGSERIAL PRIMARY KEY,
    profile_id          BIGINT NOT NULL REFERENCES search_profiles(id) ON DELETE CASCADE,
    tag                 TEXT NOT NULL,
    tag_type            TEXT NOT NULL CHECK (tag_type IN ('role', 'skill', 'language', 'technology')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, tag, tag_type)
);

CREATE TABLE IF NOT EXISTS search_profile_sources (
    profile_id          BIGINT NOT NULL REFERENCES search_profiles(id) ON DELETE CASCADE,
    source_id           BIGINT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    is_enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (profile_id, source_id)
);

CREATE TABLE IF NOT EXISTS search_runs (
    id                  BIGSERIAL PRIMARY KEY,
    profile_id          BIGINT NOT NULL REFERENCES search_profiles(id) ON DELETE CASCADE,
    trigger             TEXT NOT NULL DEFAULT 'manual' CHECK (trigger IN ('manual', 'scheduled', 'startup')),
    status              TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'completed', 'partial', 'failed')),
    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    duration_ms         BIGINT CHECK (duration_ms IS NULL OR duration_ms >= 0),
    orders_found        INTEGER NOT NULL DEFAULT 0 CHECK (orders_found >= 0),
    orders_new          INTEGER NOT NULL DEFAULT 0 CHECK (orders_new >= 0),
    duplicates_count    INTEGER NOT NULL DEFAULT 0 CHECK (duplicates_count >= 0),
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_discoveries (
    id                  BIGSERIAL PRIMARY KEY,
    order_id            BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    search_run_id       BIGINT NOT NULL REFERENCES search_runs(id) ON DELETE CASCADE,
    source_id           BIGINT NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    source_channel_id   BIGINT REFERENCES source_channels(id) ON DELETE SET NULL,
    published_at        TIMESTAMPTZ,
    captured_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    relevance_score     DOUBLE PRECISION NOT NULL DEFAULT 0,
    matched_tags        JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_new_order        BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (order_id, search_run_id, source_id)
);

CREATE TABLE IF NOT EXISTS report_exports (
    id                  BIGSERIAL PRIMARY KEY,
    search_run_id       BIGINT NOT NULL REFERENCES search_runs(id) ON DELETE CASCADE,
    file_name           TEXT NOT NULL,
    storage_key         TEXT NOT NULL,
    mime_type           TEXT NOT NULL DEFAULT 'application/pdf',
    orders_total        INTEGER NOT NULL DEFAULT 0 CHECK (orders_total >= 0),
    summary_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_search_profiles_active ON search_profiles (is_active) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_search_profile_tags_profile ON search_profile_tags (profile_id);
CREATE INDEX IF NOT EXISTS idx_search_runs_profile_created ON search_runs (profile_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_runs_status ON search_runs (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_discoveries_run ON order_discoveries (search_run_id, relevance_score DESC);
CREATE INDEX IF NOT EXISTS idx_order_discoveries_order ON order_discoveries (order_id, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_run ON report_exports (search_run_id, created_at DESC);

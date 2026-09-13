# Search Profiles, Runs and PDF Reports

This document defines the target architecture for JuniorMayWork's guided job search and reporting flow.

## Product flow

1. A user creates or updates a search profile.
2. The profile contains role, technology and language tags, budget bounds, a freshness limit and enabled sources.
3. The user starts a run immediately or keeps the profile active for scheduled collection.
4. Enabled sources are collected in parallel. A completed run records its actual duration; the 15-minute value is the refresh cadence, not an artificial wait time.
5. Raw source items are normalized, deduplicated, scored and linked to the run that discovered them.
6. The user reviews priority-ranked jobs and exports a PDF report for one completed run.

## Search profile data model

### search_profiles

| Column | Meaning |
|---|---|
| id | Primary key |
| name | Human-readable profile name |
| is_active | Enables scheduled collection after service restart |
| interval_minutes | Collection cadence, default 15 |
| min_budget_cents | Optional minimum budget |
| max_budget_cents | Optional maximum budget |
| max_age_hours | Maximum acceptable age of a source post |
| created_at / updated_at | Audit timestamps |

### search_profile_tags

Tags are normalized rather than stored as an opaque JSON array.

| Column | Meaning |
|---|---|
| id | Primary key |
| profile_id | Foreign key to search_profiles |
| tag | The selected value, for example React or Go |
| tag_type | role, skill, language or technology |

A unique index on `(profile_id, tag, tag_type)` prevents duplicate chips in a profile.

## Run and discovery model

### search_runs

A run is one concrete execution of a profile. It stores `started_at`, `finished_at`, `duration_ms`, `status`, `orders_found`, `orders_new` and an optional error message.

### order_discoveries

An order remains canonical in `orders`, while every encounter is retained as a discovery record.

| Column | Meaning |
|---|---|
| order_id | Canonical order |
| search_run_id | Run that found it |
| source_id | Source that supplied it |
| source_channel_id | Optional Telegram channel |
| published_at | Original source publication time |
| captured_at | Time JuniorMayWork observed it |
| relevance_score | Profile-specific score at discovery time |

This makes it possible to distinguish a newly discovered job from a known duplicate and to report exact source provenance.

## Existing source model

- `sources` describes an integration such as Freelancehunt, Telegram or Weblancer.
- `source_channels` is a one-to-many list of user-managed Telegram handles or similar subfeeds.
- `source_runs` is operational source telemetry; it is separate from `search_runs`, which is a user-facing profile execution.
- `orders` is unique by source identity / external identity and links to `branches`.
- `applications` and `events` remain one-to-many children of `orders`.
- `schema_nodes` is UI metadata for the schema-map canvas and must be visually separated from operational business tables.

## Scheduler behavior

At application startup, the scheduler loads active profiles and registers each profile's `interval_minutes` cadence. It must also support an immediate manual run. A job never waits for 15 minutes to display an already-collected result: completion time is the actual elapsed collection time.

The scheduler records source-level failures without failing the full run when another source succeeds. Runs are idempotent at the order level because canonical order deduplication remains enforced.

## PDF report

One PDF belongs to one completed `search_run`. The report contains:

- Search profile criteria and selected tags
- Actual start, end and elapsed collection time
- Per-source status, discovered count, new count and errors
- Summary metrics: total, new, duplicates, budget range and average budget
- Priority-ranked job table with title, source, channel, budget, age, score, matching tags and original URL
- Deterministic conclusions based on the collected data, for example strongest source, freshest source and top candidates for a first response

The PDF must describe observed data only. Any conclusions are reproducible rule-based analytics, not fabricated claims.

## Target API surface

- `GET /api/search-profiles`
- `POST /api/search-profiles`
- `PATCH /api/search-profiles/{id}`
- `DELETE /api/search-profiles/{id}`
- `POST /api/search-profiles/{id}/run`
- `GET /api/search-runs/{id}`
- `GET /api/search-runs/{id}/orders`
- `POST /api/search-runs/{id}/report.pdf`
- `GET /api/radar/sources`
- `PATCH /api/radar/sources/{id}`
- `GET /api/radar/sources/{id}/channels`
- `POST /api/radar/sources/{id}/channels`
- `PATCH /api/radar/channels/{id}`
- `DELETE /api/radar/channels/{id}`

## UI direction

The primary dashboard should expose a compact Search Builder: tag chips, budget range, freshness, source selection and a Start Search action. While a run is active, the UI displays real source progress. On completion it opens a priority inbox and enables Export PDF.

The database map should show foreign-key edges, index / unique constraints and a separate UI metadata group for `schema_nodes`.
// Пакет store: Radar — джерела, health/runs, fresh inbox і схема-мапа.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const sourceCols = `id, key, name, kind, base_url, enabled, poll_interval_seconds,
	last_success_at, last_failure_at, last_error, created_at, updated_at`

func scanSource(row pgx.Row) (model.Source, error) {
	var s model.Source
	err := row.Scan(&s.ID, &s.Key, &s.Name, &s.Kind, &s.BaseURL, &s.Enabled,
		&s.PollIntervalSeconds, &s.LastSuccessAt, &s.LastFailureAt, &s.LastError,
		&s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (s *Store) ListSources(ctx context.Context) ([]model.Source, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+sourceCols+` FROM sources ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Source{}
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, rows.Err()
}

func (s *Store) GetSourceByKey(ctx context.Context, key string) (model.Source, error) {
	source, err := scanSource(s.pool.QueryRow(ctx,
		`SELECT `+sourceCols+` FROM sources WHERE key = $1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Source{}, ErrNotFound
	}
	return source, err
}

func (s *Store) UpdateSource(ctx context.Context, id int64, source model.Source) (model.Source, error) {
	if source.PollIntervalSeconds < 30 {
		source.PollIntervalSeconds = 30
	}
	updated, err := scanSource(s.pool.QueryRow(ctx, `
		UPDATE sources SET
			name = $2, base_url = $3, enabled = $4,
			poll_interval_seconds = $5, updated_at = now()
		WHERE id = $1 RETURNING `+sourceCols,
		id, source.Name, source.BaseURL, source.Enabled, source.PollIntervalSeconds))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Source{}, ErrNotFound
	}
	return updated, err
}

const channelCols = `id, source_id, handle, name, enabled, created_at`

func scanChannel(row pgx.Row) (model.SourceChannel, error) {
	var c model.SourceChannel
	err := row.Scan(&c.ID, &c.SourceID, &c.Handle, &c.Name, &c.Enabled, &c.CreatedAt)
	return c, err
}

func (s *Store) ListSourceChannels(ctx context.Context, sourceID int64) ([]model.SourceChannel, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+channelCols+` FROM source_channels WHERE source_id = $1 ORDER BY handle`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SourceChannel{}
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, channel)
	}
	return out, rows.Err()
}

func (s *Store) CreateSourceChannel(ctx context.Context, c model.SourceChannel) (model.SourceChannel, error) {
	created, err := scanChannel(s.pool.QueryRow(ctx, `
		INSERT INTO source_channels (source_id, handle, name, enabled)
		VALUES ($1, $2, $3, $4) RETURNING `+channelCols,
		c.SourceID, c.Handle, c.Name, c.Enabled))
	return created, err
}

func (s *Store) UpdateSourceChannel(ctx context.Context, id int64, c model.SourceChannel) (model.SourceChannel, error) {
	updated, err := scanChannel(s.pool.QueryRow(ctx, `
		UPDATE source_channels SET handle = $2, name = $3, enabled = $4
		WHERE id = $1 RETURNING `+channelCols,
		id, c.Handle, c.Name, c.Enabled))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SourceChannel{}, ErrNotFound
	}
	return updated, err
}

func (s *Store) DeleteSourceChannel(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM source_channels WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const runCols = `id, source_id, started_at, finished_at, outcome, discovered_count,
	inserted_count, updated_count, duplicate_count, http_status, error_message, meta`

func scanRun(row pgx.Row) (model.SourceRun, error) {
	var r model.SourceRun
	err := row.Scan(&r.ID, &r.SourceID, &r.StartedAt, &r.FinishedAt, &r.Outcome,
		&r.DiscoveredCount, &r.InsertedCount, &r.UpdatedCount, &r.DuplicateCount,
		&r.HTTPStatus, &r.ErrorMessage, &r.Meta)
	return r, err
}

func (s *Store) CreateSourceRun(ctx context.Context, sourceID int64) (model.SourceRun, error) {
	return scanRun(s.pool.QueryRow(ctx,
		`INSERT INTO source_runs (source_id) VALUES ($1) RETURNING `+runCols, sourceID))
}

func (s *Store) FinishSourceRun(ctx context.Context, run model.SourceRun) (model.SourceRun, error) {
	meta := run.Meta
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}
	updated, err := scanRun(s.pool.QueryRow(ctx, `
		UPDATE source_runs SET finished_at = now(), outcome = $2,
			discovered_count = $3, inserted_count = $4, updated_count = $5,
			duplicate_count = $6, http_status = $7, error_message = $8, meta = $9
		WHERE id = $1 RETURNING `+runCols,
		run.ID, run.Outcome, run.DiscoveredCount, run.InsertedCount, run.UpdatedCount,
		run.DuplicateCount, run.HTTPStatus, run.ErrorMessage, meta))
	if err != nil {
		return model.SourceRun{}, err
	}
	if run.Outcome == "success" || run.Outcome == "warning" {
		_, err = s.pool.Exec(ctx, `
			UPDATE sources SET last_success_at = now(), last_error = '', updated_at = now()
			WHERE id = $1`, run.SourceID)
	} else if run.Outcome == "failure" {
		_, err = s.pool.Exec(ctx, `
			UPDATE sources SET last_failure_at = now(), last_error = $2, updated_at = now()
			WHERE id = $1`, run.SourceID, run.ErrorMessage)
	}
	return updated, err
}

func (s *Store) ListSourceRuns(ctx context.Context, sourceID *int64, limit int) ([]model.SourceRun, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT ` + runCols + ` FROM source_runs`
	args := []any{}
	if sourceID != nil {
		args = append(args, *sourceID)
		query += ` WHERE source_id = $1`
	}
	args = append(args, limit)
	query += ` ORDER BY started_at DESC LIMIT $` + fmt.Sprint(len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SourceRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

const layoutCols = `id, name, viewport, updated_at`
const nodeCols = `id, layout_id, table_key, x::double precision, y::double precision,
	width::double precision, height::double precision, color, collapsed`

func scanLayout(row pgx.Row) (model.SchemaLayout, error) {
	var l model.SchemaLayout
	err := row.Scan(&l.ID, &l.Name, &l.Viewport, &l.UpdatedAt)
	return l, err
}

func scanNode(row pgx.Row) (model.SchemaNode, error) {
	var n model.SchemaNode
	err := row.Scan(&n.ID, &n.LayoutID, &n.TableKey, &n.X, &n.Y, &n.Width, &n.Height, &n.Color, &n.Collapsed)
	return n, err
}

func (s *Store) GetSchemaLayout(ctx context.Context, name string) (model.SchemaLayout, []model.SchemaNode, error) {
	layout, err := scanLayout(s.pool.QueryRow(ctx,
		`SELECT `+layoutCols+` FROM schema_layouts WHERE name = $1`, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SchemaLayout{}, nil, ErrNotFound
	}
	if err != nil {
		return model.SchemaLayout{}, nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+nodeCols+` FROM schema_nodes WHERE layout_id = $1 ORDER BY id`, layout.ID)
	if err != nil {
		return model.SchemaLayout{}, nil, err
	}
	defer rows.Close()
	nodes := []model.SchemaNode{}
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return model.SchemaLayout{}, nil, err
		}
		nodes = append(nodes, node)
	}
	return layout, nodes, rows.Err()
}

func (s *Store) UpdateSchemaViewport(ctx context.Context, id int64, viewport json.RawMessage) (model.SchemaLayout, error) {
	if viewport == nil {
		viewport = json.RawMessage(`{"x":0,"y":0,"zoom":1}`)
	}
	layout, err := scanLayout(s.pool.QueryRow(ctx, `
		UPDATE schema_layouts SET viewport = $2, updated_at = now()
		WHERE id = $1 RETURNING `+layoutCols, id, viewport))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SchemaLayout{}, ErrNotFound
	}
	return layout, err
}

func (s *Store) MoveSchemaNode(ctx context.Context, id int64, n model.SchemaNode) (model.SchemaNode, error) {
	updated, err := scanNode(s.pool.QueryRow(ctx, `
		UPDATE schema_nodes SET x = $2, y = $3, width = $4, height = $5,
			color = $6, collapsed = $7
		WHERE id = $1 RETURNING `+nodeCols,
		id, n.X, n.Y, n.Width, n.Height, n.Color, n.Collapsed))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SchemaNode{}, ErrNotFound
	}
	return updated, err
}

const freshOrderCols = `id, source, external_id, url, title, description, budget_cents, currency,
	skills, branch_id, status, first_seen_at, last_seen_at, raw,
	source_published_at, freshness_score::double precision, relevance_score::double precision,
	priority_score::double precision, is_seen_by_user, is_dismissed, duplicate_of_id`

func scanFreshOrder(row pgx.Row) (model.FreshOrder, error) {
	var f model.FreshOrder
	err := row.Scan(&f.ID, &f.Source, &f.ExternalID, &f.URL, &f.Title, &f.Description,
		&f.BudgetCents, &f.Currency, &f.Skills, &f.BranchID, &f.Status,
		&f.FirstSeenAt, &f.LastSeenAt, &f.Raw, &f.SourcePublishedAt,
		&f.FreshnessScore, &f.RelevanceScore, &f.PriorityScore,
		&f.IsSeenByUser, &f.IsDismissed, &f.DuplicateOfID)
	return f, err
}

func (s *Store) FreshOrders(ctx context.Context, limit int, minPriority float64) ([]model.FreshOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+freshOrderCols+` FROM orders
		WHERE NOT is_seen_by_user AND NOT is_dismissed AND duplicate_of_id IS NULL
		  AND priority_score >= $1
		ORDER BY priority_score DESC, COALESCE(source_published_at, first_seen_at) DESC
		LIMIT $2`, minPriority, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.FreshOrder{}
	for rows.Next() {
		order, err := scanFreshOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, order)
	}
	return out, rows.Err()
}

func (s *Store) MarkOrderSeen(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE orders SET is_seen_by_user = TRUE, first_user_seen_at = COALESCE(first_user_seen_at, now())
		WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DismissOrder(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE orders SET is_dismissed = TRUE WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SourceRunAge повертає межу для adaptive backoff на майбутньому scraper-шарі.
func SourceRunAge(run model.SourceRun) time.Duration {
	if run.FinishedAt == nil {
		return 0
	}
	return time.Since(*run.FinishedAt)
}

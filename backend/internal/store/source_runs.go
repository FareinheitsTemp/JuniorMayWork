package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const sourceRunCols = `id, source_id, started_at, finished_at, outcome, error_text, stats`

func scanSourceRun(row pgx.Row) (model.SourceRun, error) {
	var r model.SourceRun
	if err := row.Scan(&r.ID, &r.SourceID, &r.StartedAt, &r.FinishedAt, &r.Outcome, &r.ErrorText, &r.Stats); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return r, ErrNotFound }
		return r, err
	}
	return r, nil
}

func (s *Store) StartSourceRun(ctx context.Context, sourceID int32) (model.SourceRun, error) {
	return scanSourceRun(s.pool.QueryRow(ctx,
		`INSERT INTO source_runs (source_id) VALUES ($1) RETURNING `+sourceRunCols, sourceID))
}

func (s *Store) FinishSourceRun(ctx context.Context, id int64, outcome, errorText string, stats model.RunStats) error {
	payload, err := json.Marshal(stats)
	if err != nil { return err }
	tag, err := s.pool.Exec(ctx, `
		UPDATE source_runs
		   SET finished_at = now(), outcome = $1, error_text = $2, stats = $3::jsonb
		 WHERE id = $4`, outcome, errorText, string(payload), id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return ErrNotFound }
	return nil
}

func (s *Store) ListSourceRuns(ctx context.Context, sourceID int32, limit int) ([]model.SourceRun, error) {
	if limit <= 0 || limit > 100 { limit = 20 }
	rows, err := s.pool.Query(ctx, `SELECT `+sourceRunCols+`
		FROM source_runs WHERE source_id=$1 ORDER BY started_at DESC LIMIT $2`, sourceID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []model.SourceRun
	for rows.Next() {
		var r model.SourceRun
		if err := rows.Scan(&r.ID, &r.SourceID, &r.StartedAt, &r.FinishedAt, &r.Outcome, &r.ErrorText, &r.Stats); err != nil { return nil, err }
		list = append(list, r)
	}
	return list, rows.Err()
}

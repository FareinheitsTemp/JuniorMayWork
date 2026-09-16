package store

import (
	"context"
	"errors"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const sourceCols = `id, key, name, kind, enabled`

func scanSource(row pgx.Row) (model.Source, error) {
	var source model.Source
	if err := row.Scan(&source.ID, &source.Key, &source.Name, &source.Kind, &source.Enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return source, ErrNotFound }
		return source, err
	}
	return source, nil
}

func (s *Store) ListSources(ctx context.Context, enabledOnly bool) ([]model.Source, error) {
	query := `SELECT ` + sourceCols + ` FROM sources`
	if enabledOnly { query += ` WHERE enabled` }
	query += ` ORDER BY name`
	rows, err := s.pool.Query(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []model.Source
	for rows.Next() {
		var source model.Source
		if err := rows.Scan(&source.ID, &source.Key, &source.Name, &source.Kind, &source.Enabled); err != nil { return nil, err }
		list = append(list, source)
	}
	return list, rows.Err()
}

func (s *Store) GetSource(ctx context.Context, id int32) (model.Source, error) {
	return scanSource(s.pool.QueryRow(ctx, `SELECT `+sourceCols+` FROM sources WHERE id=$1`, id))
}

// Пакет store: джерела та канали v4.
package store

import (
	"context"
	"errors"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const channelCols = `id, source_id, handle, title, enabled`

func scanChannel(row pgx.Row) (model.SourceChannel, error) {
	var channel model.SourceChannel
	if err := row.Scan(&channel.ID, &channel.SourceID, &channel.Handle, &channel.Title, &channel.Enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return channel, ErrNotFound }
		return channel, err
	}
	return channel, nil
}

func (s *Store) ListSourceChannels(ctx context.Context, sourceID int32) ([]model.SourceChannel, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+channelCols+` FROM source_channels WHERE source_id=$1 ORDER BY handle`, sourceID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []model.SourceChannel
	for rows.Next() {
		var channel model.SourceChannel
		if err := rows.Scan(&channel.ID, &channel.SourceID, &channel.Handle, &channel.Title, &channel.Enabled); err != nil { return nil, err }
		list = append(list, channel)
	}
	return list, rows.Err()
}

func (s *Store) CreateSourceChannel(ctx context.Context, input model.SourceChannel) (model.SourceChannel, error) {
	return scanChannel(s.pool.QueryRow(ctx, `
		INSERT INTO source_channels (source_id, handle, title, enabled)
		VALUES ($1,$2,$3,$4) RETURNING `+channelCols,
		input.SourceID, input.Handle, input.Title, input.Enabled))
}

func (s *Store) UpdateSourceChannel(ctx context.Context, id int32, input model.SourceChannel) (model.SourceChannel, error) {
	return scanChannel(s.pool.QueryRow(ctx, `
		UPDATE source_channels SET handle=$1, title=$2, enabled=$3 WHERE id=$4
		RETURNING `+channelCols, input.Handle, input.Title, input.Enabled, id))
}

func (s *Store) DeleteSourceChannel(ctx context.Context, id int32) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM source_channels WHERE id=$1`, id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return ErrNotFound }
	return nil
}

func (s *Store) UpdateSource(ctx context.Context, id int32, enabled bool) (model.Source, error) {
	return scanSource(s.pool.QueryRow(ctx, `UPDATE sources SET enabled=$1 WHERE id=$2 RETURNING `+sourceCols, enabled, id))
}

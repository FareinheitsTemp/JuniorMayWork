// Пакет store: операції Radar під час ingest — дедуплікація і priority scoring.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// FindDuplicateByFingerprint знаходить уже відомий аналог із будь-якого джерела
// за останні 14 днів. nil означає, що це контентно новий гіг.
func (s *Store) FindDuplicateByFingerprint(ctx context.Context, fingerprint string) (*int64, error) {
	if fingerprint == "" {
		return nil, nil
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM orders
		WHERE content_fingerprint = $1 AND first_seen_at >= now() - interval '14 days'
		ORDER BY first_seen_at DESC LIMIT 1`, fingerprint).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// UpdateOrderRadar зберігає метадані свіжості після звичайного UpsertListing.
func (s *Store) UpdateOrderRadar(
	ctx context.Context,
	id int64,
	publishedAt *time.Time,
	fingerprint string,
	freshness, relevance, priority float64,
	duplicateOf *int64,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE orders SET
			source_published_at = COALESCE($2, source_published_at, first_seen_at),
			content_fingerprint = $3,
			freshness_score = $4,
			relevance_score = $5,
			priority_score = $6,
			duplicate_of_id = CASE WHEN $7 = id THEN NULL ELSE $7 END
		WHERE id = $1`,
		id, publishedAt, fingerprint, freshness, relevance, priority, duplicateOf)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

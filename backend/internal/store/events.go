// Пакет store: події живої історії та заявки на замовлення.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const eventCols = `id, order_id, type, payload, created_at`

func scanEvent(row pgx.Row) (model.Event, error) {
	var e model.Event
	err := row.Scan(&e.ID, &e.OrderID, &e.Type, &e.Payload, &e.CreatedAt)
	return e, err
}

func (s *Store) InsertEvent(ctx context.Context, orderID *int64, typ string, payload model.EventPayload) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO events (order_id, type, payload) VALUES ($1, $2, $3)`,
		orderID, typ, payload.JSON())
	return err
}

func (s *Store) ListEvents(ctx context.Context, limit int) ([]model.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+eventCols+` FROM events
		ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ApplyToOrder: заявка + статус 'applied' + подія, однією транзакцією.
func (s *Store) ApplyToOrder(ctx context.Context, orderID int64, note string) (model.Application, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Application{}, err
	}
	defer tx.Rollback(ctx)

	var a model.Application
	err = tx.QueryRow(ctx, `
		INSERT INTO applications (order_id, order_title, note)
		SELECT id, title, $2 FROM orders WHERE id = $1
		RETURNING id, order_id, order_title, note, result, applied_at`,
		orderID, note,
	).Scan(&a.ID, &a.OrderID, &a.OrderTitle, &a.Note, &a.Result, &a.AppliedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Application{}, ErrNotFound
	}
	if err != nil {
		return model.Application{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'applied' WHERE id = $1`, orderID); err != nil {
		return model.Application{}, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO events (order_id, type, payload) VALUES ($1, 'applied', $2)`,
		orderID, model.EventPayload{OrderID: orderID, Title: a.OrderTitle, Status: "applied"}.JSON()); err != nil {
		return model.Application{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Application{}, err
	}
	return a, nil
}

func (s *Store) ListApplications(ctx context.Context, limit int) ([]model.Application, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, order_title, note, result, applied_at
		FROM applications ORDER BY applied_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Application{}
	for rows.Next() {
		var a model.Application
		if err := rows.Scan(&a.ID, &a.OrderID, &a.OrderTitle, &a.Note, &a.Result, &a.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) SetApplicationResult(ctx context.Context, id int64, result string) (model.Application, error) {
	var a model.Application
	err := s.pool.QueryRow(ctx, `
		UPDATE applications SET result = $2 WHERE id = $1
		RETURNING id, order_id, order_title, note, result, applied_at`, id, result,
	).Scan(&a.ID, &a.OrderID, &a.OrderTitle, &a.Note, &a.Result, &a.AppliedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Application{}, ErrNotFound
	}
	return a, err
}

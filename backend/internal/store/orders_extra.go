package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Orders extra — нотатки, історія статусів і зміна статусу замовлення
// поверх міграції 006 (Фаза A дизайну БД v2).

type OrderStatus struct {
	ID         int    `json:"id"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Color      string `json:"color"`
	Sort       int    `json:"sort"`
	IsTerminal bool   `json:"is_terminal"`
}

type OrderNote struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderStatusChange struct {
	ID         int64     `json:"id"`
	OrderID    int64     `json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Actor      string    `json:"actor"`
	ChangedAt  time.Time `json:"changed_at"`
}

var ErrEmptyNote = errors.New("нотатка порожня")

func (s *Store) ListOrderStatuses(ctx context.Context) ([]OrderStatus, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, key, label, color, sort, is_terminal
		FROM order_statuses ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	statuses := []OrderStatus{}
	for rows.Next() {
		var st OrderStatus
		if err := rows.Scan(&st.ID, &st.Key, &st.Label, &st.Color, &st.Sort, &st.IsTerminal); err != nil {
			return nil, err
		}
		statuses = append(statuses, st)
	}
	return statuses, rows.Err()
}

func (s *Store) ListOrderNotes(ctx context.Context, orderID int64) ([]OrderNote, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, body, created_at
		FROM order_notes WHERE order_id = $1
		ORDER BY created_at DESC LIMIT 200`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := []OrderNote{}
	for rows.Next() {
		var n OrderNote
		if err := rows.Scan(&n.ID, &n.OrderID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) CreateOrderNote(ctx context.Context, orderID int64, body string) (OrderNote, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return OrderNote{}, ErrEmptyNote
	}
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)`, orderID).Scan(&exists); err != nil {
		return OrderNote{}, err
	}
	if !exists {
		return OrderNote{}, pgx.ErrNoRows
	}
	var n OrderNote
	err := s.pool.QueryRow(ctx, `
		INSERT INTO order_notes (order_id, body) VALUES ($1, $2)
		RETURNING id, order_id, body, created_at`, orderID, body).
		Scan(&n.ID, &n.OrderID, &n.Body, &n.CreatedAt)
	return n, err
}

func (s *Store) ListOrderStatusHistory(ctx context.Context, orderID int64) ([]OrderStatusChange, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT h.id, h.order_id,
		       COALESCE(from_s.key, ''), to_s.key,
		       COALESCE(h.actor, ''), h.changed_at
		FROM order_status_history h
		JOIN order_statuses to_s ON to_s.id = h.to_status_id
		LEFT JOIN order_statuses from_s ON from_s.id = h.from_status_id
		WHERE h.order_id = $1
		ORDER BY h.changed_at DESC LIMIT 100`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	changes := []OrderStatusChange{}
	for rows.Next() {
		var c OrderStatusChange
		if err := rows.Scan(&c.ID, &c.OrderID, &c.FromStatus, &c.ToStatus, &c.Actor, &c.ChangedAt); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

// ChangeOrderStatus атомарно змінює статус замовлення: у транзакції фіксує
// рядок, пише order_status_history, синхронізує status_id і легасі status,
// і залишає слід в audit_log.
func (s *Store) ChangeOrderStatus(ctx context.Context, orderID int64, statusKey, actor string) (OrderStatus, error) {
	var target OrderStatus
	err := s.pool.QueryRow(ctx, `
		SELECT id, key, label, color, sort, is_terminal
		FROM order_statuses WHERE key = $1`, statusKey).
		Scan(&target.ID, &target.Key, &target.Label, &target.Color, &target.Sort, &target.IsTerminal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderStatus{}, ErrNotFound
		}
		return OrderStatus{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OrderStatus{}, err
	}
	defer tx.Rollback(ctx)

	var currentID *int
	if err := tx.QueryRow(ctx,
		`SELECT status_id FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&currentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderStatus{}, ErrNotFound
		}
		return OrderStatus{}, err
	}
	if currentID != nil && *currentID == target.ID {
		return target, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (order_id, from_status_id, to_status_id, actor)
		VALUES ($1, $2, $3, NULLIF($4, ''))`,
		orderID, currentID, target.ID, actor); err != nil {
		return OrderStatus{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status_id = $2, status = $3 WHERE id = $1`,
		orderID, target.ID, target.Key); err != nil {
		return OrderStatus{}, err
	}

	payload, err := json.Marshal(map[string]any{
		"from": currentID,
		"to":   target.ID,
		"key":  target.Key,
	})
	if err != nil {
		return OrderStatus{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_log (entity, entity_id, action, actor, payload)
		VALUES ('order', $1, 'status_change', NULLIF($2, ''), $3::jsonb)`,
		orderID, actor, payload); err != nil {
		return OrderStatus{}, err
	}

	return target, tx.Commit(ctx)
}

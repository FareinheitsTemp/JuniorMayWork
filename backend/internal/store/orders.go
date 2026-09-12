// Пакет store: замовлення — фільтри, upsert з джерел, stale-кандидати в архів.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const orderCols = `id, source, external_id, url, title, description, budget_cents, currency,
	skills, branch_id, status, first_seen_at, last_seen_at, raw`

func scanOrder(row pgx.Row) (model.Order, error) {
	var o model.Order
	err := row.Scan(&o.ID, &o.Source, &o.ExternalID, &o.URL, &o.Title, &o.Description,
		&o.BudgetCents, &o.Currency, &o.Skills, &o.BranchID, &o.Status,
		&o.FirstSeenAt, &o.LastSeenAt, &o.Raw)
	return o, err
}

type OrderFilter struct {
	Status   string
	BranchID *int64
	Query    string
	From     *time.Time
	To       *time.Time
	Limit    int
}

func (s *Store) ListOrders(ctx context.Context, f OrderFilter) ([]model.Order, error) {
	q := `SELECT ` + orderCols + ` FROM orders WHERE 1=1`
	args := []any{}
	if f.Status != "" {
		args = append(args, f.Status)
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if f.BranchID != nil {
		args = append(args, *f.BranchID)
		q += fmt.Sprintf(" AND branch_id = $%d", len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		q += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", len(args), len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		q += fmt.Sprintf(" AND first_seen_at >= $%d", len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		q += fmt.Sprintf(" AND first_seen_at < $%d", len(args))
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q += fmt.Sprintf(" ORDER BY first_seen_at DESC LIMIT %d", limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id int64) (model.Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx,
		`SELECT `+orderCols+` FROM orders WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, ErrNotFound
	}
	return o, err
}

// UpsertListing вставляє або оновлює замовлення з джерела.
// inserted = true означає новий листочок (щойно зірваний).
func (s *Store) UpsertListing(ctx context.Context, l model.Listing) (model.Order, bool, error) {
	skills := l.Skills
	if skills == nil {
		skills = []string{}
	}
	currency := l.Currency
	if currency == "" {
		currency = "USD"
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO orders (source, external_id, url, title, description, budget_cents, currency, skills, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (source, external_id) DO UPDATE SET
			url = EXCLUDED.url,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			budget_cents = EXCLUDED.budget_cents,
			currency = EXCLUDED.currency,
			skills = EXCLUDED.skills,
			raw = EXCLUDED.raw,
			last_seen_at = now()
		RETURNING `+orderCols+`, (xmax = 0)`,
		l.Source, l.ExternalID, l.URL, l.Title, l.Description, l.BudgetCents, currency, skills, l.Raw)
	var o model.Order
	var inserted bool
	err := row.Scan(&o.ID, &o.Source, &o.ExternalID, &o.URL, &o.Title, &o.Description,
		&o.BudgetCents, &o.Currency, &o.Skills, &o.BranchID, &o.Status,
		&o.FirstSeenAt, &o.LastSeenAt, &o.Raw, &inserted)
	return o, inserted, err
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id int64, status string) (model.Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, `
		UPDATE orders SET status = $2 WHERE id = $1
		RETURNING `+orderCols, id, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, ErrNotFound
	}
	return o, err
}

func (s *Store) UpdateOrder(ctx context.Context, id int64, o model.Order) (model.Order, error) {
	skills := o.Skills
	if skills == nil {
		skills = []string{}
	}
	updated, err := scanOrder(s.pool.QueryRow(ctx, `
		UPDATE orders SET
			title = $2, description = $3, url = $4, budget_cents = $5,
			skills = $6, branch_id = $7, status = $8
		WHERE id = $1
		RETURNING `+orderCols,
		id, o.Title, o.Description, o.URL, o.BudgetCents, skills, o.BranchID, o.Status))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Order{}, ErrNotFound
	}
	return updated, err
}

func (s *Store) SetOrderBranch(ctx context.Context, id int64, branchID *int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE orders SET branch_id = $2 WHERE id = $1`, id, branchID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteOrder(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// StaleOrders: замовлення джерела, яких не було останні вибірки — кандидати
// в архів. Виграні ('won') не чіпаємо.
func (s *Store) StaleOrders(ctx context.Context, source string, olderThan time.Time) ([]model.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderCols+` FROM orders
		WHERE source = $1 AND last_seen_at < $2 AND status <> 'won'
		ORDER BY first_seen_at`, source, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

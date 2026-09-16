// Пакет store: замовлення v4 — фільтри, upsert з джерел, архівація зниклих.
// Схема — docs/DB_DESIGN_V4.md (міграції 010–011).
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const orderSelect = `
	SELECT o.id, o.source_id, s.key, o.source_message_id,
	       o.title, o.description, o.status, o.budget_cents,
	       o.external_url, o.published_at, o.first_seen_at, o.last_seen_at,
	       COALESCE((SELECT array_agg(sk.name ORDER BY sk.name)
	          FROM order_skills os2 JOIN skills sk ON sk.id = os2.skill_id
	         WHERE os2.order_id = o.id), ARRAY[]::text[])
	  FROM orders o
	  JOIN sources s ON s.id = o.source_id`

func scanOrder(row pgx.Row) (model.Order, error) {
	var o model.Order
	err := row.Scan(&o.ID, &o.SourceID, &o.SourceKey, &o.SourceMessageID,
		&o.Title, &o.Description, &o.Status, &o.BudgetCents,
		&o.ExternalURL, &o.PublishedAt, &o.FirstSeenAt, &o.LastSeenAt,
		&o.Skills)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

// UpsertListing вставляє або оновлює замовлення з джерела.
// inserted = true означає нове замовлення (щойно зірване).
func (s *Store) UpsertListing(ctx context.Context, l model.Listing) (model.Order, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Order{}, false, err
	}
	defer tx.Rollback(ctx)

	var sourceID int32
	if err := tx.QueryRow(ctx,
		`SELECT id FROM sources WHERE key = $1`, l.SourceKey).Scan(&sourceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Order{}, false, fmt.Errorf("джерело %q не знайдено в sources", l.SourceKey)
		}
		return model.Order{}, false, err
	}

	var msgID *string
	if l.SourceMessageID != "" {
		msgID = &l.SourceMessageID
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO orders (source_id, source_message_id, title, description,
		                    budget_cents, external_url, published_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (source_id, source_message_id) DO UPDATE SET
			title        = EXCLUDED.title,
			description  = EXCLUDED.description,
			budget_cents = EXCLUDED.budget_cents,
			external_url = EXCLUDED.external_url,
			published_at = EXCLUDED.published_at,
			last_seen_at = now()
		RETURNING id, (xmax = 0) AS inserted`,
		sourceID, msgID, l.Title, l.Description, l.BudgetCents, l.URL, l.PublishedAt, model.StatusNew)

	var orderID int64
	var inserted bool
	if err := row.Scan(&orderID, &inserted); err != nil {
		return model.Order{}, false, err
	}

	if err := syncOrderSkills(ctx, tx, orderID, l.Skills); err != nil {
		return model.Order{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Order{}, false, err
	}

	o, err := s.GetOrder(ctx, orderID)
	return o, inserted, err
}

// syncOrderSkills синхронізує M:N зв'язки order_skills: додає нові навички,
// прибирає зниклі. Навички з довідника skills створює за потреби.
func syncOrderSkills(ctx context.Context, tx pgx.Tx, orderID int64, names []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM order_skills WHERE order_id = $1`, orderID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var skillID int64
		err := tx.QueryRow(ctx, `
			INSERT INTO skills (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, name).Scan(&skillID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_skills (order_id, skill_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, orderID, skillID); err != nil {
			return err
		}
	}
	return nil
}

// ListOrders — список замовлень з фільтрами v4.
func (s *Store) ListOrders(ctx context.Context, f model.OrderFilter) ([]model.Order, error) {
	q := orderSelect + ` WHERE 1=1`
	args := []any{}
	n := func() string { return fmt.Sprintf("$%d", len(args)+1) }

	if f.Status != "" {
		q += " AND o.status = " + n()
		args = append(args, f.Status)
	}
	if f.Skill != "" {
		q += " AND EXISTS (SELECT 1 FROM order_skills os JOIN skills sk ON sk.id = os.skill_id " +
			"WHERE os.order_id = o.id AND lower(sk.name) = lower(" + n() + "))"
		args = append(args, f.Skill)
	}
	if f.MaxBudgetCents != nil {
		q += " AND o.budget_cents <= " + n()
		args = append(args, *f.MaxBudgetCents)
	}
	if f.DateFrom != nil {
		q += " AND o.first_seen_at >= " + n()
		args = append(args, *f.DateFrom)
	}
	if f.DateTo != nil {
		q += " AND o.first_seen_at < " + n()
		args = append(args, *f.DateTo)
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q += " AND (o.title ILIKE " + n() + " OR o.description ILIKE " + n() + ")"
		args = append(args, like, like)
	}

	q += " ORDER BY o.first_seen_at DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.SourceID, &o.SourceKey, &o.SourceMessageID,
			&o.Title, &o.Description, &o.Status, &o.BudgetCents,
			&o.ExternalURL, &o.PublishedAt, &o.FirstSeenAt, &o.LastSeenAt, &o.Skills); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// GetOrder — одне замовлення за id.
func (s *Store) GetOrder(ctx context.Context, id int64) (model.Order, error) {
	return scanOrder(s.pool.QueryRow(ctx, orderSelect+` WHERE o.id = $1`, id))
}

// SetOrderStatus — зміна статусу з перевіркою допустимих значень (CHECK у БД теж є).
func (s *Store) SetOrderStatus(ctx context.Context, id int64, status string) (model.Order, error) {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return model.Order{}, err
	}
	return s.GetOrder(ctx, id)
}

// PatchOrder — точкова правка title/description/status/budget_cents.
func (s *Store) PatchOrder(ctx context.Context, id int64, patch model.OrderPatch) (model.Order, error) {
	q := `UPDATE orders SET`
	args := []any{}
	if patch.Title != nil {
		args = append(args, *patch.Title)
		q += fmt.Sprintf(" title = $%d,", len(args))
	}
	if patch.Description != nil {
		args = append(args, *patch.Description)
		q += fmt.Sprintf(" description = $%d,", len(args))
	}
	if patch.Status != nil {
		args = append(args, *patch.Status)
		q += fmt.Sprintf(" status = $%d,", len(args))
	}
	if patch.BudgetCents != nil {
		args = append(args, *patch.BudgetCents)
		q += fmt.Sprintf(" budget_cents = $%d,", len(args))
	}
	if len(args) == 0 {
		return s.GetOrder(ctx, id)
	}
	q = strings.TrimSuffix(q, ",") + fmt.Sprintf(" WHERE id = $%d", len(args)+1)
	args = append(args, id)
	if _, err := s.pool.Exec(ctx, q, args...); err != nil {
		return model.Order{}, err
	}
	return s.GetOrder(ctx, id)
}

// StaleOrders — замовлення джерела, яких не було з before: кандидати в архів.
// Виграні ('won') не чіпаємо.
func (s *Store) StaleOrders(ctx context.Context, sourceKey string, before time.Time) ([]model.Order, error) {
	rows, err := s.pool.Query(ctx, orderSelect+`
		WHERE s.key = $1 AND o.last_seen_at < $2 AND o.status <> $3
		ORDER BY o.first_seen_at DESC`, sourceKey, before, model.StatusWon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.SourceID, &o.SourceKey, &o.SourceMessageID,
			&o.Title, &o.Description, &o.Status, &o.BudgetCents,
			&o.ExternalURL, &o.PublishedAt, &o.FirstSeenAt, &o.LastSeenAt, &o.Skills); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// ArchiveOrders — переносить зниклі замовлення в archived_orders і
// видаляє з orders. Повертає кількість заархівованих.
// reason: 'removed' (зникло з джерела) | 'manual' (видалив користувач).
func (s *Store) ArchiveOrders(ctx context.Context, ids []int64, reason string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var archived int
	for _, id := range ids {
		tag, err := tx.Exec(ctx, `
			WITH deleted AS (
				DELETE FROM orders WHERE id = $1
				RETURNING id, source_id, title, description, status, budget_cents,
				          external_url, published_at, first_seen_at, last_seen_at
			)
			INSERT INTO archived_orders (id, source_key, title, status, budget_cents,
			                            external_url, snapshot, reason)
			SELECT d.id, s.key, d.title, d.status, d.budget_cents, d.external_url,
			       to_jsonb(d) || jsonb_build_object('source_key', s.key), $2
			  FROM deleted d JOIN sources s ON s.id = d.source_id`, id, reason)
		if err != nil {
			return archived, err
		}
		archived += int(tag.RowsAffected())
	}
	return archived, tx.Commit(ctx)
}

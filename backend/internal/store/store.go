// Пакет store: усі SQL-запити до PostgreSQL (тільки параметризовані).
// store.go — база і CRUD віток.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const branchCols = `id, name, keywords, max_budget_cents, is_active, created_at`

func scanBranch(row pgx.Row) (model.Branch, error) {
	var b model.Branch
	err := row.Scan(&b.ID, &b.Name, &b.Keywords, &b.MaxBudgetCents, &b.IsActive, &b.CreatedAt)
	return b, err
}

func (s *Store) ListBranches(ctx context.Context, activeOnly bool) ([]model.Branch, error) {
	q := `SELECT ` + branchCols + ` FROM branches`
	if activeOnly {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Branch{}
	for rows.Next() {
		b, err := scanBranch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) GetBranch(ctx context.Context, id int64) (model.Branch, error) {
	b, err := scanBranch(s.pool.QueryRow(ctx,
		`SELECT `+branchCols+` FROM branches WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Branch{}, ErrNotFound
	}
	return b, err
}

func (s *Store) CreateBranch(ctx context.Context, b model.Branch) (model.Branch, error) {
	if b.MaxBudgetCents <= 0 {
		b.MaxBudgetCents = 10000
	}
	keywords := b.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	created, err := scanBranch(s.pool.QueryRow(ctx, `
		INSERT INTO branches (name, keywords, max_budget_cents, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING `+branchCols,
		b.Name, keywords, b.MaxBudgetCents, b.IsActive))
	return created, err
}

func (s *Store) UpdateBranch(ctx context.Context, id int64, b model.Branch) (model.Branch, error) {
	if b.MaxBudgetCents <= 0 {
		b.MaxBudgetCents = 10000
	}
	keywords := b.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	updated, err := scanBranch(s.pool.QueryRow(ctx, `
		UPDATE branches SET
			name = $2, keywords = $3, max_budget_cents = $4, is_active = $5
		WHERE id = $1
		RETURNING `+branchCols,
		id, b.Name, keywords, b.MaxBudgetCents, b.IsActive))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Branch{}, ErrNotFound
	}
	return updated, err
}

func (s *Store) DeleteBranch(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM branches WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

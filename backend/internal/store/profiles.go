package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

const profileSelect = `
	SELECT p.id, p.name, p.max_budget_cents, p.date_from, p.date_to,
	       p.results_limit, p.is_active, p.created_at,
	       COALESCE((SELECT array_agg(sk.name ORDER BY sk.name)
	          FROM profile_skills ps JOIN skills sk ON sk.id = ps.skill_id
	         WHERE ps.profile_id = p.id), ARRAY[]::text[]),
	       COALESCE((SELECT array_agg(ps2.source_id ORDER BY ps2.source_id)
	          FROM profile_sources ps2 WHERE ps2.profile_id = p.id), ARRAY[]::integer[])
	  FROM profiles p`

func scanProfile(row pgx.Row) (model.Profile, error) {
	var p model.Profile
	if err := row.Scan(&p.ID, &p.Name, &p.MaxBudgetCents, &p.DateFrom, &p.DateTo,
		&p.ResultsLimit, &p.IsActive, &p.CreatedAt, &p.Skills, &p.SourceIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return p, ErrNotFound }
		return p, err
	}
	return p, nil
}

func (s *Store) ListProfiles(ctx context.Context) ([]model.Profile, error) {
	rows, err := s.pool.Query(ctx, profileSelect+` ORDER BY p.is_active DESC, p.created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []model.Profile
	for rows.Next() {
		var p model.Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.MaxBudgetCents, &p.DateFrom, &p.DateTo, &p.ResultsLimit, &p.IsActive, &p.CreatedAt, &p.Skills, &p.SourceIDs); err != nil { return nil, err }
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *Store) GetProfile(ctx context.Context, id int64) (model.Profile, error) {
	return scanProfile(s.pool.QueryRow(ctx, profileSelect+` WHERE p.id = $1`, id))
}

func (s *Store) CreateProfile(ctx context.Context, input model.Profile) (model.Profile, error) {
	if err := validateProfile(input); err != nil { return model.Profile{}, err }
	tx, err := s.pool.Begin(ctx)
	if err != nil { return model.Profile{}, err }
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO profiles (name, max_budget_cents, date_from, date_to, results_limit, is_active)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, input.Name, input.MaxBudgetCents, input.DateFrom, input.DateTo, input.ResultsLimit, input.IsActive).Scan(&id)
	if err != nil { return model.Profile{}, err }
	if err := syncProfileSkills(ctx, tx, id, input.Skills); err != nil { return model.Profile{}, err }
	if err := syncProfileSources(ctx, tx, id, input.SourceIDs); err != nil { return model.Profile{}, err }
	if err := tx.Commit(ctx); err != nil { return model.Profile{}, err }
	return s.GetProfile(ctx, id)
}

func (s *Store) UpdateProfile(ctx context.Context, id int64, input model.Profile) (model.Profile, error) {
	if err := validateProfile(input); err != nil { return model.Profile{}, err }
	tx, err := s.pool.Begin(ctx)
	if err != nil { return model.Profile{}, err }
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE profiles SET name=$1, max_budget_cents=$2, date_from=$3, date_to=$4, results_limit=$5, is_active=$6 WHERE id=$7`, input.Name, input.MaxBudgetCents, input.DateFrom, input.DateTo, input.ResultsLimit, input.IsActive, id)
	if err != nil { return model.Profile{}, err }
	if tag.RowsAffected() == 0 { return model.Profile{}, ErrNotFound }
	if err := syncProfileSkills(ctx, tx, id, input.Skills); err != nil { return model.Profile{}, err }
	if err := syncProfileSources(ctx, tx, id, input.SourceIDs); err != nil { return model.Profile{}, err }
	if err := tx.Commit(ctx); err != nil { return model.Profile{}, err }
	return s.GetProfile(ctx, id)
}

func (s *Store) DeleteProfile(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM profiles WHERE id=$1`, id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return ErrNotFound }
	return nil
}

func validateProfile(p model.Profile) error {
	if strings.TrimSpace(p.Name) == "" { return errors.New("назва профілю обов'язкова") }
	if p.ResultsLimit < 1 || p.ResultsLimit > 1000 { return errors.New("results_limit має бути від 1 до 1000") }
	if p.MaxBudgetCents != nil && *p.MaxBudgetCents < 0 { return errors.New("бюджет не може бути від'ємним") }
	if p.DateFrom != nil && p.DateTo != nil && p.DateFrom.After(*p.DateTo) { return errors.New("date_from не може бути після date_to") }
	return nil
}

func syncProfileSkills(ctx context.Context, tx pgx.Tx, profileID int64, names []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM profile_skills WHERE profile_id=$1`, profileID); err != nil { return err }
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] { continue }
		seen[name] = true
		var skillID int64
		if err := tx.QueryRow(ctx, `INSERT INTO skills (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, name).Scan(&skillID); err != nil { return err }
		if _, err := tx.Exec(ctx, `INSERT INTO profile_skills (profile_id, skill_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, profileID, skillID); err != nil { return err }
	}
	return nil
}

func syncProfileSources(ctx context.Context, tx pgx.Tx, profileID int64, sourceIDs []int32) error {
	if _, err := tx.Exec(ctx, `DELETE FROM profile_sources WHERE profile_id=$1`, profileID); err != nil { return err }
	for _, sourceID := range sourceIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO profile_sources (profile_id, source_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, profileID, sourceID); err != nil { return err }
	}
	return nil
}

func (s *Store) StartScrapeRun(ctx context.Context, profileID int64) (model.ScrapeRun, error) {
	var run model.ScrapeRun
	err := s.pool.QueryRow(ctx, `INSERT INTO scrape_runs (profile_id, outcome) VALUES ($1,'ok') RETURNING id, profile_id, started_at, finished_at, outcome, error_text, stats`, profileID).
		Scan(&run.ID, &run.ProfileID, &run.StartedAt, &run.FinishedAt, &run.Outcome, &run.ErrorText, &run.Stats)
	return run, err
}

func (s *Store) FinishScrapeRun(ctx context.Context, id int64, outcome string, errText string, stats model.RunStats) error {
	payload, err := json.Marshal(stats)
	if err != nil { return err }
	tag, err := s.pool.Exec(ctx, `UPDATE scrape_runs SET finished_at=now(), outcome=$1, error_text=$2, stats=$3 WHERE id=$4`, outcome, errText, string(payload), id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return ErrNotFound }
	return nil
}

func (s *Store) RecordOrderMatch(ctx context.Context, runID, orderID int64, score int) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO order_matches (run_id, order_id, relevance_score) VALUES ($1,$2,$3) ON CONFLICT (run_id,order_id) DO UPDATE SET relevance_score=EXCLUDED.relevance_score, captured_at=now()`, runID, orderID, score)
	return err
}

func (s *Store) ListProfileRuns(ctx context.Context, profileID int64) ([]model.ScrapeRun, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, profile_id, started_at, finished_at, outcome, error_text, stats FROM scrape_runs WHERE profile_id=$1 ORDER BY started_at DESC`, profileID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []model.ScrapeRun
	for rows.Next() {
		var r model.ScrapeRun
		if err := rows.Scan(&r.ID, &r.ProfileID, &r.StartedAt, &r.FinishedAt, &r.Outcome, &r.ErrorText, &r.Stats); err != nil { return nil, err }
		list = append(list, r)
	}
	return list, rows.Err()
}

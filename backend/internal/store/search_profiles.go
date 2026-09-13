package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const searchProfileCols = `id, name, is_active, interval_minutes, min_budget_cents, max_budget_cents, max_age_hours, created_at, updated_at`

func scanSearchProfile(row pgx.Row) (model.SearchProfile, error) {
	var p model.SearchProfile
	err := row.Scan(
		&p.ID, &p.Name, &p.IsActive, &p.IntervalMinutes,
		&p.MinBudgetCents, &p.MaxBudgetCents, &p.MaxAgeHours,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (s *Store) ListSearchProfiles(ctx context.Context) ([]model.SearchProfile, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+searchProfileCols+` FROM search_profiles ORDER BY is_active DESC, updated_at DESC, id DESC`)
	if err != nil { return nil, err }
	defer rows.Close()

	profiles := make([]model.SearchProfile, 0)
	for rows.Next() {
		p, err := scanSearchProfile(rows)
		if err != nil { return nil, err }
		if err := s.loadSearchProfileChildren(ctx, &p); err != nil { return nil, err }
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *Store) GetSearchProfile(ctx context.Context, id int64) (model.SearchProfile, error) {
	p, err := scanSearchProfile(s.pool.QueryRow(ctx, `SELECT `+searchProfileCols+` FROM search_profiles WHERE id = $1`, id))
	if err != nil { return model.SearchProfile{}, err }
	if err := s.loadSearchProfileChildren(ctx, &p); err != nil { return model.SearchProfile{}, err }
	return p, nil
}

func (s *Store) CreateSearchProfile(ctx context.Context, input model.SearchProfile) (model.SearchProfile, error) {
	if err := validateSearchProfile(input); err != nil { return model.SearchProfile{}, err }
	tx, err := s.pool.Begin(ctx)
	if err != nil { return model.SearchProfile{}, err }
	defer tx.Rollback(ctx)

	p, err := scanSearchProfile(tx.QueryRow(ctx, `
		INSERT INTO search_profiles (name, is_active, interval_minutes, min_budget_cents, max_budget_cents, max_age_hours)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+searchProfileCols,
		input.Name, input.IsActive, input.IntervalMinutes, input.MinBudgetCents, input.MaxBudgetCents, input.MaxAgeHours,
	))
	if err != nil { return model.SearchProfile{}, err }
	if err := replaceSearchProfileChildren(ctx, tx, p.ID, input.Tags, input.Sources); err != nil { return model.SearchProfile{}, err }
	if err := tx.Commit(ctx); err != nil { return model.SearchProfile{}, err }
	return s.GetSearchProfile(ctx, p.ID)
}

func (s *Store) UpdateSearchProfile(ctx context.Context, id int64, input model.SearchProfile) (model.SearchProfile, error) {
	if err := validateSearchProfile(input); err != nil { return model.SearchProfile{}, err }
	tx, err := s.pool.Begin(ctx)
	if err != nil { return model.SearchProfile{}, err }
	defer tx.Rollback(ctx)

	_, err = scanSearchProfile(tx.QueryRow(ctx, `
		UPDATE search_profiles
		SET name = $2, is_active = $3, interval_minutes = $4, min_budget_cents = $5, max_budget_cents = $6,
			max_age_hours = $7, updated_at = now()
		WHERE id = $1
		RETURNING `+searchProfileCols,
		id, input.Name, input.IsActive, input.IntervalMinutes, input.MinBudgetCents, input.MaxBudgetCents, input.MaxAgeHours,
	))
	if err != nil { return model.SearchProfile{}, err }
	if err := replaceSearchProfileChildren(ctx, tx, id, input.Tags, input.Sources); err != nil { return model.SearchProfile{}, err }
	if err := tx.Commit(ctx); err != nil { return model.SearchProfile{}, err }
	return s.GetSearchProfile(ctx, id)
}

func (s *Store) DeleteSearchProfile(ctx context.Context, id int64) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM search_profiles WHERE id = $1`, id)
	if err != nil { return err }
	if cmd.RowsAffected() == 0 { return pgx.ErrNoRows }
	return nil
}

func (s *Store) loadSearchProfileChildren(ctx context.Context, p *model.SearchProfile) error {
	tags, err := s.pool.Query(ctx, `SELECT id, profile_id, tag, tag_type, created_at FROM search_profile_tags WHERE profile_id = $1 ORDER BY tag_type, tag`, p.ID)
	if err != nil { return err }
	defer tags.Close()
	p.Tags = make([]model.SearchProfileTag, 0)
	for tags.Next() {
		var tag model.SearchProfileTag
		if err := tags.Scan(&tag.ID, &tag.ProfileID, &tag.Tag, &tag.TagType, &tag.CreatedAt); err != nil { return err }
		p.Tags = append(p.Tags, tag)
	}
	if err := tags.Err(); err != nil { return err }

	sources, err := s.pool.Query(ctx, `SELECT profile_id, source_id, is_enabled FROM search_profile_sources WHERE profile_id = $1 ORDER BY source_id`, p.ID)
	if err != nil { return err }
	defer sources.Close()
	p.Sources = make([]model.SearchProfileSource, 0)
	for sources.Next() {
		var source model.SearchProfileSource
		if err := sources.Scan(&source.ProfileID, &source.SourceID, &source.IsEnabled); err != nil { return err }
		p.Sources = append(p.Sources, source)
	}
	return sources.Err()
}

func replaceSearchProfileChildren(ctx context.Context, tx pgx.Tx, profileID int64, tags []model.SearchProfileTag, sources []model.SearchProfileSource) error {
	if _, err := tx.Exec(ctx, `DELETE FROM search_profile_tags WHERE profile_id = $1`, profileID); err != nil { return err }
	if _, err := tx.Exec(ctx, `DELETE FROM search_profile_sources WHERE profile_id = $1`, profileID); err != nil { return err }
	for _, tag := range tags {
		if _, err := tx.Exec(ctx, `INSERT INTO search_profile_tags (profile_id, tag, tag_type) VALUES ($1, $2, $3)`, profileID, strings.TrimSpace(tag.Tag), tag.TagType); err != nil { return err }
	}
	for _, source := range sources {
		if _, err := tx.Exec(ctx, `INSERT INTO search_profile_sources (profile_id, source_id, is_enabled) VALUES ($1, $2, $3)`, profileID, source.SourceID, source.IsEnabled); err != nil { return err }
	}
	return nil
}

func validateSearchProfile(p model.SearchProfile) error {
	if strings.TrimSpace(p.Name) == "" { return errors.New("profile name is required") }
	if p.IntervalMinutes < 5 || p.IntervalMinutes > 1440 { return errors.New("interval_minutes must be between 5 and 1440") }
	if p.MinBudgetCents != nil && *p.MinBudgetCents < 0 { return errors.New("min_budget_cents cannot be negative") }
	if p.MaxBudgetCents != nil && *p.MaxBudgetCents < 0 { return errors.New("max_budget_cents cannot be negative") }
	if p.MinBudgetCents != nil && p.MaxBudgetCents != nil && *p.MaxBudgetCents < *p.MinBudgetCents { return errors.New("max_budget_cents must be at least min_budget_cents") }
	if p.MaxAgeHours != nil && (*p.MaxAgeHours < 1 || *p.MaxAgeHours > 720) { return errors.New("max_age_hours must be between 1 and 720") }
	for _, tag := range p.Tags {
		if strings.TrimSpace(tag.Tag) == "" { return errors.New("tag cannot be blank") }
		switch tag.TagType {
		case "role", "skill", "language", "technology":
		default: return errors.New("invalid tag_type")
		}
	}
	for _, source := range p.Sources {
		if source.SourceID <= 0 { return errors.New("source_id must be positive") }
	}
	return nil
}

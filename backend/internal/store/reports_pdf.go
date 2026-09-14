package store

import (
	"context"
	"time"
)

type PDFReportJob struct {
	Title       string
	Source      string
	URL         string
	BudgetCents int64
	Currency    string
	Priority    int
	Freshness   int
}

type PDFRunReport struct {
	RunID       int64
	ProfileName string
	Status      string
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Jobs        []PDFReportJob
}

func (s *Store) GetPDFRunReport(ctx context.Context, runID int64) (PDFRunReport, error) {
	var report PDFRunReport
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, COALESCE(p.name, ''), r.status, r.started_at, r.finished_at
		FROM search_runs r
		LEFT JOIN search_profiles p ON p.id = r.profile_id
		WHERE r.id = $1`, runID).
		Scan(&report.RunID, &report.ProfileName, &report.Status, &report.StartedAt, &report.FinishedAt)
	if err != nil {
		return PDFRunReport{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT COALESCE(o.title, ''), COALESCE(o.source, ''), COALESCE(o.url, ''),
		       COALESCE(o.budget_cents, 0), COALESCE(o.currency, ''),
		       COALESCE(o.priority_score, 0), COALESCE(o.freshness_score, 0)
		FROM order_discoveries d
		JOIN orders o ON o.id = d.order_id
		WHERE d.search_run_id = $1
		ORDER BY o.priority_score DESC NULLS LAST
		LIMIT 200`, runID)
	if err != nil {
		return PDFRunReport{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var job PDFReportJob
		if err := rows.Scan(&job.Title, &job.Source, &job.URL, &job.BudgetCents, &job.Currency, &job.Priority, &job.Freshness); err != nil {
			return PDFRunReport{}, err
		}
		report.Jobs = append(report.Jobs, job)
	}
	return report, rows.Err()
}

func (s *Store) InsertReport(ctx context.Context, runID int64, fileName, storageKey string, ordersTotal int, summaryJSON string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reports (search_run_id, file_name, storage_key, mime_type, orders_total, summary_json)
		VALUES ($1, $2, $3, 'application/pdf', $4, $5::jsonb)
		RETURNING id`, runID, fileName, storageKey, ordersTotal, summaryJSON).Scan(&id)
	return id, err
}

func (s *Store) GetReport(ctx context.Context, id int64) (storageKey string, fileName string, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT storage_key, file_name FROM reports WHERE id = $1`, id).
		Scan(&storageKey, &fileName)
	return storageKey, fileName, err
}

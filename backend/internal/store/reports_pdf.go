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
	StatusLabel string
}

type PDFSourceStat struct {
	Source string
	Count  int
	Share  float64
}

type PDFDailyPoint struct {
	Day   string
	Count int
}

type PDFStatusStat struct {
	Label string
	Count int
}

type PDFBudgetStats struct {
	Min        int64
	Median     int64
	Avg        int64
	Max        int64
	WithBudget int
}

type PDFSkillStat struct {
	Skill string
	Count int
}

// PDFRunReport — повний набір даних для детального аналітичного PDF-звіту:
// замовлення прогона + агрегати за джерелами, днями, статусами, бюджетом,
// навичками, заявками та дублікатами.
type PDFRunReport struct {
	RunID       int64
	ProfileName string
	Status      string
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Jobs        []PDFReportJob
	Sources     []PDFSourceStat
	Daily       []PDFDailyPoint
	Statuses    []PDFStatusStat
	Budget      PDFBudgetStats
	Skills      []PDFSkillStat
	Applications int
	Duplicates   int
}

const pdfJoin = `
	FROM order_discoveries d
	JOIN orders o ON o.id = d.order_id
	WHERE d.search_run_id = $1`

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
		       COALESCE(o.priority_score, 0), COALESCE(o.freshness_score, 0),
		       COALESCE(st.label, '')
		FROM order_discoveries d
		JOIN orders o ON o.id = d.order_id
		LEFT JOIN order_statuses st ON st.id = o.status_id
		WHERE d.search_run_id = $1
		ORDER BY o.priority_score DESC NULLS LAST
		LIMIT 200`, runID)
	if err != nil {
		return PDFRunReport{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var job PDFReportJob
		if err := rows.Scan(&job.Title, &job.Source, &job.URL, &job.BudgetCents, &job.Currency, &job.Priority, &job.Freshness, &job.StatusLabel); err != nil {
			return PDFRunReport{}, err
		}
		report.Jobs = append(report.Jobs, job)
	}
	if err := rows.Err(); err != nil {
		return PDFRunReport{}, err
	}

	// Розподіл за джерелами.
	total := len(report.Jobs)
	sourceCounts := []PDFSourceStat{}
	if err := s.scanPairs(ctx, `SELECT o.source, count(*)::int`+pdfJoin+`
		GROUP BY o.source ORDER BY count(*) DESC`, runID, func(name string, count int) {
		share := 0.0
		if total > 0 {
			share = float64(count) / float64(total) * 100
		}
		sourceCounts = append(sourceCounts, PDFSourceStat{Source: name, Count: count, Share: share})
	}); err != nil {
		return PDFRunReport{}, err
	}
	report.Sources = sourceCounts

	// Дублікати в прогоні.
	if err := s.pool.QueryRow(ctx, `SELECT count(*)::int FROM order_discoveries d
		JOIN orders o ON o.id = d.order_id WHERE o.duplicate_of_id IS NOT NULL AND d.search_run_id = $1`, runID).
		Scan(&report.Duplicates); err != nil {
		return PDFRunReport{}, err
	}

	// Динаміка надходжень: останні 30 днів (v_daily_intake, міграція 006).
	daily := []PDFDailyPoint{}
	dRows, err := s.pool.Query(ctx, `SELECT day::text, orders_total FROM v_daily_intake ORDER BY day DESC LIMIT 30`)
	if err != nil {
		return PDFRunReport{}, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var p PDFDailyPoint
		if err := dRows.Scan(&p.Day, &p.Count); err != nil {
			return PDFRunReport{}, err
		}
		daily = append(daily, p)
	}
	if err := dRows.Err(); err != nil {
		return PDFRunReport{}, err
	}
	for i, j := 0, len(daily)-1; i < j; i, j = i+1, j-1 {
		daily[i], daily[j] = daily[j], daily[i]
	}
	report.Daily = daily

	// Розподіл за статусами.
	statusStats := []PDFStatusStat{}
	if err := s.scanPairs(ctx, `SELECT COALESCE(st.label, 'Без статусу'), count(*)::int
		FROM order_discoveries d
		JOIN orders o ON o.id = d.order_id
		LEFT JOIN order_statuses st ON st.id = o.status_id
		WHERE d.search_run_id = $1
		GROUP BY 1 ORDER BY 2 DESC`, runID, func(name string, count int) {
		statusStats = append(statusStats, PDFStatusStat{Label: name, Count: count})
	}); err != nil {
		return PDFRunReport{}, err
	}
	report.Statuses = statusStats

	// Бюджетна статистика.
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(min(o.budget_cents), 0),
		       COALESCE((percentile_cont(0.5) WITHIN GROUP (ORDER BY o.budget_cents))::bigint, 0),
		       COALESCE(round(avg(o.budget_cents))::bigint, 0),
		       COALESCE(max(o.budget_cents), 0),
		       count(o.budget_cents)::int`+pdfJoin, runID).
		Scan(&report.Budget.Min, &report.Budget.Median, &report.Budget.Avg, &report.Budget.Max, &report.Budget.WithBudget); err != nil {
		return PDFRunReport{}, err
	}

	// Топ-10 навичок.
	skillStats := []PDFSkillStat{}
	if err := s.scanPairs(ctx, `
		SELECT sk, count(*)::int FROM (
			SELECT lower(trim(sk)) AS sk
			FROM order_discoveries d
			JOIN orders o ON o.id = d.order_id
			CROSS JOIN LATERAL unnest(o.skills) AS sk
			WHERE d.search_run_id = $1
		) t WHERE sk <> '' GROUP BY sk ORDER BY count(*) DESC LIMIT 10`, runID,
		func(name string, count int) {
			skillStats = append(skillStats, PDFSkillStat{Skill: name, Count: count})
		}); err != nil {
		return PDFRunReport{}, err
	}
	report.Skills = skillStats

	// Заявки по замовленнях прогона.
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM applications a
		JOIN order_discoveries d ON d.order_id = a.order_id
		WHERE d.search_run_id = $1`, runID).
		Scan(&report.Applications); err != nil {
		return PDFRunReport{}, err
	}

	return report, nil
}

func (s *Store) scanPairs(ctx context.Context, query string, arg int64, fn func(string, int)) error {
	rows, err := s.pool.Query(ctx, query, arg)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return err
		}
		fn(name, count)
	}
	return rows.Err()
}

func (s *Store) InsertReport(ctx context.Context, runID int64, reportType, fileName, storageKey string, ordersTotal int, summaryJSON string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reports (search_run_id, report_type, file_name, storage_key, mime_type, orders_total, summary_json)
		VALUES ($1, $2, $3, $4, 'application/pdf', $5, $6::jsonb)
		RETURNING id`, runID, reportType, fileName, storageKey, ordersTotal, summaryJSON).Scan(&id)
	return id, err
}

func (s *Store) GetReport(ctx context.Context, id int64) (storageKey string, fileName string, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT storage_key, file_name FROM reports WHERE id = $1`, id).
		Scan(&storageKey, &fileName)
	return storageKey, fileName, err
}

func (s *Store) LogReportDownload(ctx context.Context, reportID int64, userAgent string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO report_downloads (report_id, user_agent) VALUES ($1, NULLIF($2, ''))`,
		reportID, userAgent)
	return err
}

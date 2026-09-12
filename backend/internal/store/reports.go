// Пакет store: звіти за діапазоном дат і статистика дашборда (з дельтами для порівнянь).
package store

import (
	"context"
	"time"
)

type BranchCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type SourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type DailyCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type ReportSummary struct {
	From            time.Time      `json:"from"`
	To              time.Time      `json:"to"`
	TotalOrders     int            `json:"total_orders"`
	Removed         int            `json:"removed"`
	Applications    int            `json:"applications"`
	AvgBudgetCents  *int           `json:"avg_budget_cents"`
	ByBranch        []BranchCount `json:"by_branch"`
	BySource        []SourceCount `json:"by_source"`
	ByStatus        []StatusCount `json:"by_status"`
	Daily           []DailyCount  `json:"daily"`
	Previous        *ReportSummary `json:"previous,omitempty"`
}

func (s *Store) ReportSummary(ctx context.Context, from, to time.Time) (ReportSummary, error) {
	var r ReportSummary
	r.From, r.To = from, to

	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM orders WHERE first_seen_at >= $1 AND first_seen_at < $2),
			(SELECT COUNT(*) FROM events WHERE type = 'removed' AND created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM applications WHERE applied_at >= $1 AND applied_at < $2),
			(SELECT CAST(ROUND(AVG(budget_cents)) AS INTEGER) FROM orders
			 WHERE budget_cents IS NOT NULL AND first_seen_at >= $1 AND first_seen_at < $2)`,
		from, to).Scan(&r.TotalOrders, &r.Removed, &r.Applications, &r.AvgBudgetCents)
	if err != nil {
		return r, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT b.name, COUNT(o.id)
		FROM branches b
		LEFT JOIN orders o
		  ON o.branch_id = b.id AND o.first_seen_at >= $1 AND o.first_seen_at < $2
		GROUP BY b.name ORDER BY 2 DESC`, from, to)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var bc BranchCount
		if err := rows.Scan(&bc.Name, &bc.Count); err != nil {
			rows.Close()
			return r, err
		}
		r.ByBranch = append(r.ByBranch, bc)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT source, COUNT(*) FROM orders
		WHERE first_seen_at >= $1 AND first_seen_at < $2
		GROUP BY source ORDER BY 2 DESC`, from, to)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var sc SourceCount
		if err := rows.Scan(&sc.Source, &sc.Count); err != nil {
			rows.Close()
			return r, err
		}
		r.BySource = append(r.BySource, sc)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM orders
		WHERE first_seen_at >= $1 AND first_seen_at < $2
		GROUP BY status ORDER BY 2 DESC`, from, to)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var sc StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			rows.Close()
			return r, err
		}
		r.ByStatus = append(r.ByStatus, sc)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', first_seen_at), 'YYYY-MM-DD'), COUNT(*)
		FROM orders
		WHERE first_seen_at >= $1 AND first_seen_at < $2
		GROUP BY 1 ORDER BY 1`, from, to)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var dc DailyCount
		if err := rows.Scan(&dc.Day, &dc.Count); err != nil {
			rows.Close()
			return r, err
		}
		r.Daily = append(r.Daily, dc)
	}
	rows.Close()

	return r, nil
}

// Stats: знімок дашборда з дельтами (вчора/тиждень/місяць) для порівнянь.
type Stats struct {
	ActiveBranches    int    `json:"active_branches"`
	OrdersActive       int    `json:"orders_active"`
	OrdersNewToday     int    `json:"orders_new_today"`
	OrdersYesterday    int    `json:"orders_yesterday"`
	OrdersLast7Days    int    `json:"orders_last_7_days"`
	OrdersPrev7Days    int    `json:"orders_prev_7_days"`
	ApplicationsTotal  int    `json:"applications_total"`
	ApplicationsWeek   int    `json:"applications_week"`
	WonTotal           int    `json:"won_total"`
	WonMonth            int    `json:"won_month"`
	AvgBudgetCents     *int   `json:"avg_budget_cents"`
	TopSource          string `json:"top_source"`
	TopBranch          string `json:"top_branch"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM branches WHERE is_active),
			(SELECT COUNT(*) FROM orders WHERE status <> 'archived'),
			(SELECT COUNT(*) FROM orders WHERE status = 'new' AND first_seen_at >= current_date),
			(SELECT COUNT(*) FROM orders WHERE first_seen_at >= current_date - 1 AND first_seen_at < current_date),
			(SELECT COUNT(*) FROM orders WHERE first_seen_at >= current_date - 7),
			(SELECT COUNT(*) FROM orders WHERE first_seen_at >= current_date - 14 AND first_seen_at < current_date - 7),
			(SELECT COUNT(*) FROM applications),
			(SELECT COUNT(*) FROM applications WHERE applied_at >= current_date - 7),
			(SELECT COUNT(*) FROM orders WHERE status = 'won'),
			(SELECT COUNT(*) FROM orders WHERE status = 'won' AND first_seen_at >= current_date - 30),
			(SELECT CAST(ROUND(AVG(budget_cents)) AS INTEGER) FROM orders WHERE budget_cents IS NOT NULL),
			(SELECT COALESCE((SELECT source FROM orders GROUP BY source ORDER BY COUNT(*) DESC LIMIT 1), '')),
			(SELECT COALESCE((SELECT b.name FROM orders o JOIN branches b ON b.id = o.branch_id
			               GROUP BY b.name ORDER BY COUNT(*) DESC LIMIT 1), ''))`).
		Scan(&st.ActiveBranches, &st.OrdersActive, &st.OrdersNewToday, &st.OrdersYesterday,
			&st.OrdersLast7Days, &st.OrdersPrev7Days, &st.ApplicationsTotal, &st.ApplicationsWeek,
			&st.WonTotal, &st.WonMonth, &st.AvgBudgetCents, &st.TopSource, &st.TopBranch)
	return st, err
}

// Пакет store: звіти за діапазоном дат і статистика дашборда.
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
	From         time.Time     `json:"from"`
	To           time.Time     `json:"to"`
	TotalOrders  int           `json:"total_orders"`
	Removed      int           `json:"removed"`
	Applications int           `json:"applications"`
	ByBranch     []BranchCount `json:"by_branch"`
	BySource     []SourceCount `json:"by_source"`
	ByStatus     []StatusCount `json:"by_status"`
	Daily        []DailyCount  `json:"daily"`
}

func (s *Store) ReportSummary(ctx context.Context, from, to time.Time) (ReportSummary, error) {
	var r ReportSummary
	r.From, r.To = from, to

	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM orders WHERE first_seen_at >= $1 AND first_seen_at < $2),
			(SELECT COUNT(*) FROM events WHERE type = 'removed' AND created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM applications WHERE applied_at >= $1 AND applied_at < $2)`,
		from, to).Scan(&r.TotalOrders, &r.Removed, &r.Applications)
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

type Stats struct {
	ActiveBranches    int `json:"active_branches"`
	OrdersActive       int `json:"orders_active"`
	OrdersNewToday     int `json:"orders_new_today"`
	ApplicationsTotal  int `json:"applications_total"`
	WonTotal           int `json:"won_total"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM branches WHERE is_active),
			(SELECT COUNT(*) FROM orders WHERE status <> 'archived'),
			(SELECT COUNT(*) FROM orders WHERE status = 'new' AND first_seen_at >= current_date),
			(SELECT COUNT(*) FROM applications),
			(SELECT COUNT(*) FROM orders WHERE status = 'won')`).
		Scan(&st.ActiveBranches, &st.OrdersActive, &st.OrdersNewToday, &st.ApplicationsTotal, &st.WonTotal)
	return st, err
}

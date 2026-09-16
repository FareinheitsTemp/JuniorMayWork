package store

import (
	"context"
	"time"
)

type DashboardDaily struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type DashboardSource struct {
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	LastRunAt      *time.Time `json:"last_run_at"`
	LastOutcome    string     `json:"last_outcome"`
	LastDiscovered int        `json:"last_discovered"`
}

type DashboardStatus struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Count int    `json:"count"`
}

type DashboardTotals struct {
	Orders        int `json:"orders"`
	OrdersWeek    int `json:"orders_week"`
	NewOrders     int `json:"new_orders"`
	Applications  int `json:"applications"`
	ActiveSources int `json:"active_sources"`
}

type DashboardData struct {
	Daily    []DashboardDaily  `json:"daily"`
	Sources  []DashboardSource `json:"sources"`
	Statuses []DashboardStatus `json:"statuses"`
	Totals  DashboardTotals    `json:"totals"`
}

// GetDashboard — метрики головної сторінки поверх view міграції 006.
func (s *Store) GetDashboard(ctx context.Context) (DashboardData, error) {
	var d DashboardData

	daily := []DashboardDaily{}
	rows, err := s.pool.Query(ctx, `
		SELECT day::text, orders_total FROM v_daily_intake ORDER BY day DESC LIMIT 30`)
	if err != nil {
		return DashboardData{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p DashboardDaily
		if err := rows.Scan(&p.Day, &p.Count); err != nil {
			return DashboardData{}, err
		}
		daily = append(daily, p)
	}
	if err := rows.Err(); err != nil {
		return DashboardData{}, err
	}
	for i, j := 0, len(daily)-1; i < j; i, j = i+1, j-1 {
		daily[i], daily[j] = daily[j], daily[i]
	}
	d.Daily = daily

	sources := []DashboardSource{}
	sRows, err := s.pool.Query(ctx, `
		SELECT key, name, enabled, last_success_at, last_run_at,
		       COALESCE(last_outcome, ''), COALESCE(last_discovered, 0)
		FROM v_source_health ORDER BY key`)
	if err != nil {
		return DashboardData{}, err
	}
	defer sRows.Close()
	for sRows.Next() {
		var src DashboardSource
		if err := sRows.Scan(&src.Key, &src.Name, &src.Enabled, &src.LastSuccessAt, &src.LastRunAt, &src.LastOutcome, &src.LastDiscovered); err != nil {
			return DashboardData{}, err
		}
		sources = append(sources, src)
	}
	if err := sRows.Err(); err != nil {
		return DashboardData{}, err
	}
	d.Sources = sources

	statuses := []DashboardStatus{}
	stRows, err := s.pool.Query(ctx, `
		SELECT COALESCE(st.label, 'Без статусу'), COALESCE(st.color, ''), count(*)::int
		FROM orders o
		LEFT JOIN order_statuses st ON st.id = o.status_id
		GROUP BY 1, 2
		ORDER BY 3 DESC`)
	if err != nil {
		return DashboardData{}, err
	}
	defer stRows.Close()
	for stRows.Next() {
		var st DashboardStatus
		if err := stRows.Scan(&st.Label, &st.Color, &st.Count); err != nil {
			return DashboardData{}, err
		}
		statuses = append(statuses, st)
	}
	if err := stRows.Err(); err != nil {
		return DashboardData{}, err
	}
	d.Statuses = statuses

	if err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::int FROM orders),
			(SELECT count(*)::int FROM orders WHERE first_seen_at >= now() - interval '7 days'),
			(SELECT count(*)::int FROM orders o JOIN order_statuses st ON st.id = o.status_id WHERE st.key = 'new'),
			(SELECT count(*)::int FROM applications),
			(SELECT count(*)::int FROM sources WHERE enabled)`).
		Scan(&d.Totals.Orders, &d.Totals.OrdersWeek, &d.Totals.NewOrders, &d.Totals.Applications, &d.Totals.ActiveSources); err != nil {
		return DashboardData{}, err
	}

	return d, nil
}

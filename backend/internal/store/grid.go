package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Grid — серверна частина гнучкого керування БД: generic CRUD
// з жорстким whitelist таблиць. Імена таблиць/колонок валідуються
// проти information_schema, значення завжди параметризовані.

var gridTables = map[string]struct{}{
	"sources": {}, "source_channels": {}, "source_runs": {},
	"branches": {}, "orders": {}, "applications": {}, "events": {},
	"search_profiles": {}, "search_runs": {}, "reports": {},
	"order_statuses": {}, "application_results": {}, "skills": {}, "tags": {},
	"order_notes": {}, "order_status_history": {}, "audit_log": {},
	"report_downloads": {}, "search_run_daily_stats": {},
}

func GridTableAllowed(table string) bool {
	_, ok := gridTables[table]
	return ok
}

func GridTableNames() []string {
	names := make([]string, 0, len(gridTables))
	for name := range gridTables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type GridColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (s *Store) GridColumns(ctx context.Context, table string) ([]GridColumn, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := []GridColumn{}
	for rows.Next() {
		var c GridColumn
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// GridRows повертає сторінку рядків + total. sortCol/dir/q валідуються
// проти фактичної схеми; пошук — ILIKE лише по текстових колонках.
func (s *Store) GridRows(ctx context.Context, table string, limit, offset int, sortCol, dir, search string) ([]map[string]any, int, []GridColumn, error) {
	if !GridTableAllowed(table) {
		return nil, 0, nil, ErrNotFound
	}
	cols, err := s.GridColumns(ctx, table)
	if err != nil {
		return nil, 0, nil, err
	}
	byName := map[string]GridColumn{}
	for _, c := range cols {
		byName[c.Name] = c
	}

	where := ""
	args := []any{}
	if search != "" {
		likes := []string{}
		for _, c := range cols {
			if strings.Contains(c.Type, "text") || strings.Contains(c.Type, "char") {
				likes = append(likes, fmt.Sprintf("%s ILIKE $%d", quoteIdent(c.Name), len(args)+1))
				args = append(args, "%"+search+"%")
			}
		}
		if len(likes) > 0 {
			where = "WHERE " + strings.Join(likes, " OR ")
		}
	}

	orderBy := `"id"`
	if sortCol != "" {
		if _, ok := byName[sortCol]; ok {
			orderBy = quoteIdent(sortCol)
		}
	}
	if dir != "desc" {
		dir = "asc"
	}

	var total int
	if err := s.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s %s", quoteIdent(table), where),
		args...).Scan(&total); err != nil {
		return nil, 0, nil, err
	}

	rows, err := s.pool.Query(ctx,
		fmt.Sprintf("SELECT * FROM %s %s ORDER BY %s %s LIMIT %d OFFSET %d",
			quoteIdent(table), where, orderBy, dir, limit, offset),
		args...)
	if err != nil {
		return nil, 0, nil, err
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, 0, nil, err
		}
		row := map[string]any{}
		for i, fd := range rows.FieldDescriptions() {
			v := values[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[string(fd.Name)] = v
		}
		result = append(result, row)
	}
	return result, total, cols, rows.Err()
}

func gridAllowedKeys(values map[string]any, cols []GridColumn) []string {
	allowed := map[string]bool{}
	for _, c := range cols {
		allowed[c.Name] = true
	}
	keys := []string{}
	for name := range values {
		if name != "id" && allowed[name] {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *Store) GridInsert(ctx context.Context, table string, values map[string]any) error {
	if !GridTableAllowed(table) {
		return ErrNotFound
	}
	cols, err := s.GridColumns(ctx, table)
	if err != nil {
		return err
	}
	keys := gridAllowedKeys(values, cols)
	if len(keys) == 0 {
		return errors.New("немає допустимих полів для вставки")
	}
	quoted := make([]string, len(keys))
	placeholders := make([]string, len(keys))
	vals := make([]any, len(keys))
	for i, k := range keys {
		quoted[i] = quoteIdent(k)
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		vals[i] = values[k]
	}
	_, err = s.pool.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			quoteIdent(table), strings.Join(quoted, ", "), strings.Join(placeholders, ", ")),
		vals...)
	return err
}

func (s *Store) GridUpdate(ctx context.Context, table string, id int64, values map[string]any) error {
	if !GridTableAllowed(table) {
		return ErrNotFound
	}
	cols, err := s.GridColumns(ctx, table)
	if err != nil {
		return err
	}
	keys := gridAllowedKeys(values, cols)
	if len(keys) == 0 {
		return errors.New("немає допустимих полів для оновлення")
	}
	sets := []string{}
	vals := []any{}
	for _, k := range keys {
		vals = append(vals, values[k])
		sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdent(k), len(vals)))
	}
	vals = append(vals, id)
	tag, err := s.pool.Exec(ctx,
		fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
			quoteIdent(table), strings.Join(sets, ", "), len(vals)),
		vals...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GridDelete(ctx context.Context, table string, id int64) error {
	if !GridTableAllowed(table) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE id = $1", quoteIdent(table)), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

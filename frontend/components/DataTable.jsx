'use client';

import { useMemo, useState } from 'react';

// Універсальна таблиця в дусі Supabase: живий пошук, сортування кліком
// по заголовку, пагінація. columns: [{key, label, render?, sortValue?, sortable?}].
export default function DataTable({
  columns,
  rows,
  rowKey = 'id',
  pageSize = 15,
  empty = 'Немає даних',
  toolbar = null,
}) {
  const [query, setQuery] = useState('');
  const [sortKey, setSortKey] = useState(null);
  const [sortDir, setSortDir] = useState('asc');
  const [page, setPage] = useState(0);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((row) =>
      columns.some((c) => {
        const v = c.sortValue ? c.sortValue(row) : row[c.key];
        return v != null && String(v).toLowerCase().includes(q);
      })
    );
  }, [rows, query, columns]);

  const sorted = useMemo(() => {
    if (!sortKey) return filtered;
    const col = columns.find((c) => c.key === sortKey);
    const val = (row) => (col && col.sortValue ? col.sortValue(row) : row[sortKey]);
    return [...filtered].sort((a, b) => {
      const va = val(a);
      const vb = val(b);
      if (va === vb) return 0;
      const res = va > vb ? 1 : -1;
      return sortDir === 'asc' ? res : -res;
    });
  }, [filtered, sortKey, sortDir, columns]);

  const pages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const safePage = Math.min(page, pages - 1);
  const pageRows = sorted.slice(safePage * pageSize, safePage * pageSize + pageSize);

  function toggleSort(key) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  }

  return (
    <div className="grid">
      <div className="grid__toolbar">
        <input
          className="input grid__search"
          placeholder="Пошук по таблиці…"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setPage(0);
          }}
        />
        <span className="grid__counter">
          Показано {sorted.length === 0 ? 0 : safePage * pageSize + 1}–
          {Math.min((safePage + 1) * pageSize, sorted.length)} з {sorted.length}
        </span>
        {toolbar}
      </div>

      <div className="grid__scroll">
        <table className="table">
          <thead>
            <tr>
              {columns.map((c) => (
                <th
                  key={c.key}
                  className={c.sortable === false ? '' : 'grid__th-sort'}
                  onClick={c.sortable === false ? undefined : () => toggleSort(c.key)}
                >
                  {c.label}
                  {sortKey === c.key && (
                    <span className="grid__sort">{sortDir === 'asc' ? ' ↑' : ' ↓'}</span>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {pageRows.length === 0 && (
              <tr>
                <td colSpan={columns.length} className="page__hint">{empty}</td>
              </tr>
            )}
            {pageRows.map((row) => (
              <tr key={row[rowKey]}>
                {columns.map((c) => (
                  <td key={c.key}>{c.render ? c.render(row) : (row[c.key] ?? '—')}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pages > 1 && (
        <div className="grid__pager">
          <button type="button" className="button" onClick={() => setPage(0)} disabled={safePage === 0}>«</button>
          <button type="button" className="button" onClick={() => setPage(safePage - 1)} disabled={safePage === 0}>‹</button>
          <span className="grid__page">Сторінка {safePage + 1} / {pages}</span>
          <button type="button" className="button" onClick={() => setPage(safePage + 1)} disabled={safePage === pages - 1}>›</button>
          <button type="button" className="button" onClick={() => setPage(pages - 1)} disabled={safePage === pages - 1}>»</button>
        </div>
      )}
    </div>
  );
}

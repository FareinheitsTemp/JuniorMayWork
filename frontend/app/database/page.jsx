'use client';

import { useCallback, useEffect, useState } from 'react';
import DataTable from '@/components/DataTable';
import '@/styles/blocks/grid-extra.scss';

const TABLE_LABELS = {
  sources: 'Джерела (sources)',
  source_channels: 'Канали джерел (source_channels)',
  source_runs: 'Прогони джерел (source_runs)',
  branches: 'Вітки (branches)',
  orders: 'Замовлення (orders)',
  applications: 'Заявки (applications)',
  events: 'Події (events)',
  search_profiles: 'Профілі пошуку (search_profiles)',
  search_runs: 'Прогони пошуку (search_runs)',
  reports: 'Звіти (reports)',
  order_statuses: 'Статуси (order_statuses)',
  application_results: 'Результати заявок (app_results)',
  skills: 'Навички (skills)',
  tags: 'Теги (tags)',
  order_notes: 'Нотатки (order_notes)',
  order_status_history: 'Історія статусів (status_history)',
  audit_log: 'Журнал аудиту (audit_log)',
  report_downloads: 'Завантаження (report_downloads)',
  search_run_daily_stats: 'Щоденна стат (daily_stats)',
};

async function gridRequest(path, options = {}) {
  const res = await fetch(`/api/grid${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body && body.error) message = body.error;
    } catch {}
    throw new Error(message);
  }
  return res.json();
}

function inputType(column) {
  if (column.type === 'boolean') return 'checkbox';
  if (column.type.includes('int') || column.type.includes('numeric') || column.type.includes('float')) return 'number';
  if (column.type.includes('json')) return 'textarea';
  return 'text';
}

function formatCell(value) {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'boolean') return value ? 'TRUE' : 'FALSE';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export default function DatabasePage() {
  const [tables, setTables] = useState([]);
  const [table, setTable] = useState('orders');
  const [rows, setRows] = useState([]);
  const [columns, setColumns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState(null);
  const [creating, setCreating] = useState(false);
  const [draft, setDraft] = useState({});
  const [busy, setBusy] = useState(false);

  const loadTables = useCallback(async () => {
    try {
      const data = await gridRequest('');
      setTables(data.tables || []);
    } catch (err) {
      setError(`Таблиці недоступні: ${err.message}`);
    }
  }, []);

  const loadRows = useCallback(async () => {
    setLoading(true);
    try {
      const data = await gridRequest(`/${table}?limit=200`);
      setRows(data.rows || []);
      setColumns(data.columns || []);
      setError('');
    } catch (err) {
      setError(`Дані недоступні: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, [table]);

  useEffect(() => {
    loadTables();
  }, [loadTables]);

  useEffect(() => {
    loadRows();
  }, [loadRows]);

  function startEdit(row) {
    setCreating(false);
    setEditing({ id: row.id });
    const values = {};
    for (const column of columns) {
      if (column.name === 'id') continue;
      values[column.name] = row[column.name] ?? (column.type === 'boolean' ? false : '');
    }
    setDraft(values);
  }

  function startCreate() {
    setEditing(null);
    setCreating(true);
    const values = {};
    for (const column of columns) {
      if (column.name === 'id') continue;
      values[column.name] = column.type === 'boolean' ? false : '';
    }
    setDraft(values);
  }

  function closeForm() {
    setEditing(null);
    setCreating(false);
    setDraft({});
  }

  async function saveForm() {
    const body = {};
    for (const [key, value] of Object.entries(draft)) {
      if (value === '') continue;
      body[key] = value;
    }
    if (editing && !creating && Object.keys(body).length === 0) {
      closeForm();
      return;
    }
    setBusy(true);
    try {
      if (creating) {
        await gridRequest(`/${table}`, { method: 'POST', body: JSON.stringify(body) });
      } else if (editing) {
        await gridRequest(`/${table}/${editing.id}`, { method: 'PATCH', body: JSON.stringify(body) });
      }
      closeForm();
      await loadRows();
    } catch (err) {
      setError(`Не вдалося зберегти: ${err.message}`);
    } finally {
      setBusy(false);
    }
  }

  async function removeRow(row) {
    if (!window.confirm(`Видалити рядок #${row.id} з таблиці «${TABLE_LABELS[table] || table}»?`)) return;
    setBusy(true);
    try {
      await gridRequest(`/${table}/${row.id}`, { method: 'DELETE' });
      await loadRows();
    } catch (err) {
      setError(`Не вдалося видалити: ${err.message}`);
    } finally {
      setBusy(false);
    }
  }

  const formColumns = columns.filter((column) => column.name !== 'id');
  const showForm = creating || editing !== null;

  const tableColumns = [
    ...columns.map((column) => ({
      key: column.name,
      label: column.name,
      sortValue: (row) => row[column.name],
      render: (row) => (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12.5 }}>
          {formatCell(row[column.name])}
        </span>
      ),
    })),
    {
      key: '__actions',
      label: '',
      sortable: false,
      render: (row) => (
        <div className="row-actions">
          <button className="button button--sm" type="button" onClick={() => startEdit(row)}>✎ Редагувати</button>
          <button className="button button--danger button--sm" type="button" onClick={() => removeRow(row)}>✕</button>
        </div>
      ),
    },
  ];

  return (
    <section className="page">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontSize: 13, fontWeight: 600 }}>›_</span>
            <h1 className="page__title">SQL Data Grid</h1>
          </div>
          <p className="page__subtitle">Прямий доступ до 19 таблиць бази даних v2: селектор, пошук, пагінація, CRUD-редагування.</p>
        </div>
        <button className="button button--primary" type="button" disabled={busy || !columns.length} onClick={startCreate}>
          + Додати рядок
        </button>
      </div>

      <div className="tabs" style={{ marginBottom: 18, borderBottom: '1px solid var(--line)' }}>
        {tables.map((name) => (
          <button
            key={name}
            type="button"
            className={`tab ${name === table ? 'tab--active' : ''}`}
            onClick={() => setTable(name)}
          >
            {TABLE_LABELS[name] || name}
          </button>
        ))}
      </div>

      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      <div className="grid__scroll">
        {loading ? (
          <div className="skeleton-list" style={{ padding: 16 }}>
            <div className="skeleton" />
            <div className="skeleton" />
            <div className="skeleton" />
          </div>
        ) : (
          <DataTable
            columns={tableColumns}
            rows={rows}
            pageSize={25}
            empty="У цій таблиці ще немає рядків."
          />
        )}
      </div>

      {showForm && (
        <div className="modal-overlay" onClick={closeForm}>
          <div className="modal-card" onClick={(event) => event.stopPropagation()}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--line)', paddingBottom: 10, marginBottom: 14 }}>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent)' }}>
                {creating ? `INSERT INTO ${table}` : `UPDATE ${table} WHERE id = ${editing.id}`}
              </span>
              <button className="button button--sm" type="button" onClick={closeForm}>✕</button>
            </div>

            {formColumns.map((column) => (
              <label key={column.name} className={`field ${inputType(column) === 'checkbox' ? 'field--checkbox' : ''}`}>
                {inputType(column) === 'checkbox' ? (
                  <>
                    <input
                      type="checkbox"
                      checked={!!draft[column.name]}
                      onChange={(event) => setDraft({ ...draft, [column.name]: event.target.checked })}
                    />
                    <span style={{ fontFamily: 'var(--font-mono)' }}>{column.name}</span>
                    <span className="field__hint">({column.type})</span>
                  </>
                ) : (
                  <>
                    <span style={{ fontFamily: 'var(--font-mono)' }}>{column.name}</span>
                    <span className="field__hint">({column.type})</span>
                    {inputType(column) === 'textarea' ? (
                      <textarea
                        className="textarea"
                        value={String(draft[column.name] ?? '')}
                        onChange={(event) => setDraft({ ...draft, [column.name]: event.target.value })}
                      />
                    ) : (
                      <input
                        className="input"
                        type={inputType(column)}
                        value={String(draft[column.name] ?? '')}
                        onChange={(event) => setDraft({ ...draft, [column.name]: inputType(column) === 'number' ? (event.target.value === '' ? '' : Number(event.target.value)) : event.target.value })}
                      />
                    )}
                  </>
                )}
              </label>
            ))}
            <div className="modal-card__actions">
              <button className="button" type="button" onClick={closeForm}>Скасувати</button>
              <button className="button button--primary" type="button" disabled={busy} onClick={saveForm}>Зберегти зміни</button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

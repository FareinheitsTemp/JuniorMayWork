'use client';

import { useCallback, useEffect, useState } from 'react';
import DataTable from '@/components/DataTable';
import '@/styles/blocks/grid-extra.scss';

const TABLE_LABELS = {
  sources: 'Джерела',
  source_channels: 'Канали джерел',
  source_runs: 'Прогони джерел',
  branches: 'Вітки',
  orders: 'Замовлення',
  applications: 'Заявки',
  events: 'Події',
  search_profiles: 'Профілі пошуку',
  search_runs: 'Прогони пошуку',
  reports: 'Звіти',
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
  if (typeof value === 'boolean') return value ? 'так' : 'ні';
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
      render: (row) => formatCell(row[column.name]),
    })),
    {
      key: '__actions',
      label: '',
      sortable: false,
      render: (row) => (
        <div className="row-actions">
          <button className="button" type="button" onClick={() => startEdit(row)}>Редагувати</button>
          <button className="button button--danger" type="button" onClick={() => removeRow(row)}>Видалити</button>
        </div>
      ),
    },
  ];

  return (
    <div>
      <div className="page__head">
        <h1 className="page__title">Керування базою даних</h1>
        <p className="page__subtitle">Перегляд і редагування таблиць — виберіть таблицю нижче.</p>
      </div>

      <div className="tabs" style={{ marginBottom: 18 }}>
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

      {error && <p style={{ color: 'var(--bad)' }}>{error}</p>}

      {loading ? (
        <p className="empty-hint">Завантаження даних…</p>
      ) : (
        <DataTable
          columns={tableColumns}
          rows={rows}
          pageSize={20}
          empty="У цій таблиці ще немає рядків."
          toolbar={
            <button className="button" type="button" disabled={busy || !columns.length} onClick={startCreate}>
              + Додати рядок
            </button>
          }
        />
      )}

      {showForm && (
        <div className="modal-overlay" onClick={closeForm}>
          <div className="modal-card" onClick={(event) => event.stopPropagation()}>
            <h2>{creating ? `Новий рядок — ${TABLE_LABELS[table] || table}` : `Рядок #${editing.id}`}</h2>
            {formColumns.map((column) => (
              <label key={column.name} className={`field ${inputType(column) === 'checkbox' ? 'field--checkbox' : ''}`}>
                {inputType(column) === 'checkbox' ? (
                  <>
                    <input
                      type="checkbox"
                      checked={!!draft[column.name]}
                      onChange={(event) => setDraft({ ...draft, [column.name]: event.target.checked })}
                    />
                    {column.name} <span className="field__hint">({column.type})</span>
                  </>
                ) : (
                  <>
                    {column.name} <span className="field__hint">({column.type})</span>
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
              <button className="button" type="button" disabled={busy} onClick={saveForm}>Зберегти</button>
              <button className="button" type="button" onClick={closeForm}>Скасувати</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

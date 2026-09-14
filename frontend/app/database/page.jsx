'use client';

import { useCallback, useEffect, useState } from 'react';
import DataTable from '@/components/DataTable';

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

const PAGE_SIZE = 100;

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
      const data = await gridRequest(`/${table}?limit=${PAGE_SIZE}`);
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
    if (!window.confirm(`Видалити рядок #${row.id} з таблиці ${table}?`)) return;
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
      render: (row) => formatCell(row[column.name]),
    })),
    {
      key: '__actions',
      label: '',
      sortable: false,
      render: (row) => (
        <span style={{ display: 'flex', gap: 6 }}>
          <button className='button' type='button' onClick={() => startEdit(row)}>✎</button>
          <button className='button' type='button' onClick={() => removeRow(row)}>✕</button>
        </span>
      ),
    },
  ];

  return (
    <div className='db'>
      <div className='db__head'>
        <h2 className='page__title'>Керування базою даних</h2>
        <p className='page__subtitle'>Перегляд і редагування таблиць — виберіть таблицю нижче.</p>
      </div>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
        {tables.map((name) => (
          <button
            key={name}
            type='button'
            className='button'
            style={name === table ? { borderColor: '#2f6fed', fontWeight: 600 } : undefined}
            onClick={() => setTable(name)}
          >
            {TABLE_LABELS[name] || name}
          </button>
        ))}
      </div>

      {error && <p style={{ color: '#f85149' }}>{error}</p>}

      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
        <button className='button' type='button' disabled={busy || !columns.length} onClick={startCreate}>Додати рядок</button>
        <span style={{ color: '#8b95a8', fontSize: 13 }}>
          {loading ? 'Завантаження…' : `${rows.length} рядків (максимум ${PAGE_SIZE}) у ${TABLE_LABELS[table] || table}`}
        </span>
      </div>

      {loading ? (
        <p style={{ color: '#8b95a8' }}>Завантаження даних…</p>
      ) : (
        <DataTable columns={tableColumns} rows={rows} />
      )}

      {showForm && (
        <div
          style={{ position: 'fixed', inset: 0, background: 'rgba(15,19,29,0.6)', display: 'grid', placeItems: 'center', zIndex: 50 }}
          onClick={closeForm}
        >
          <div
            style={{ background: '#fff', borderRadius: 10, padding: 20, width: 480, maxHeight: '80vh', overflow: 'auto' }}
            onClick={(event) => event.stopPropagation()}
          >
            <h2 style={{ marginTop: 0, fontSize: 18 }}>{creating ? `Новий рядок — ${TABLE_LABELS[table] || table}` : `Рядок #${editing.id}`}</h2>
            {formColumns.map((column) => (
              <label key={column.name} style={{ display: 'block', marginBottom: 10, fontSize: 13 }}>
                {column.name} <span style={{ color: '#8b95a8' }}>({column.type})</span>
                {inputType(column) === 'checkbox' ? (
                  <input
                    type='checkbox'
                    checked={!!draft[column.name]}
                    onChange={(event) => setDraft({ ...draft, [column.name]: event.target.checked })}
                  />
                ) : inputType(column) === 'textarea' ? (
                  <textarea
                    style={{ width: '100%', minHeight: 60, boxSizing: 'border-box' }}
                    value={String(draft[column.name] ?? '')}
                    onChange={(event) => setDraft({ ...draft, [column.name]: event.target.value })}
                  />
                ) : (
                  <input
                    type={inputType(column)}
                    style={{ width: '100%', boxSizing: 'border-box' }}
                    value={String(draft[column.name] ?? '')}
                    onChange={(event) => setDraft({ ...draft, [column.name]: inputType(column) === 'number' ? (event.target.value === '' ? '' : Number(event.target.value)) : event.target.value })}
                  />
                )}
              </label>
            ))}
            <div style={{ display: 'flex', gap: 8 }}>
              <button className='button' type='button' disabled={busy} onClick={saveForm}>Зберегти</button>
              <button className='button' type='button' onClick={closeForm}>Скасувати</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

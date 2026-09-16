'use client';

import { useCallback, useEffect, useState } from 'react';
import '@/styles/blocks/grid-extra.scss';

const PAGE_SIZE = 100;

async function request(path, options = {}) {
  const res = await fetch(`/api${path}`, {
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

function formatDate(value) {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString('uk-UA');
}

function money(cents, currency) {
  if (!cents) return '—';
  const amount = (cents / 100).toLocaleString('uk-UA');
  return currency ? `${amount} ${currency}` : amount;
}

// Замовлення: пріоритетний список + панель деталей з нотатками й історією статусів.
export default function OrdersPage() {
  const [statuses, setStatuses] = useState([]);
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [detail, setDetail] = useState(null);
  const [noteText, setNoteText] = useState('');
  const [busy, setBusy] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [search, setSearch] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [statusData, gridData] = await Promise.all([
        request('/orders/statuses'),
        request(`/grid/orders?limit=${PAGE_SIZE}`),
      ]);
      setStatuses(statusData.statuses || []);
      setRows(gridData.rows || []);
      setError('');
    } catch (err) {
      setError(`Дані недоступні: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const statusByKey = {};
  statuses.forEach((s) => { statusByKey[s.key] = s; });

  async function openOrder(row) {
    setSelected(row);
    setDetail(null);
    setNoteText('');
    try {
      const [notesData, historyData] = await Promise.all([
        request(`/orders/${row.id}/notes`),
        request(`/orders/${row.id}/history`),
      ]);
      setDetail({ notes: notesData.notes || [], history: historyData.history || [] });
    } catch (err) {
      setDetail({ notes: [], history: [] });
      setError(`Деталі недоступні: ${err.message}`);
    }
  }

  function closeOrder() {
    setSelected(null);
    setDetail(null);
    setNoteText('');
  }

  async function changeStatus(order, statusKey) {
    if (!statusKey || statusKey === order.status) return;
    setBusy(true);
    try {
      await request(`/orders/${order.id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status: statusKey }),
      });
      setSelected({ ...order, status: statusKey });
      const historyData = await request(`/orders/${order.id}/history`);
      setDetail((prev) => ({ notes: (prev && prev.notes) || [], history: historyData.history || [] }));
      await load();
    } catch (err) {
      setError(`Не вдалося змінити статус: ${err.message}`);
    } finally {
      setBusy(false);
    }
  }

  async function addNote(order) {
    const body = noteText.trim();
    if (!body) return;
    setBusy(true);
    try {
      await request(`/orders/${order.id}/notes`, {
        method: 'POST',
        body: JSON.stringify({ body }),
      });
      setNoteText('');
      const notesData = await request(`/orders/${order.id}/notes`);
      setDetail((prev) => ({ history: (prev && prev.history) || [], notes: notesData.notes || [] }));
    } catch (err) {
      setError(`Не вдалося додати нотатку: ${err.message}`);
    } finally {
      setBusy(false);
    }
  }

  const visible = rows.filter((row) => {
    if (statusFilter && row.status !== statusFilter) return false;
    if (search) {
      const haystack = `${row.title || ''} ${row.source || ''}`.toLowerCase();
      if (!haystack.includes(search.toLowerCase())) return false;
    }
    return true;
  });

  return (
    <section className="page">
      <h1 className="page__title">Замовлення</h1>
      <p className="page__subtitle">Пріоритетний список знайдених замовлень: нотатки, історія і зміна статусів.</p>
      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      <div className="tabs">
        <button type="button" className={`tab ${statusFilter === '' ? 'tab--active' : ''}`} onClick={() => setStatusFilter('')}>Усі</button>
        {statuses.map((s) => (
          <button type="button" key={s.key} className={`tab ${statusFilter === s.key ? 'tab--active' : ''}`} onClick={() => setStatusFilter(s.key)}>
            {s.label}
          </button>
        ))}
      </div>

      <div className="field" style={{ marginBottom: 12 }}>
        <input className="input" placeholder="Пошук за назвою чи джерелом…" value={search} onChange={(e) => setSearch(e.target.value)} />
      </div>

      <div className="card">
        {loading ? (
          <div className="skeleton-list">
            <div className="skeleton" />
            <div className="skeleton" />
            <div className="skeleton" />
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>id</th>
                <th>Замовлення</th>
                <th>Джерело</th>
                <th>Бюджет</th>
                <th>Статус</th>
                <th>Побачено</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((row) => {
                const st = statusByKey[row.status];
                return (
                  <tr key={row.id} style={{ cursor: 'pointer' }} onClick={() => openOrder(row)}>
                    <td>#{row.id}</td>
                    <td>
                      {row.url ? (
                        <a href={row.url} target="_blank" rel="noreferrer" onClick={(e) => e.stopPropagation()}>{row.title || '—'}</a>
                      ) : (row.title || '—')}
                    </td>
                    <td>{row.source || '—'}</td>
                    <td>{money(row.budget_cents, row.currency)}</td>
                    <td>
                      <span className="status-dot" style={{ background: (st && st.color) || 'var(--text-dim)' }} />
                      {' '}{(st && st.label) || row.status || '—'}
                    </td>
                    <td>{formatDate(row.first_seen_at)}</td>
                  </tr>
                );
              })}
              {visible.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty-hint">Замовлень не знайдено.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {selected && (
        <div className="modal-overlay" onClick={closeOrder}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <h2 style={{ marginTop: 0, fontSize: 18 }}>{selected.title || `Замовлення #${selected.id}`}</h2>
            <p style={{ color: 'var(--text-dim)', fontSize: 13 }}>
              #{selected.id} · {selected.source || '—'} · {money(selected.budget_cents, selected.currency)} · {formatDate(selected.first_seen_at)}
            </p>
            {selected.url && (
              <p>
                <a href={selected.url} target="_blank" rel="noreferrer">Відкрити джерело ↗</a>
              </p>
            )}

            <div className="field">
              <label className="field__label">Статус</label>
              <select className="select" value={selected.status || ''} disabled={busy} onChange={(e) => changeStatus(selected, e.target.value)}>
                <option value="" disabled>— оберіть статус —</option>
                {statuses.map((s) => (
                  <option key={s.key} value={s.key}>{s.label}</option>
                ))}
              </select>
            </div>

            <h3 style={{ fontSize: 15 }}>Нотатки</h3>
            <div className="field">
              <textarea className="textarea" placeholder="Нова нотатка…" value={noteText} onChange={(e) => setNoteText(e.target.value)} />
              <button className="button" type="button" disabled={busy || !noteText.trim()} onClick={() => addNote(selected)}>Додати нотатку</button>
            </div>
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {((detail && detail.notes) || []).map((note) => (
                <li key={note.id} className="card" style={{ marginBottom: 8, padding: '8px 10px' }}>
                  <div style={{ fontSize: 13 }}>{note.body}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-dim)' }}>{formatDate(note.created_at)}</div>
                </li>
              ))}
              {((detail && detail.notes) || []).length === 0 && (
                <li className="empty-hint">Нотаток ще немає.</li>
              )}
            </ul>

            <h3 style={{ fontSize: 15 }}>Історія статусів</h3>
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {((detail && detail.history) || []).map((change) => (
                <li key={change.id} style={{ marginBottom: 6, fontSize: 13 }}>
                  <span className="status-dot" style={{ background: (statusByKey[change.to_status] && statusByKey[change.to_status].color) || 'var(--text-dim)' }} />
                  {' '}{change.from_status ? `${change.from_status} → ` : ''}{change.to_status}
                  {' · '}{formatDate(change.changed_at)}{change.actor ? ` · ${change.actor}` : ''}
                </li>
              ))}
              {((detail && detail.history) || []).length === 0 && (
                <li className="empty-hint">Змін статусу ще не було.</li>
              )}
            </ul>

            <div className="modal-card__actions">
              <button className="button" type="button" onClick={closeOrder}>Закрити</button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

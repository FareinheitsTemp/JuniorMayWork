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
  return currency ? `${amount} ${currency}` : `$${amount}`;
}

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
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontSize: 13, fontWeight: 600 }}>›_</span>
        <h1 className="page__title">Каталог замовлень</h1>
      </div>
      <p className="page__subtitle">Пріоритетний список вакансій: інспектор деталей, Git-style історія статусів, нотатки.</p>
      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      <div className="tabs">
        <button type="button" className={`tab ${statusFilter === '' ? 'tab--active' : ''}`} onClick={() => setStatusFilter('')}>
          Усі ({rows.length})
        </button>
        {statuses.map((s) => {
          const count = rows.filter((r) => r.status === s.key).length;
          return (
            <button type="button" key={s.key} className={`tab ${statusFilter === s.key ? 'tab--active' : ''}`} onClick={() => setStatusFilter(s.key)}>
              {s.label} <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, opacity: 0.7 }}>({count})</span>
            </button>
          );
        })}
      </div>

      <div className="field" style={{ marginBottom: 14 }}>
        <input className="input" placeholder="Пошук за назвою або джерелом (фільтр на льоту)…" value={search} onChange={(e) => setSearch(e.target.value)} />
      </div>

      <div className="grid__scroll">
        {loading ? (
          <div className="skeleton-list" style={{ padding: 16 }}>
            <div className="skeleton" />
            <div className="skeleton" />
            <div className="skeleton" />
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: 60 }}>id</th>
                <th>Замовлення</th>
                <th>Джерело</th>
                <th>Бюджет</th>
                <th>Статус</th>
                <th>Час надходження</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((row) => {
                const st = statusByKey[row.status];
                return (
                  <tr key={row.id} style={{ cursor: 'pointer' }} onClick={() => openOrder(row)}>
                    <td style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>#{row.id}</td>
                    <td>
                      <span style={{ fontWeight: 500 }}>{row.title || '—'}</span>
                    </td>
                    <td>
                      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, padding: '2px 6px', background: 'var(--bg-soft)', borderRadius: 'var(--radius-sm)' }}>
                        {row.source || '—'}
                      </span>
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>
                      {money(row.budget_cents, row.currency)}
                    </td>
                    <td>
                      <span className="status-dot" style={{ background: (st && st.color) || 'var(--text-muted)' }} />
                      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>
                        {(st && st.label) || row.status || '—'}
                      </span>
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
                      {formatDate(row.first_seen_at)}
                    </td>
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
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--line)', paddingBottom: 10, marginBottom: 14 }}>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent)' }}>
                ORDER_INSPECTOR #{selected.id}
              </span>
              <button className="button button--sm" type="button" onClick={closeOrder}>✕</button>
            </div>

            <h2 style={{ fontSize: 16, lineHeight: 1.4 }}>{selected.title || `Замовлення #${selected.id}`}</h2>

            <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', margin: '10px 0 16px', fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
              <span>джерело: <strong style={{ color: 'var(--text)' }}>{selected.source || '—'}</strong></span>
              <span>·</span>
              <span>бюджет: <strong style={{ color: 'var(--accent)' }}>{money(selected.budget_cents, selected.currency)}</strong></span>
            </div>

            {selected.url && (
              <div style={{ marginBottom: 16 }}>
                <a className="button button--sm" href={selected.url} target="_blank" rel="noreferrer">
                  Відкрити на сайті джерела ↗
                </a>
              </div>
            )}

            <div className="field">
              <label className="field__label">Змінити статус замовлення</label>
              <select className="select" value={selected.status || ''} disabled={busy} onChange={(e) => changeStatus(selected, e.target.value)}>
                {statuses.map((s) => (
                  <option key={s.key} value={s.key}>{s.label}</option>
                ))}
              </select>
            </div>

            <h3>Нотатки та коментарі</h3>
            <div className="field">
              <textarea className="textarea" placeholder="Додати робочу нотатку до замовлення…" value={noteText} onChange={(e) => setNoteText(e.target.value)} />
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 6 }}>
                <button className="button button--primary" type="button" disabled={busy || !noteText.trim()} onClick={() => addNote(selected)}>
                  + Додати нотатку
                </button>
              </div>
            </div>

            <div style={{ display: 'grid', gap: 6, marginBottom: 16 }}>
              {((detail && detail.notes) || []).map((note) => (
                <div key={note.id} style={{ background: 'var(--bg-soft)', border: '1px solid var(--line)', borderRadius: 'var(--radius)', padding: '8px 12px' }}>
                  <div style={{ fontSize: 13 }}>{note.body}</div>
                  <div style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>
                    {formatDate(note.created_at)}
                  </div>
                </div>
              ))}
              {((detail && detail.notes) || []).length === 0 && (
                <div className="empty-hint" style={{ padding: '8px 0' }}>Нотаток ще немає.</div>
              )}
            </div>

            <h3>Історія переходів (Audit Timeline)</h3>
            <div style={{ display: 'grid', gap: 6, borderLeft: '2px solid var(--line)', paddingLeft: 12, marginLeft: 4 }}>
              {((detail && detail.history) || []).map((change) => (
                <div key={change.id} style={{ fontSize: 12.5, fontFamily: 'var(--font-mono)' }}>
                  <span className="status-dot" style={{ background: (statusByKey[change.to_status] && statusByKey[change.to_status].color) || 'var(--text-muted)' }} />
                  <span style={{ color: 'var(--text-dim)' }}>{change.from_status ? `${change.from_status} → ` : ''}</span>
                  <strong style={{ color: 'var(--accent)' }}>{change.to_status}</strong>
                  <span style={{ color: 'var(--text-muted)', marginLeft: 8 }}>{formatDate(change.changed_at)}</span>
                </div>
              ))}
              {((detail && detail.history) || []).length === 0 && (
                <div className="empty-hint" style={{ padding: '4px 0' }}>Історія чиста.</div>
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';

const STATUSES = ['new', 'seen', 'applied', 'won', 'lost', 'archived'];

function money(cents) {
  if (cents == null) return '—';
  return `$${(cents / 100).toLocaleString('en-US')}`;
}

// Замовлення: повний контроль над тим, що в БД — фільтри, статуси, вітки, заявки.
export default function OrdersPage() {
  const [orders, setOrders] = useState([]);
  const [branches, setBranches] = useState([]);
  const [status, setStatus] = useState('');
  const [branchId, setBranchId] = useState('');
  const [query, setQuery] = useState('');
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const params = { limit: 300 };
      if (status) params.status = status;
      if (branchId) params.branch_id = branchId;
      if (query) params.q = query;
      const [ords, brs] = await Promise.all([api.orders(params), api.branches()]);
      setOrders(ords || []);
      setBranches(brs || []);
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }, [status, branchId, query]);

  useEffect(() => {
    load();
  }, [load]);

  async function updateOrder(order, patch) {
    try {
      await api.updateOrder(order.id, patch);
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  async function applyTo(order) {
    try {
      await api.apply(order.id);
      setFlash(`Заявку на «${order.title}» подано`);
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  async function remove(order) {
    if (!window.confirm(`Видалити «${order.title}» з БД?`)) return;
    try {
      await api.deleteOrder(order.id);
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  return (
    <section className="page">
      <h1 className="page__title">Замовлення</h1>
      {flash && <div className="page__hint">{flash}</div>}

      <div className="page__row">
        <div className="field">
          <label className="field__label" htmlFor="f-status">Статус</label>
          <select id="f-status" className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">усі</option>
            {STATUSES.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field__label" htmlFor="f-branch">Вітка</label>
          <select id="f-branch" className="select" value={branchId} onChange={(e) => setBranchId(e.target.value)}>
            <option value="">усі</option>
            {branches.map((b) => (
              <option key={b.id} value={b.id}>{b.name}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field__label" htmlFor="f-q">Пошук</label>
          <input
            id="f-q"
            className="input"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="react, лендінг…"
          />
        </div>
      </div>

      <div className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Замовлення</th>
              <th>Джерело</th>
              <th>Бюджет</th>
              <th>Вітка</th>
              <th>Статус</th>
              <th>Дії</th>
            </tr>
          </thead>
          <tbody>
            {orders.length === 0 && (
              <tr>
                <td colSpan={6} className="page__hint">Нічого не знайдено.</td>
              </tr>
            )}
            {orders.map((o) => (
              <tr key={o.id}>
                <td>
                  <a href={o.url} target="_blank" rel="noreferrer">{o.title}</a>
                </td>
                <td>{o.source}</td>
                <td>{money(o.budget_cents)}</td>
                <td>
                  <select
                    className="select"
                    value={o.branch_id ?? ''}
                    onChange={(e) => e.target.value && updateOrder(o, { branch_id: Number(e.target.value) })}
                  >
                    <option value="">—</option>
                    {branches.map((b) => (
                      <option key={b.id} value={b.id}>{b.name}</option>
                    ))}
                  </select>
                </td>
                <td>
                  <select
                    className="select"
                    value={o.status}
                    onChange={(e) => updateOrder(o, { status: e.target.value })}
                  >
                    {STATUSES.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                </td>
                <td>
                  <div className="leaf__actions">
                    <button
                      type="button"
                      className="button button--primary"
                      onClick={() => applyTo(o)}
                      disabled={o.status !== 'new' && o.status !== 'seen'}
                    >
                      Заявка
                    </button>
                    <button type="button" className="button button--danger" onClick={() => remove(o)}>×</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

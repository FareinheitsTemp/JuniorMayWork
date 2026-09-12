'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import DataTable from '@/components/DataTable';
import { statusLabel, EVENT_LABELS, APP_RESULT_LABELS } from '@/lib/status';

const TABS = [
  { key: 'orders', label: 'Замовлення' },
  { key: 'events', label: 'Події' },
  { key: 'applications', label: 'Заявки' },
  { key: 'branches', label: 'Вітки' },
];

const STATUSES = ['new', 'seen', 'applied', 'won', 'lost', 'archived'];

function money(cents) {
  if (cents == null) return '—';
  return `$${(cents / 100).toLocaleString('en-US')}`;
}

// «База даних»: прямий доступ до всіх таблиць у стилі Supabase.
export default function DatabasePage() {
  const [tab, setTab] = useState('orders');
  const [orders, setOrders] = useState([]);
  const [events, setEvents] = useState([]);
  const [apps, setApps] = useState([]);
  const [branches, setBranches] = useState([]);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const [o, e, a, b] = await Promise.all([
        api.orders({ limit: 500 }),
        api.events(500),
        api.applications(500),
        api.branches(),
      ]);
      setOrders(o || []);
      setEvents(e || []);
      setApps(a || []);
      setBranches(b || []);
    } catch (err) {
      setFlash(`Помилка: ${err.message}`);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function patchOrder(order, patch) {
    try {
      await api.updateOrder(order.id, patch);
      await load();
    } catch (err) {
      setFlash(`Помилка: ${err.message}`);
    }
  }

  async function removeOrder(order) {
    if (!window.confirm(`Видалити #${order.id} «${order.title}»?`)) return;
    try {
      await api.deleteOrder(order.id);
      await load();
    } catch (err) {
      setFlash(`Помилка: ${err.message}`);
    }
  }

  async function removeBranch(b) {
    if (!window.confirm(`Видалити вітку «${b.name}»?`)) return;
    try {
      await api.deleteBranch(b.id);
      await load();
    } catch (err) {
      setFlash(`Помилка: ${err.message}`);
    }
  }

  const orderColumns = [
    { key: 'id', label: 'id' },
    { key: 'source', label: 'Джерело' },
    {
      key: 'title',
      label: 'Заголовок',
      render: (o) => (
        <a href={o.url} target="_blank" rel="noreferrer">{o.title}</a>
      ),
    },
    {
      key: 'budget',
      label: 'Бюджет',
      sortValue: (o) => o.budget_cents ?? -1,
      render: (o) => money(o.budget_cents),
    },
    {
      key: 'branch_id',
      label: 'Вітка',
      sortValue: (o) => o.branch_id ?? 0,
      render: (o) => (
        <select
          className="select"
          value={o.branch_id ?? ''}
          onChange={(e) => e.target.value && patchOrder(o, { branch_id: Number(e.target.value) })}
        >
          <option value="">—</option>
          {branches.map((b) => (
            <option key={b.id} value={b.id}>{b.name}</option>
          ))}
        </select>
      ),
    },
    {
      key: 'status',
      label: 'Статус',
      render: (o) => (
        <select
          className="select"
          value={o.status}
          onChange={(e) => patchOrder(o, { status: e.target.value })}
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>{statusLabel(s)}</option>
          ))}
        </select>
      ),
    },
    {
      key: 'first_seen_at',
      label: 'Побачено',
      sortValue: (o) => o.first_seen_at,
      render: (o) => new Date(o.first_seen_at).toLocaleString('uk-UA'),
    },
    {
      key: '_del',
      label: '',
      sortable: false,
      render: (o) => (
        <button type="button" className="button button--danger" onClick={() => removeOrder(o)}>
          ×
        </button>
      ),
    },
  ];

  const eventColumns = [
    { key: 'id', label: 'id' },
    {
      key: 'type',
      label: 'Тип',
      render: (e) => (
        <span className={`badge badge--${e.type === 'removed' ? 'removed' : e.payload?.status || 'new'}`}>
          {EVENT_LABELS[e.type] || e.type}
        </span>
      ),
    },
    {
      key: 'title',
      label: 'Замовлення',
      sortValue: (e) => e.payload?.title ?? '',
      render: (e) =>
        e.payload?.url ? (
          <a href={e.payload.url} target="_blank" rel="noreferrer">
            {e.payload.title || `#${e.payload.order_id}`}
          </a>
        ) : (
          `#${e.order_id ?? '—'}`
        ),
    },
    {
      key: 'created_at',
      label: 'Дата',
      sortValue: (e) => e.created_at,
      render: (e) => new Date(e.created_at).toLocaleString('uk-UA'),
    },
  ];

  const appColumns = [
    { key: 'id', label: 'id' },
    {
      key: 'applied_at',
      label: 'Дата',
      sortValue: (a) => a.applied_at,
      render: (a) => new Date(a.applied_at).toLocaleString('uk-UA'),
    },
    { key: 'order_title', label: 'Замовлення' },
    { key: 'note', label: 'Нотатка', render: (a) => a.note || '—' },
    {
      key: 'result',
      label: 'Результат',
      render: (a) => (
        <span className="badge badge--applied">{APP_RESULT_LABELS[a.result] || a.result}</span>
      ),
    },
  ];

  const branchColumns = [
    { key: 'id', label: 'id' },
    { key: 'name', label: 'Назва' },
    {
      key: 'keywords',
      label: 'Ключові слова',
      sortValue: (b) => (b.keywords || []).join(' '),
      render: (b) => (b.keywords || []).join(', '),
    },
    {
      key: 'max_budget_cents',
      label: 'Ліміт',
      render: (b) => `$${(b.max_budget_cents / 100).toFixed(0)}`,
    },
    { key: 'is_active', label: 'Активна', render: (b) => (b.is_active ? 'так' : 'ні') },
    {
      key: '_del',
      label: '',
      sortable: false,
      render: (b) => (
        <button type="button" className="button button--danger" onClick={() => removeBranch(b)}>
          ×
        </button>
      ),
    },
  ];

  return (
    <section className="page">
      <h1 className="page__title">База даних</h1>
      <p className="page__subtitle">
        Замовлення: {orders.length} · Події: {events.length} · Заявки: {apps.length} · Вітки: {branches.length}
      </p>
      {flash && <div className="page__hint">{flash}</div>}

      <div className="tabs">
        {TABS.map((t) => (
          <button
            key={t.key}
            type="button"
            className={`tab ${tab === t.key ? 'tab--active' : ''}`}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'orders' && (
        <div className="card">
          <DataTable columns={orderColumns} rows={orders} empty="Таблиця порожня." />
        </div>
      )}
      {tab === 'events' && (
        <div className="card">
          <DataTable columns={eventColumns} rows={events} empty="Таблиця порожня." />
        </div>
      )}
      {tab === 'applications' && (
        <div className="card">
          <DataTable columns={appColumns} rows={apps} empty="Таблиця порожня." />
        </div>
      )}
      {tab === 'branches' && (
        <div className="card">
          <DataTable columns={branchColumns} rows={branches} empty="Таблиця порожня." />
        </div>
      )}
    </section>
  );
}

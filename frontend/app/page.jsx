'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useLive } from '@/lib/useLive';
import StatCard from '@/components/StatCard';
import TreeView from '@/components/TreeView';
import EventFeed from '@/components/EventFeed';
import NewOrderAlert from '@/components/NewOrderAlert';
import Donut from '@/components/Donut';
import { statusLabel } from '@/lib/status';

const STATUS_COLORS = {
  new: '#3fcf8e',
  seen: '#ffb547',
  applied: '#5b8cff',
  won: '#2ea87a',
  lost: '#ff6b7a',
  archived: '#8f9cb2',
};

function money(cents) {
  if (cents == null) return '—';
  return `$${(cents / 100).toFixed(0)}`;
}

function delta(current, previous) {
  if (previous == null || previous === 0) return null;
  return Math.round(((current - previous) / previous) * 100);
}

function todayStr(offsetDays = 0) {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  return d.toISOString().slice(0, 10);
}

export default function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [report, setReport] = useState(null);
  const [branches, setBranches] = useState([]);
  const [orders, setOrders] = useState([]);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const [st, rep, br, ords] = await Promise.all([
        api.stats(),
        api.report(todayStr(-7), todayStr()),
        api.branches(),
        api.orders({ status: 'new', limit: 300 }),
      ]);
      setStats(st);
      setReport(rep);
      setBranches(br || []);
      setOrders(ords || []);
    } catch (e) {
      setFlash(`Не вдалося завантажити дані: ${e.message}`);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // живі події: новий листочок → оновлюємо все
  useLive((event) => {
    if (event.type === 'new') {
      load();
    }
  });

  async function handleAction(kind, order) {
    try {
      if (kind === 'apply') {
        await api.apply(order.id);
        setFlash(`Заявку на «${order.title}» подано`);
      } else if (kind === 'seen') {
        await api.updateOrder(order.id, { status: 'seen' });
      }
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  async function runNow() {
    setBusy(true);
    try {
      await api.scraperRun();
      setFlash('Збір запущено');
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    } finally {
      setBusy(false);
    }
  }

  const donutParts = (report?.by_status || []).map((s) => ({
    label: statusLabel(s.status),
    value: s.count,
    color: STATUS_COLORS[s.status] || '#8f9cb2',
  }));

  return (
    <section className="page">
      <NewOrderAlert />
      <div className="page__row" style={{ justifyContent: 'space-between' }}>
        <h1 className="page__title">Дашборд</h1>
        <button type="button" className="button button--primary" onClick={runNow} disabled={busy}>
          {busy ? 'Запускаю…' : 'Зібрати зараз'}
        </button>
      </div>

      {flash && <div className="page__hint">{flash}</div>}

      <div className="page__row">
        <StatCard label="Активні вітки" value={stats?.active_branches ?? '—'} />
        <StatCard label="В роботі" value={stats?.orders_active ?? '—'} />
        <StatCard
          label="Нових сьогодні"
          value={stats?.orders_new_today ?? '—'}
          delta={stats ? delta(stats.orders_new_today, stats.orders_yesterday) : null}
          hint="проти вчора"
        />
        <StatCard
          label="За 7 днів"
          value={stats?.orders_last_7_days ?? '—'}
          delta={stats ? delta(stats.orders_last_7_days, stats.orders_prev_7_days) : null}
          hint="проти минулого тижня"
        />
        <StatCard
          label="Сер. чек"
          value={stats ? money(stats.avg_budget_cents) : '—'}
          hint={stats?.top_source ? `топ-джерело: ${stats.top_source}` : undefined}
        />
        <StatCard
          label="Виграно"
          value={stats?.won_total ?? '—'}
          hint={stats ? `за місяць: ${stats.won_month}` : undefined}
        />
      </div>

      <div className="dashboard">
        <div className="dashboard__tree card">
          <h2 className="feed__title">Дерево замовлень</h2>
          <TreeView branches={branches} orders={orders} onAction={handleAction} />
        </div>
        <div className="dashboard__side">
          <div className="card">
            <h2 className="feed__title">Статуси за 7 днів</h2>
            <Donut parts={donutParts} />
          </div>
          <EventFeed />
        </div>
      </div>
    </section>
  );
}

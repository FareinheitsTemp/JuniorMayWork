'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useLive } from '@/lib/useLive';
import StatCard from '@/components/StatCard';
import TreeView from '@/components/TreeView';
import EventFeed from '@/components/EventFeed';
import NewOrderAlert from '@/components/NewOrderAlert';

export default function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [branches, setBranches] = useState([]);
  const [orders, setOrders] = useState([]);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const [st, br, ords] = await Promise.all([
        api.stats(),
        api.branches(),
        api.orders({ status: 'new', limit: 300 }),
      ]);
      setStats(st);
      setBranches(br || []);
      setOrders(ords || []);
    } catch (e) {
      setFlash(`Не вдалося завантажити дані: ${e.message}`);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // живі події: новий листочок → оновлюємо дерево й статистику
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
        <StatCard label="Замовлень в роботі" value={stats?.orders_active ?? '—'} />
        <StatCard label="Нових сьогодні" value={stats?.orders_new_today ?? '—'} />
        <StatCard label="Заявок" value={stats?.applications_total ?? '—'} />
        <StatCard label="Виграно" value={stats?.won_total ?? '—'} />
      </div>

      <div className="dashboard">
        <div className="dashboard__tree card">
          <h2 className="feed__title">Дерево замовлень</h2>
          <TreeView branches={branches} orders={orders} onAction={handleAction} />
        </div>
        <EventFeed />
      </div>
    </section>
  );
}

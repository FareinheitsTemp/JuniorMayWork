'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { radar } from '@/lib/radar';
import { useLive } from '@/lib/useLive';
import StatCard from '@/components/StatCard';
import EventFeed from '@/components/EventFeed';
import NewOrderAlert from '@/components/NewOrderAlert';
import Donut from '@/components/Donut';
import FreshInbox from '@/components/FreshInbox';
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

function SourcePipeline() {
  const [runs, setRuns] = useState([]);

  const load = useCallback(async () => {
    try {
      setRuns(await radar.runs(12));
    } catch {
      setRuns([]);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = setInterval(load, 15000);
    return () => clearInterval(timer);
  }, [load]);

  return (
    <section className="pipeline card">
      <div className="pipeline__head">
        <h2 className="feed__title">Стан джерел</h2>
        <span className="pipeline__live">● live</span>
      </div>
      <div className="pipeline__list">
        {runs.length === 0 && <p className="page__hint">Ще немає запусків Radar.</p>}
        {runs.slice(0, 6).map((run) => (
          <div key={run.id} className="pipeline__run">
            <i className={`pipeline__dot pipeline__dot--${run.outcome}`} />
            <span className="pipeline__source">source #{run.source_id}</span>
            <span>{run.discovered_count} знайдено</span>
            <span>{run.inserted_count} нових</span>
          </div>
        ))}
      </div>
    </section>
  );
}

export default function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [report, setReport] = useState(null);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const [st, rep] = await Promise.all([
        api.stats(),
        api.report(todayStr(-7), todayStr()),
      ]);
      setStats(st);
      setReport(rep);
    } catch (e) {
      setFlash(`Не вдалося завантажити дані: ${e.message}`);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useLive((event) => {
    if (event.type === 'new') load();
  });

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
        <div>
          <h1 className="page__title">Радар вакансій</h1>
          <p className="page__subtitle">Свіжі, релевантні та ще не зачеплені гіги</p>
        </div>
        <button type="button" className="button button--primary" onClick={runNow} disabled={busy}>
          {busy ? 'Запускаю…' : 'Зібрати зараз'}
        </button>
      </div>

      {flash && <div className="page__hint">{flash}</div>}

      <div className="page__row">
        <StatCard label="Активні вітки" value={stats?.active_branches ?? '—'} hint={"на сьогодні:"} />
        <StatCard label="В роботі" value={stats?.orders_active ?? '—'} hint={"Ваші робочі вітки"}/>
        <StatCard label="Нових сьогодні" value={stats?.orders_new_today ?? '—'} delta={stats ? delta(stats.orders_new_today, stats.orders_yesterday) : null} hint="проти вчора" />
        <StatCard label="За 7 днів" value={stats?.orders_last_7_days ?? '—'} delta={stats ? delta(stats.orders_last_7_days, stats.orders_prev_7_days) : null} hint="проти минулого тижня" />
        <StatCard label="Сер. чек" value={stats ? money(stats.avg_budget_cents) : '—'} hint={stats?.top_source ? `топ-джерело: ${stats.top_source}` : undefined} />
        <StatCard label="Виграно" value={stats?.won_total ?? '—'} hint={stats ? `за місяць: ${stats.won_month}` : undefined} />
      </div>

      <div className="radar-dashboard">
        <FreshInbox onChange={load} />
        <div className="radar-dashboard__side">
          <div className="card">
            <h2 className="feed__title">Статуси за 7 днів</h2>
            <Donut parts={donutParts} />
          </div>
          <SourcePipeline />
          <EventFeed />
        </div>
      </div>
    </section>
  );
}

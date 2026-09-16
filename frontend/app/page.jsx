'use client';

import { useCallback, useEffect, useState } from 'react';
import StatCard from '@/components/StatCard';
import Donut from '@/components/Donut';
import '@/styles/blocks/grid-extra.scss';

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

const OUTCOME_LABELS = {
  success: 'працює',
  warning: 'попередження',
  failure: 'помилка',
  running: 'виконується',
};

const OUTCOME_DOT = {
  success: 'ok',
  warning: 'warn',
  failure: 'bad',
  running: 'warn',
};

// Дашборд: метрики з GET /api/dashboard (view v_daily_intake і v_source_health).
export default function DashboardPage() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await request('/dashboard'));
      setError('');
    } catch (err) {
      setError(`Дані недоступні: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = setInterval(load, 30000);
    return () => clearInterval(timer);
  }, [load]);

  const daily = (data && data.daily) || [];
  const maxDaily = Math.max(1, ...daily.map((p) => p.count));
  const statusParts = ((data && data.statuses) || [])
    .filter((s) => s.count > 0)
    .map((s) => ({ label: s.label, value: s.count, color: s.color || 'var(--text-dim)' }));
  const totals = (data && data.totals) || {};

  return (
    <section className="page">
      <div className="page__row" style={{ justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 className="page__title">Дашборд</h1>
          <p className="page__subtitle">Загальна карти: замовлення, джерела і динаміка за 30 днів.</p>
        </div>
        <button className="button" type="button" onClick={load}>Оновити</button>
      </div>
      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      {loading && !data ? (
        <div className="skeleton-list">
          <div className="skeleton" />
          <div className="skeleton" />
          <div className="skeleton" />
        </div>
      ) : (
        <>
          <div className="stat-row">
            <div className="card"><StatCard label="Замовлень усього" value={totals.orders ?? '—'} /></div>
            <div className="card"><StatCard label="За 7 днів" value={totals.orders_week ?? '—'} /></div>
            <div className="card"><StatCard label="Нових" value={totals.new_orders ?? '—'} /></div>
            <div className="card"><StatCard label="Заявок" value={totals.applications ?? '—'} /></div>
            <div className="card"><StatCard label="Активних джерел" value={totals.active_sources ?? '—'} /></div>
            {statusParts.length > 0 && (
              <div className="donut-card">
                <Donut parts={statusParts} size={110} />
                <div className="donut-card__legend">
                  {statusParts.map((part) => (
                    <div key={part.label} className="donut-card__legend-item">
                      <span className="status-dot" style={{ background: part.color }} />
                      {part.label}: {part.value}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <h2 className="section-title">Динаміка надходжень (30 днів)</h2>
          <div className="card" style={{ padding: '16px 18px' }}>
            {daily.length === 0 ? (
              <p className="empty-hint">Даних ще немає — запусти збір на сторінці «Налаштування».</p>
            ) : (
              <div>
                <div className="dash-chart">
                  {daily.map((p) => (
                    <div
                      key={p.day}
                      className="dash-chart__bar"
                      title={`${p.day}: ${p.count}`}
                      style={{ height: `${Math.max(2, (p.count / maxDaily) * 100)}%` }}
                    />
                  ))}
                </div>
                <div className="dash-chart__axis">
                  <span>{(daily[0] && daily[0].day || '').slice(5)}</span>
                  <span>макс за добу: {maxDaily}</span>
                  <span>{(daily[daily.length - 1] && daily[daily.length - 1].day || '').slice(5)}</span>
                </div>
              </div>
            )}
          </div>

          <h2 className="section-title">Джерела</h2>
          <div className="grid__scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>Джерело</th>
                  <th>Увімкнено</th>
                  <th>Останній прогін</th>
                  <th>Статус</th>
                  <th>Знайдено</th>
                  <th>Останній успіх</th>
                </tr>
              </thead>
              <tbody>
                {((data && data.sources) || []).map((s) => (
                  <tr key={s.key}>
                    <td>{s.name || s.key}</td>
                    <td>{s.enabled ? 'так' : 'ні'}</td>
                    <td>{formatDate(s.last_run_at)}</td>
                    <td>
                      <span className={`status-dot status-dot--${OUTCOME_DOT[s.last_outcome] || 'idle'}`} />
                      {OUTCOME_LABELS[s.last_outcome] || s.last_outcome || '—'}
                    </td>
                    <td>{s.last_discovered}</td>
                    <td>{formatDate(s.last_success_at)}</td>
                  </tr>
                ))}
                {((data && data.sources) || []).length === 0 && (
                  <tr>
                    <td colSpan={6} className="empty-hint">Джерел поки немає.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  );
}

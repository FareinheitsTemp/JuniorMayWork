'use client';

import { useCallback, useEffect, useState } from 'react';
import StatCard from '@/components/StatCard';
import Donut from '@/components/Donut';
import { TerminalIcon, RefreshIcon } from '@/components/Icons';
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
  success: 'ACTIVE',
  warning: 'WARN',
  failure: 'FAIL',
  running: 'RUNNING',
};

const OUTCOME_DOT = {
  success: 'ok',
  warning: 'warn',
  failure: 'bad',
  running: 'warn',
};

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
    .map((s) => ({ label: s.label, value: s.count, color: s.color }));
  const totals = (data && data.totals) || {};

  return (
    <section className="page">
      <div className="page__row" style={{ justifyContent: 'space-between', alignItems: 'center', marginBottom: 18 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <TerminalIcon size={18} style={{ color: 'var(--accent)' }} />
            <h1 className="page__title">Дашборд системи</h1>
          </div>
          <p className="page__subtitle">Телеметрія збору замовлень, аналітика джерел і динаміка за 30 днів.</p>
        </div>
        <button className="button" type="button" onClick={load}>
          <RefreshIcon size={14} />
          <span>Оновити</span>
        </button>
      </div>
      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      {loading && !data ? (
        <div className="skeleton-list">
          <div className="skeleton" style={{ height: 90 }} />
          <div className="skeleton" style={{ height: 160 }} />
        </div>
      ) : (
        <>
          <div className="stat-row">
            <div className="card"><StatCard label="Замовлень усього" value={totals.orders} /></div>
            <div className="card"><StatCard label="За останні 7 днів" value={totals.orders_week} /></div>
            <div className="card"><StatCard label="Нових (черга)" value={totals.new_orders} /></div>
            <div className="card"><StatCard label="Поданих заявок" value={totals.applications} /></div>
            <div className="card"><StatCard label="Активних каналів" value={totals.active_sources} /></div>
            {statusParts.length > 0 && (
              <div className="donut-card">
                <Donut parts={statusParts} size={96} strokeWidth={12} />
                <div className="donut-card__legend">
                  {statusParts.map((part) => (
                    <div key={part.label} className="donut-card__legend-item">
                      <span className="status-dot" style={{ background: part.color || 'var(--accent)' }} />
                      <span>{part.label}: <strong>{part.value}</strong></span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <h2 className="section-title">Динаміка надходжень (останні 30 днів)</h2>
          <div className="card" style={{ padding: '16px 20px', marginBottom: 24 }}>
            {daily.length === 0 ? (
              <p className="empty-hint">Даних ще немає — запустіть збір на сторінці «Налаштування».</p>
            ) : (
              <div>
                <div className="dash-chart">
                  {daily.map((p) => (
                    <div
                      key={p.day}
                      className="dash-chart__bar"
                      title={`${p.day}: ${p.count}`}
                      style={{ height: `${Math.max(3, (p.count / maxDaily) * 100)}%` }}
                    />
                  ))}
                </div>
                <div className="dash-chart__axis">
                  <span>{(daily[0] && daily[0].day || '').slice(5)}</span>
                  <span style={{ color: 'var(--accent)' }}>пік доби: {maxDaily}</span>
                  <span>{(daily[daily.length - 1] && daily[daily.length - 1].day || '').slice(5)}</span>
                </div>
              </div>
            )}
          </div>

          <h2 className="section-title">Канали збору (Sources Telemetry)</h2>
          <div className="grid__scroll">
            <table className="table">
              <thead>
                <tr>
                  <th>Джерело</th>
                  <th>Статус</th>
                  <th>Останній прогін</th>
                  <th>Знайдено</th>
                  <th>Останній успіх</th>
                </tr>
              </thead>
              <tbody>
                {((data && data.sources) || []).map((s) => (
                  <tr key={s.key}>
                    <td>
                      <strong style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{s.name || s.key}</strong>
                    </td>
                    <td>
                      <span className={`status-dot status-dot--${OUTCOME_DOT[s.last_outcome] || 'idle'}`} />
                      <span style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        fontWeight: 600,
                        color: OUTCOME_DOT[s.last_outcome] === 'ok' ? 'var(--good)' : OUTCOME_DOT[s.last_outcome] === 'bad' ? 'var(--bad)' : 'var(--warn)',
                      }}>
                        {OUTCOME_LABELS[s.last_outcome] || s.last_outcome || '—'}
                      </span>
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
                      {formatDate(s.last_run_at)}
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 13, fontWeight: 600 }}>
                      {s.last_discovered}
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)' }}>
                      {formatDate(s.last_success_at)}
                    </td>
                  </tr>
                ))}
                {((data && data.sources) || []).length === 0 && (
                  <tr>
                    <td colSpan={5} className="empty-hint">Джерел поки немає.</td>
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

'use client';

import { useCallback, useEffect, useState } from 'react';
import StatCard from '@/components/StatCard';
import Donut from '@/components/Donut';
import '@/styles/blocks/grid-extra.scss';

const RUN_STATUS_LABELS = {
  queued: 'у черзі',
  running: 'виконується',
  completed: 'завершено',
  partial: 'частково',
  failed: 'помилка',
};

const RUN_STATUS_DOT = {
  queued: 'idle',
  running: 'warn',
  completed: 'ok',
  partial: 'warn',
  failed: 'bad',
};

const RUN_STATUS_COLORS = {
  queued: 'var(--text-dim)',
  running: 'var(--warn)',
  completed: 'var(--good)',
  partial: 'var(--warn)',
  failed: 'var(--bad)',
};

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

export default function ReportsPage() {
  const [summary, setSummary] = useState(null);
  const [runs, setRuns] = useState([]);
  const [reports, setReports] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyRun, setBusyRun] = useState(null);
  const [lastReport, setLastReport] = useState(null);

  const load = useCallback(async () => {
    try {
      const [summaryData, runsData, reportsData] = await Promise.all([
        request('/reports/summary'),
        request('/grid/search_runs?limit=50'),
        request('/grid/reports?limit=50'),
      ]);
      setSummary(summaryData);
      setRuns(runsData.rows || []);
      setReports(reportsData.rows || []);
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

  async function generatePDF(run) {
    setBusyRun(run.id);
    setLastReport(null);
    try {
      const data = await request(`/runs/${run.id}/report`, { method: 'POST' });
      setLastReport(data);
      await load();
    } catch (err) {
      setError(`Не вдалося згенерувати PDF: ${err.message}`);
    } finally {
      setBusyRun(null);
    }
  }

  const statusParts = Object.entries(RUN_STATUS_LABELS)
    .map(([key, label]) => ({
      label,
      value: runs.filter((run) => run.status === key).length,
      color: RUN_STATUS_COLORS[key],
    }))
    .filter((part) => part.value > 0);

  return (
    <div>
      <div className="page__head">
        <h1 className="page__title">Звіти</h1>
        <p className="page__subtitle">Зведення по замовленнях та PDF-звіти прогонів пошуку.</p>
      </div>

      {error && <p style={{ color: 'var(--bad)' }}>{error}</p>}

      <div className="stat-row">
        <div className="card"><StatCard label="Замовлень усього" value={summary ? (summary.total_orders ?? '—') : '…'} /></div>
        <div className="card"><StatCard label="Прогонів пошуку" value={runs.length} /></div>
        <div className="card"><StatCard label="PDF-звітів" value={reports.length} /></div>
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

      {lastReport && (
        <div className="report-banner">
          <strong>PDF готовий:</strong> {lastReport.file_name}
          <a className="button" href={lastReport.download}>Завантажити</a>
        </div>
      )}

      <h2 className="section-title">Прогони пошуку</h2>
      {loading ? (
        <p className="empty-hint">Завантаження…</p>
      ) : (
        <div className="grid__scroll">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Статус</th>
                <th>Почато</th>
                <th>Завершено</th>
                <th>Знайдено</th>
                <th>PDF</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => (
                <tr key={run.id}>
                  <td>#{run.id}</td>
                  <td>
                    <span className={`status-dot status-dot--${RUN_STATUS_DOT[run.status] || 'idle'}`} />
                    {RUN_STATUS_LABELS[run.status] || run.status || '—'}
                  </td>
                  <td>{formatDate(run.started_at)}</td>
                  <td>{formatDate(run.finished_at)}</td>
                  <td>{run.orders_found ?? '—'}</td>
                  <td>
                    <button className="button" type="button" disabled={busyRun === run.id} onClick={() => generatePDF(run)}>
                      {busyRun === run.id ? 'Генерується…' : 'Згенерувати PDF'}
                    </button>
                  </td>
                </tr>
              ))}
              {runs.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty-hint">Прогонів пошуку ще немає.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <h2 className="section-title">Готові PDF-звіти</h2>
      <div className="grid__scroll">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Файл</th>
              <th>Замовлень</th>
              <th>Створено</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {reports.map((report) => (
              <tr key={report.id}>
                <td>#{report.id}</td>
                <td>{report.file_name}</td>
                <td>{report.orders_total ?? '—'}</td>
                <td>{formatDate(report.created_at)}</td>
                <td>
                  <a className="button" href={`/api/reports/${report.id}/download`}>Завантажити</a>
                </td>
              </tr>
            ))}
            {reports.length === 0 && (
              <tr>
                <td colSpan={5} className="empty-hint">Ще не згенеровано жодного PDF.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

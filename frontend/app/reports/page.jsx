'use client';

import { useCallback, useEffect, useState } from 'react';
import StatCard from '@/components/StatCard';
import Donut from '@/components/Donut';

const RUN_STATUS_LABELS = {
  queued: 'у черзі',
  running: 'виконується',
  completed: 'завершено',
  partial: 'частково',
  failed: 'помилка',
};

const RUN_STATUS_COLORS = {
  queued: '#8b95a8',
  running: '#d29922',
  completed: '#3fb950',
  partial: '#a371f7',
  failed: '#f85149',
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

  const th = { textAlign: 'left', padding: '8px 10px', borderBottom: '1px solid #e3e7ee', fontSize: 13, color: '#5a6474' };
  const td = { padding: '8px 10px', borderBottom: '1px solid #eef1f5', fontSize: 13 };

  return (
    <div className='reports'>
      <div className='db__head'>
        <h1 className='page__title'>Звіти</h1>
        <p className='page__subtitle'>Зведення по замовленнях та PDF-звіти прогонів пошуку.</p>
      </div>

      {error && <p style={{ color: '#f85149' }}>{error}</p>}

      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center', marginBottom: 20 }}>
        <StatCard label='Замовлень усього' value={summary ? (summary.total_orders ?? '—') : '…'} />
        <StatCard label='Прогонів пошуку' value={runs.length} />
        <StatCard label='PDF-звітів' value={reports.length} />
        {statusParts.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Donut parts={statusParts} size={120} />
            <div style={{ fontSize: 12, color: '#5a6474' }}>
              {statusParts.map((part) => (
                <div key={part.label} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ width: 8, height: 8, borderRadius: 4, background: part.color, display: 'inline-block' }} />
                  {part.label}: {part.value}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {lastReport && (
        <div style={{ background: '#eaf6ec', border: '1px solid #bfe3c6', borderRadius: 8, padding: '10px 14px', marginBottom: 18, display: 'flex', alignItems: 'center', gap: 10 }}>
          <strong>PDF готовий:</strong> {lastReport.file_name}
          <a className='button' style={{ textDecoration: 'none' }} href={lastReport.download}>Завантажити</a>
        </div>
      )}

      <h2 style={{ fontSize: 16, margin: '0 0 10px' }}>Прогони пошуку</h2>
      {loading ? (
        <p style={{ color: '#8b95a8' }}>Завантаження…</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={th}>ID</th>
              <th style={th}>Статус</th>
              <th style={th}>Почато</th>
              <th style={th}>Завершено</th>
              <th style={th}>Знайдено</th>
              <th style={th}>PDF</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr key={run.id}>
                <td style={td}>#{run.id}</td>
                <td style={td}>{RUN_STATUS_LABELS[run.status] || run.status || '—'}</td>
                <td style={td}>{formatDate(run.started_at)}</td>
                <td style={td}>{formatDate(run.finished_at)}</td>
                <td style={td}>{run.orders_found ?? '—'}</td>
                <td style={td}>
                  <button className='button' type='button' disabled={busyRun === run.id} onClick={() => generatePDF(run)}>
                    {busyRun === run.id ? 'Генерується…' : 'Згенерувати PDF'}
                  </button>
                </td>
              </tr>
            ))}
            {runs.length === 0 && (
              <tr>
                <td style={td} colSpan={6}>Прогонів пошуку ще немає.</td>
              </tr>
            )}
          </tbody>
        </table>
      )}

      <h2 style={{ fontSize: 16, margin: '24px 0 10px' }}>Готові PDF-звіти</h2>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={th}>ID</th>
            <th style={th}>Файл</th>
            <th style={th}>Замовлень</th>
            <th style={th}>Створено</th>
            <th style={th} />
          </tr>
        </thead>
        <tbody>
          {reports.map((report) => (
            <tr key={report.id}>
              <td style={td}>#{report.id}</td>
              <td style={td}>{report.file_name}</td>
              <td style={td}>{report.orders_total ?? '—'}</td>
              <td style={td}>{formatDate(report.created_at)}</td>
              <td style={td}>
                <a className='button' style={{ textDecoration: 'none' }} href={`/api/reports/${report.id}/download`}>Завантажити</a>
              </td>
            </tr>
          ))}
          {reports.length === 0 && (
            <tr>
              <td style={td} colSpan={5}>Ще не згенеровано жодного PDF.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

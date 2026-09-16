'use client';

import { useCallback, useEffect, useState } from 'react';
import StatCard from '@/components/StatCard';
import Donut from '@/components/Donut';
import { TerminalIcon, FileTextIcon, DownloadIcon, TrashIcon } from '@/components/Icons';
import '@/styles/blocks/grid-extra.scss';

const RUN_STATUS_LABELS = {
  queued: 'QUEUED',
  running: 'RUNNING',
  completed: 'DONE',
  partial: 'PARTIAL',
  failed: 'FAILED',
};

const RUN_STATUS_DOT = {
  queued: 'idle',
  running: 'warn',
  completed: 'ok',
  partial: 'warn',
  failed: 'bad',
};

const RUN_STATUS_COLORS = {
  queued: 'var(--text-muted)',
  running: 'var(--warn)',
  completed: 'var(--good)',
  partial: 'var(--accent)',
  failed: 'var(--bad)',
};

const TYPE_LABELS = {
  run: 'RUN',
  summary: 'SUMMARY',
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
  const [busyReport, setBusyReport] = useState(null);
  const [lastReport, setLastReport] = useState(null);
  const [runFilter, setRunFilter] = useState('');

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

  async function removeReport(report) {
    if (!window.confirm(`Видалити звіт #${report.id} (${report.file_name})?`)) return;
    setBusyReport(report.id);
    try {
      await request(`/grid/reports/${report.id}`, { method: 'DELETE' });
      await load();
    } catch (err) {
      setError(`Не вдалося видалити звіт: ${err.message}`);
    } finally {
      setBusyReport(null);
    }
  }

  const runsVisible = runFilter ? runs.filter((run) => run.status === runFilter) : runs;

  const statusParts = Object.entries(RUN_STATUS_LABELS)
    .map(([key, label]) => ({
      label,
      value: runsVisible.filter((run) => run.status === key).length,
      color: RUN_STATUS_COLORS[key],
    }))
    .filter((part) => part.value > 0);

  return (
    <section className="page">
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <TerminalIcon size={18} style={{ color: 'var(--accent)' }} />
        <h1 className="page__title">Звіти та аналітика</h1>
      </div>
      <p className="page__subtitle">Генерація багатосторінкових PDF-звітів, аналітика прогонів і архів експортів.</p>

      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      <div className="stat-row">
        <div className="card"><StatCard label="Замовлень усього" value={summary ? (summary.total_orders ?? '—') : '…'} /></div>
        <div className="card"><StatCard label="Прогонів пошуку" value={runs.length} /></div>
        <div className="card"><StatCard label="Готових PDF" value={reports.length} /></div>
        {statusParts.length > 0 && (
          <div className="donut-card">
            <Donut parts={statusParts} size={96} strokeWidth={12} />
            <div className="donut-card__legend">
              {statusParts.map((part) => (
                <div key={part.label} className="donut-card__legend-item">
                  <span className="status-dot" style={{ background: part.color }} />
                  <span>{part.label}: <strong>{part.value}</strong></span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {lastReport && (
        <div className="report-banner">
          <FileTextIcon size={16} style={{ color: 'var(--good)' }} />
          <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--good)' }}>PDF_READY:</span>
          <strong style={{ flex: 1, fontFamily: 'var(--font-mono)' }}>{lastReport.file_name}</strong>
          <a className="button button--primary button--sm" href={lastReport.download}>
            <DownloadIcon size={13} />
            <span>Завантажити PDF</span>
          </a>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', margin: '24px 0 10px' }}>
        <h2 className="section-title" style={{ margin: 0 }}>Прогони пошуку (Search Runs)</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}>filter:</span>
          <select className="select" style={{ padding: '4px 8px', fontSize: 12, fontFamily: 'var(--font-mono)' }} value={runFilter} onChange={(e) => setRunFilter(e.target.value)}>
            <option value="">ALL_STATUSES</option>
            {Object.entries(RUN_STATUS_LABELS).map(([key, label]) => (
              <option key={key} value={key}>{label}</option>
            ))}
          </select>
        </div>
      </div>

      {loading ? (
        <div className="skeleton-list">
          <div className="skeleton" />
          <div className="skeleton" />
        </div>
      ) : (
        <div className="grid__scroll" style={{ marginBottom: 28 }}>
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: 60 }}>Run ID</th>
                <th>Статус</th>
                <th>Час старту</th>
                <th>Завершено</th>
                <th>Знайдено</th>
                <th style={{ textAlign: 'right' }}>Дія</th>
              </tr>
            </thead>
            <tbody>
              {runsVisible.map((run) => (
                <tr key={run.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>#{run.id}</td>
                  <td>
                    <span className={`status-dot status-dot--${RUN_STATUS_DOT[run.status] || 'idle'}`} />
                    <span style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      fontWeight: 600,
                      color: RUN_STATUS_DOT[run.status] === 'ok' ? 'var(--good)' : RUN_STATUS_DOT[run.status] === 'bad' ? 'var(--bad)' : 'var(--warn)',
                    }}>
                      {RUN_STATUS_LABELS[run.status] || run.status || '—'}
                    </span>
                  </td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
                    {formatDate(run.started_at)}
                  </td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)' }}>
                    {formatDate(run.finished_at)}
                  </td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 13, fontWeight: 600 }}>
                    {run.orders_found ?? '—'}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <button className="button button--primary button--sm" type="button" disabled={busyRun === run.id} onClick={() => generatePDF(run)}>
                      <FileTextIcon size={12} />
                      <span>{busyRun === run.id ? 'Генерація…' : 'Згенерувати PDF'}</span>
                    </button>
                  </td>
                </tr>
              ))}
              {runsVisible.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty-hint">Прогонів пошуку з цим статусом немає.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <h2 className="section-title">Згенеровані PDF-звіти (Archive)</h2>
      <div className="grid__scroll">
        <table className="table">
          <thead>
            <tr>
              <th style={{ width: 60 }}>id</th>
              <th>Тип</th>
              <th>Файл звіту</th>
              <th>Замовлень</th>
              <th>Створено</th>
              <th style={{ textAlign: 'right' }}>Дії</th>
            </tr>
          </thead>
          <tbody>
            {reports.map((report) => (
              <tr key={report.id}>
                <td style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>#{report.id}</td>
                <td>
                  <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, padding: '2px 6px', background: 'var(--bg-soft)', borderRadius: 'var(--radius-sm)', color: 'var(--accent)' }}>
                    {TYPE_LABELS[report.report_type] || report.report_type || 'RUN'}
                  </span>
                </td>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12.5 }}>
                  {report.file_name}
                </td>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 13, fontWeight: 600 }}>
                  {report.orders_total ?? '—'}
                </td>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
                  {formatDate(report.created_at)}
                </td>
                <td style={{ textAlign: 'right' }}>
                  <div className="row-actions">
                    <a className="button button--sm" href={`/api/reports/${report.id}/download`}>
                      <DownloadIcon size={12} />
                      <span>Завантажити</span>
                    </a>
                    <button className="button button--danger button--sm" type="button" disabled={busyReport === report.id} onClick={() => removeReport(report)}>
                      <TrashIcon size={12} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {reports.length === 0 && (
              <tr>
                <td colSpan={6} className="empty-hint">Ще не згенеровано жодного PDF-звіту.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}

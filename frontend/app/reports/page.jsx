'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import StatCard from '@/components/StatCard';

function todayStr(offsetDays = 0) {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  return d.toISOString().slice(0, 10);
}

// BarChart: горизонтальні CSS-бари (без бібліотек).
function BarChart({ title, rows }) {
  const max = Math.max(1, ...rows.map((r) => r.count));
  return (
    <div className="chart card">
      <h2 className="feed__title">{title}</h2>
      <div className="chart__list">
        {rows.length === 0 && <div className="page__hint">немає даних за період</div>}
        {rows.map((r) => {
          const label = r.name ?? r.day ?? r.source ?? '—';
          return (
            <div key={String(label)} className="chart__row">
              <span className="chart__label">{label}</span>
              <div className="chart__bar-track">
                <div className="chart__bar" style={{ width: `${(r.count / max) * 100}%` }} />
              </div>
              <span className="chart__count">{r.count}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Звіти: які замовлення ловились з яку по яку дату + заявки з результатами + CSV.
export default function ReportsPage() {
  const [from, setFrom] = useState(todayStr(-30));
  const [to, setTo] = useState(todayStr());
  const [report, setReport] = useState(null);
  const [apps, setApps] = useState([]);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      const [rep, applications] = await Promise.all([
        api.report(from, to),
        api.applications(200),
      ]);
      setReport(rep);
      setApps(applications || []);
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }, [from, to]);

  useEffect(() => {
    load();
  }, [load]);

  async function setAppResult(id, result) {
    try {
      await api.setApplicationResult(id, result);
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  function exportCsv() {
    if (!report) return;
    const lines = [
      'metric,key,count',
      `total,orders,${report.total_orders}`,
      `total,removed,${report.removed}`,
      `total,applications,${report.applications}`,
      ...(report.by_branch || []).map((r) => `branch,${JSON.stringify(r.name)},${r.count}`),
      ...(report.by_source || []).map((r) => `source,${r.source},${r.count}`),
      ...(report.by_status || []).map((r) => `status,${r.status},${r.count}`),
      ...(report.daily || []).map((r) => `day,${r.day},${r.count}`),
    ];
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `jmw-report-${from}_${to}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <section className="page">
      <div className="page__row" style={{ justifyContent: 'space-between' }}>
        <h1 className="page__title">Звіти</h1>
        <button type="button" className="button" onClick={exportCsv}>
          Експорт CSV
        </button>
      </div>
      {flash && <div className="page__hint">{flash}</div>}

      <div className="page__row">
        <div className="field">
          <label className="field__label" htmlFor="r-from">З дати</label>
          <input id="r-from" type="date" className="input" value={from} onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div className="field">
          <label className="field__label" htmlFor="r-to">По дату</label>
          <input id="r-to" type="date" className="input" value={to} onChange={(e) => setTo(e.target.value)} />
        </div>
      </div>

      <div className="page__row">
        <StatCard label="Замовлень впіймано" value={report?.total_orders ?? '—'} />
        <StatCard label="Зникло (в архів)" value={report?.removed ?? '—'} />
        <StatCard label="Заявок" value={report?.applications ?? '—'} />
      </div>

      <BarChart title="По вітках" rows={report?.by_branch || []} />
      <BarChart title="По джерелах" rows={report?.by_source || []} />
      <BarChart title="По днях" rows={report?.daily || []} />

      <div className="card">
        <h2 className="feed__title">Заявки та результати</h2>
        <table className="table">
          <thead>
            <tr>
              <th>Дата</th>
              <th>Замовлення</th>
              <th>Нотатка</th>
              <th>Результат</th>
            </tr>
          </thead>
          <tbody>
            {apps.length === 0 && (
              <tr>
                <td colSpan={4} className="page__hint">Заявок поки не було.</td>
              </tr>
            )}
            {apps.map((a) => (
              <tr key={a.id}>
                <td>{new Date(a.applied_at).toLocaleString('uk-UA')}</td>
                <td>{a.order_title}</td>
                <td>{a.note || '—'}</td>
                <td>
                  <select
                    className="select"
                    value={a.result}
                    onChange={(e) => setAppResult(a.id, e.target.value)}
                  >
                    {['pending', 'accepted', 'declined'].map((r) => (
                      <option key={r} value={r}>{r}</option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { api } from '@/lib/api';
import { radar } from '@/lib/radar';
import { useLive } from '@/lib/useLive';

const OUTCOME = {
  success: { label: 'працює', cls: 'success' },
  failure: { label: 'помилка', cls: 'failure' },
  warning: { label: 'частково', cls: 'warning' },
  running: { label: 'виконується', cls: 'running' },
};

function ago(value) {
  if (!value) return '—';
  const minutes = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 60000));
  if (minutes < 1) return 'щойно';
  if (minutes < 60) return `${minutes} хв тому`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} год тому`;
  return `${Math.floor(hours / 24)} дн тому`;
}

export default function SettingsPage() {
  const [runs, setRuns] = useState([]);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState('');
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      setRuns(await radar.runs(60));
      setError('');
    } catch (e) {
      setError(e.message);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = setInterval(load, 30000);
    return () => clearInterval(timer);
  }, [load]);

  useLive((event) => {
    if (event.type === 'new') load();
  });

  async function runNow() {
    setBusy(true);
    try {
      await api.scraperRun();
      setFlash('Збір запущено');
      setTimeout(load, 4000);
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    } finally {
      setBusy(false);
    }
  }

  const sources = useMemo(() => {
    const map = new Map();
    for (const run of runs || []) {
      if (!map.has(run.source_id)) map.set(run.source_id, []);
      map.get(run.source_id).push(run);
    }
    return [...map.entries()].map(([id, list]) => ({
      id,
      last: list[0],
      total: list.length,
      failures: list.filter((r) => r.outcome === 'failure').length,
      discovered: list.reduce((s, r) => s + (r.discovered_count || 0), 0),
      inserted: list.reduce((s, r) => s + (r.inserted_count || 0), 0),
    }));
  }, [runs]);

  return (
    <section className="page">
      <div className="page__row" style={{ justifyContent: 'space-between' }}>
        <div>
          <h1 className="page__title">Джерела та збір</h1>
          <p className="page__subtitle">Health джерел, запуск збору та керування каналами</p>
        </div>
        <div className="sources__actions">
          <button type="button" className="button" onClick={load}>Оновити</button>
          <button type="button" className="button button--primary" onClick={runNow} disabled={busy}>
            {busy ? 'Запускаю…' : 'Зібрати зараз'}
          </button>
        </div>
      </div>

      {flash && <div className="page__hint">{flash}</div>}
      {error && <div className="inbox__error">{error}</div>}

      <section className="card sources">
        <h2 className="feed__title">Стан джерел</h2>
        {sources.length === 0 && <p className="page__hint">Ще немає запусків — натисни «Зібрати зараз».</p>}
        {sources.length > 0 && (
          <table className="sources__table">
            <thead>
              <tr>
                <th>Джерело</th>
                <th>Статус</th>
                <th>Останній запуск</th>
                <th>Запусків</th>
                <th>Знайдено</th>
                <th>Нових</th>
                <th>Помилок</th>
              </tr>
            </thead>
            <tbody>
              {sources.map((s) => {
                const outcome = OUTCOME[s.last.outcome] || { label: s.last.outcome || '—', cls: 'warning' };
                const when = s.last.started_at || s.last.created_at || s.last.finished_at;
                return (
                  <tr key={s.id}>
                    <td className="sources__name">source #{s.id}</td>
                    <td><span className={`sources__badge sources__badge--${outcome.cls}`}>{outcome.label}</span></td>
                    <td>{ago(when)}</td>
                    <td>{s.total}</td>
                    <td>{s.discovered}</td>
                    <td>{s.inserted}</td>
                    <td>{s.failures > 0 ? s.failures : '—'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </section>

      <section className="card sources">
        <h2 className="feed__title">Канали Telegram</h2>
        <p className="page__hint">
          Додавання й вимикання каналів з’явиться наступним комітом: потрібен CRUD на бекенді,
          щоб список жив у базі, а не в <code>.env</code>.
        </p>
      </section>
    </section>
  );
}

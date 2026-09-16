'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { radar } from '@/lib/radar';
import { TerminalIcon, RefreshIcon } from '@/components/Icons';
import '@/styles/blocks/grid-extra.scss';

const OUTCOME = {
  success: { label: 'ONLINE', dot: 'ok' },
  warning: { label: 'WARN', dot: 'warn' },
  failure: { label: 'FAIL', dot: 'bad' },
  running: { label: 'POLLING', dot: 'warn' },
};

function formatDate(val) {
  if (!val) return '—';
  const d = new Date(val);
  return Number.isNaN(d.getTime()) ? String(val) : d.toLocaleString('uk-UA');
}

export default function SettingsPage() {
  const [sources, setSources] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [flash, setFlash] = useState('');
  const [runningKey, setRunningKey] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const list = await api.sources();
      const withRuns = await Promise.all(
        list.map(async (s) => {
          try {
            const runs = await radar.sourceRuns(s.id, 1);
            return { ...s, last: runs[0] || null };
          } catch {
            return { ...s, last: null };
          }
        })
      );
      setSources(withRuns);
      setError('');
    } catch (err) {
      setError(`Не вдалося завантажити джерела: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function triggerRun(s) {
    setRunningKey(s.key);
    try {
      const res = await api.triggerSource(s.key);
      setFlash(`Запуск ${s.name}: знайдено ${res.discovered}, нових ${res.inserted}`);
      await load();
    } catch (err) {
      setError(`Помилка запуску: ${err.message}`);
    } finally {
      setRunningKey(null);
    }
  }

  async function toggleSource(s) {
    try {
      await api.updateSource(s.id, { enabled: !s.enabled });
      await load();
    } catch (err) {
      setError(`Не вдалося змінити стан: ${err.message}`);
    }
  }

  return (
    <section className="page">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <TerminalIcon size={18} style={{ color: 'var(--accent)' }} />
            <h1 className="page__title">Налаштування каналів (Pipeline Config)</h1>
          </div>
          <p className="page__subtitle">Керування підключеннями джерел, частотою збору та ручний запуск пайплайну.</p>
        </div>
        <button className="button" type="button" onClick={load}>
          <RefreshIcon size={14} />
          <span>Оновити стан</span>
        </button>
      </div>

      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}
      {flash && (
        <div className="report-banner" style={{ marginBottom: 16 }}>
          <strong style={{ color: 'var(--good)' }}>PIPELINE:</strong>
          <span>{flash}</span>
          <button className="button button--sm" type="button" style={{ marginLeft: 'auto' }} onClick={() => setFlash('')}>✕</button>
        </div>
      )}

      <div className="grid__scroll">
        {loading ? (
          <div className="skeleton-list" style={{ padding: 16 }}>
            <div className="skeleton" />
            <div className="skeleton" />
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Джерело</th>
                <th>Тип</th>
                <th>Стан</th>
                <th>Останній прогін</th>
                <th>Телеметрія прогону</th>
                <th style={{ textAlign: 'right' }}>Ручний запуск</th>
              </tr>
            </thead>
            <tbody>
              {sources.map((s) => {
                const outcome = (s.last && OUTCOME[s.last.outcome]) || { label: 'IDLE', dot: 'idle' };
                return (
                  <tr key={s.id}>
                    <td>
                      <div style={{ fontWeight: 600 }}>{s.name}</div>
                      <div style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}>
                        key: {s.key} {s.base_url ? `· ${s.base_url}` : ''}
                      </div>
                    </td>
                    <td>
                      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, padding: '2px 6px', background: 'var(--bg-soft)', borderRadius: 'var(--radius-sm)' }}>
                        {s.kind}
                      </span>
                    </td>
                    <td>
                      <label className="field--checkbox" style={{ margin: 0 }}>
                        <input
                          type="checkbox"
                          checked={s.enabled}
                          onChange={() => toggleSource(s)}
                        />
                        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5 }}>
                          {s.enabled ? 'ENABLED' : 'PAUSED'}
                        </span>
                      </label>
                    </td>
                    <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-dim)' }}>
                      {s.last ? formatDate(s.last.started_at) : '—'}
                    </td>
                    <td>
                      <span className={`status-dot status-dot--${outcome.dot}`} />
                      <span style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11,
                        fontWeight: 600,
                        color: outcome.dot === 'ok' ? 'var(--good)' : outcome.dot === 'bad' ? 'var(--bad)' : 'var(--warn)',
                        marginRight: 8,
                      }}>
                        {outcome.label}
                      </span>
                      {s.last && (
                        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-muted)' }}>
                          +{s.last.discovered_count} знайдено / +{s.last.inserted_count} нових
                        </span>
                      )}
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <button
                        className="button button--primary button--sm"
                        type="button"
                        disabled={runningKey === s.key || !s.enabled}
                        onClick={() => triggerRun(s)}
                      >
                        <RefreshIcon size={12} />
                        <span>{runningKey === s.key ? 'Збір…' : 'Запустити'}</span>
                      </button>
                    </td>
                  </tr>
                );
              })}
              {sources.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty-hint">Джерел збору не налаштовано.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}

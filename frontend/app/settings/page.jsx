'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useLive } from '@/lib/useLive';
import StatCard from '@/components/StatCard';

// Налаштування: стан скрейпера, ручний запуск, шпаргалка конфігу.
export default function SettingsPage() {
  const [status, setStatus] = useState(null);
  const [busy, setBusy] = useState(false);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      setStatus(await api.scraperStatus());
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = setInterval(load, 10000);
    return () => clearInterval(timer);
  }, [load]);

  // будь-яка подія — привід освіжити статус
  useLive(() => {
    load();
  });

  async function run() {
    setBusy(true);
    try {
      await api.scraperRun();
      setFlash('Збір запущено');
      setTimeout(load, 1500);
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="page">
      <h1 className="page__title">Налаштування</h1>
      {flash && <div className="page__hint">{flash}</div>}

      <div className="card">
        <h2 className="feed__title">Скрейпер</h2>
        <div className="page__row">
          <StatCard label="Стан" value={status?.running ? 'працює' : 'чекає'} />
          <StatCard
            label="Останній збір"
            value={status?.last_run ? new Date(status.last_run).toLocaleString('uk-UA') : '—'}
          />
          <StatCard
            label="Наступний"
            value={status?.next_run ? new Date(status.next_run).toLocaleString('uk-UA') : '—'}
          />
          <StatCard label="Інтервал" value={status?.interval ?? '—'} />
        </div>
        <div className="page__row">
          <span className="page__hint">
            Джерела: {(status?.sources || []).join(', ') || '—'}
          </span>
          <button type="button" className="button button--primary" onClick={run} disabled={busy}>
            {busy ? 'Запускаю…' : 'Зібрати зараз'}
          </button>
        </div>
      </div>

      <div className="card">
        <h2 className="feed__title">Конфігурація (змінні оточення бекенду)</h2>
        <ul className="page__hint">
          <li><code>JMW_DATABASE_URL</code> — рядок підключення до PostgreSQL</li>
          <li><code>JMW_POLL_INTERVAL</code> — інтервал опитування джерел (дефолт 90s)</li>
          <li><code>JMW_SOURCES</code> — джерела через кому (upwork, reddit, weblancer)</li>
          <li><code>JMW_ARCHIVE_DIR</code> — каталог JSON-архіву зниклих замовлень</li>
        </ul>
        <p className="page__hint">
          Зміни конфігу вступають у силу після перезапуску бекенду. Якщо Weblancer змінив верстку —
          підкоригуй селектори в backend/internal/scraper/sources/weblancer.go.
        </p>
      </div>
    </section>
  );
}

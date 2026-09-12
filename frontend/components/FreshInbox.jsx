'use client';

import { useCallback, useEffect, useState } from 'react';
import { radar } from '@/lib/radar';

function money(cents) {
  if (cents == null) return '—';
  return `$${(cents / 100).toFixed(0)}`;
}

function age(value) {
  const minutes = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 60000));
  if (minutes < 1) return 'щойно';
  if (minutes < 60) return `${minutes} хв тому`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} год тому`;
  return `${Math.floor(hours / 24)} дн тому`;
}

// Свіжий inbox: свідомо максимум 8 рядків, щоб не перетворювати dashboard на кишку.
export default function FreshInbox({ onChange }) {
  const [orders, setOrders] = useState([]);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      setOrders(await radar.fresh(8));
      setError('');
    } catch (err) {
      setError(err.message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function action(kind, order) {
    try {
      if (kind === 'seen') await radar.markSeen(order.id);
      if (kind === 'dismiss') await radar.dismiss(order.id);
      setOrders((prev) => prev.filter((item) => item.id !== order.id));
      onChange?.();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <section className="inbox card">
      <div className="inbox__head">
        <div>
          <h2 className="feed__title">Свіжі гіги</h2>
          <p className="inbox__hint">Тільки ще не переглянуті, унікальні та пріоритетні</p>
        </div>
        <button type="button" className="button" onClick={load}>Оновити</button>
      </div>
      {error && <div className="inbox__error">{error}</div>}
      <div className="inbox__list">
        {orders.length === 0 && (
          <div className="inbox__empty">Черга порожня. Нові гіги з’являться після наступного збору.</div>
        )}
        {orders.map((order) => (
          <article key={order.id} className="inbox__item" style={{ '--score': Math.round(order.priority_score) }}>
            <div className="inbox__score" title={`Priority score: ${order.priority_score.toFixed(1)}`}>
              {Math.round(order.priority_score)}
            </div>
            <div className="inbox__main">
              <a className="inbox__title" href={order.url} target="_blank" rel="noreferrer">
                {order.title}
              </a>
              <div className="inbox__meta">
                <span>{order.source}</span>
                <span>{money(order.budget_cents)}</span>
                <span>{age(order.source_published_at || order.first_seen_at)}</span>
                <span>fresh {Math.round(order.freshness_score)}</span>
                <span>match {Math.round(order.relevance_score)}</span>
              </div>
            </div>
            <div className="inbox__actions">
              <button type="button" className="button" onClick={() => action('seen', order)}>Побачив</button>
              <button type="button" className="button button--danger" onClick={() => action('dismiss', order)}>×</button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}

'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { useLive } from '@/lib/useLive';
import { EVENT_LABELS } from '@/lib/status';

function eventTitle(event) {
  const p = event.payload || {};
  if (p.title) return p.title;
  return `замовлення #${p.order_id ?? event.order_id ?? '?'}`;
}

// Живий фід: останні події з БД + нові по WebSocket у реальному часі.
export default function EventFeed({ limit = 30 }) {
  const [events, setEvents] = useState([]);

  useEffect(() => {
    api.events(100)
      .then((list) => setEvents(Array.isArray(list) ? list.slice(0, limit) : []))
      .catch(() => setEvents([]));
  }, [limit]);

  const connected = useLive((event) => {
    setEvents((prev) => [event, ...prev].slice(0, limit));
  });

  return (
    <div className="feed card">
      <div className="feed__head">
        <h2 className="feed__title">Живий фід</h2>
        <span className={`feed__state ${connected ? 'feed__state--on' : ''}`}>
          {connected ? '● підключено' : '○ офлайн'}
        </span>
      </div>
      <ul className="feed__list">
        {events.length === 0 && (
          <li className="feed__empty">Подій поки немає — запусти збір на сторінці «Налаштування».</li>
        )}
        {events.map((e) => (
          <li key={e.id || `${e.type}-${e.created_at}`} className="feed__item">
            <span className={`badge badge--${e.type === 'removed' ? 'removed' : e.payload?.status || 'new'}`}>
              {EVENT_LABELS[e.type] || e.type}
            </span>
            <span className="feed__text">{eventTitle(e)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

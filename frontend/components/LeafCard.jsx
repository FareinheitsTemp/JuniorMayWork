'use client';

function money(cents) {
  if (cents == null) return '—';
  return `$${(cents / 100).toLocaleString('en-US')}`;
}

// Листочок: картка замовлення з діями.
export default function LeafCard({ order, onAction }) {
  return (
    <div className="leaf">
      <div className="leaf__head">
        <a className="leaf__title" href={order.url} target="_blank" rel="noreferrer">
          {order.title}
        </a>
        <span className={`badge badge--${order.status}`}>{order.status}</span>
      </div>
      <div className="leaf__meta">
        <span className="leaf__source">{order.source}</span>
        <span className="leaf__budget">{money(order.budget_cents)}</span>
        <span className="leaf__time">{new Date(order.first_seen_at).toLocaleString('uk-UA')}</span>
      </div>
      {order.skills != null && order.skills.length > 0 && (
        <div className="leaf__skills">
          {order.skills.map((s) => (
            <span key={s} className="leaf__skill">{s}</span>
          ))}
        </div>
      )}
      <div className="leaf__actions">
        <button
          type="button"
          className="button button--primary"
          onClick={() => onAction('apply', order)}
          disabled={order.status !== 'new' && order.status !== 'seen'}
        >
          Подати заявку
        </button>
        {order.status === 'new' && (
          <button type="button" className="button" onClick={() => onAction('seen', order)}>
            Побачив
          </button>
        )}
      </div>
    </div>
  );
}

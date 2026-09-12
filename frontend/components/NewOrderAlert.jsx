'use client';

import { useState } from 'react';
import { useLive } from '@/lib/useLive';

const ALERT_BUDGET_CENTS = 5000; // до $50
const TOAST_MS = 6000;

function beep() {
  try {
    const ctx = new (window.AudioContext || window.webkitAudioContext)();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.frequency.value = 880;
    gain.gain.setValueAtTime(0.15, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.6);
    osc.start();
    osc.stop(ctx.currentTime + 0.6);
    osc.onended = () => ctx.close();
  } catch {
    // браузер може блокувати звук без взаємодії — не критично
  }
}

// Тости + звук на нові замовлення з бюджетом до $50.
// Клік по тосту веде прямо на замовлення.
export default function NewOrderAlert() {
  const [toasts, setToasts] = useState([]);

  useLive((event) => {
    if (event.type !== 'new') return;
    const budget = event.payload?.budget_cents;
    if (budget == null || budget > ALERT_BUDGET_CENTS) return;

    const id = `${event.payload?.order_id}-${Date.now()}`;
    setToasts((prev) => [
      ...prev,
      { id, url: event.payload?.url, title: event.payload?.title, budget },
    ]);
    beep();
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, TOAST_MS);
  });

  if (toasts.length === 0) return null;

  return (
    <div className="toast">
      {toasts.map((t) => (
        <a key={t.id} className="toast__item" href={t.url} target="_blank" rel="noreferrer">
          <span className="toast__budget">${(t.budget / 100).toFixed(0)}</span>
          <span className="toast__title">{t.title}</span>
        </a>
      ))}
    </div>
  );
}

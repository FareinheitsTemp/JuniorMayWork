'use client';

import { useEffect, useRef, useState } from 'react';

const WS_URL = 'ws://localhost:8080/ws';
const RECONNECT_MS = 2000;

// useLive(onEvent) — підписка на живі події бекенду.
// Повертає connected (true, поки WebSocket відкритий). Автоперепідключення.
export function useLive(onEvent) {
  const [connected, setConnected] = useState(false);
  const handlerRef = useRef(onEvent);
  handlerRef.current = onEvent;

  useEffect(() => {
    let ws;
    let closed = false;
    let retryTimer;

    function connect() {
      ws = new WebSocket(WS_URL);
      ws.onopen = () => setConnected(true);
      ws.onmessage = (msg) => {
        try {
          const event = JSON.parse(msg.data);
          handlerRef.current?.(event);
        } catch {
          // ігноруємо некоректні кадри
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (!closed) retryTimer = setTimeout(connect, RECONNECT_MS);
      };
      ws.onerror = () => ws.close();
    }

    connect();
    return () => {
      closed = true;
      clearTimeout(retryTimer);
      if (ws) ws.close();
    };
  }, []);

  return connected;
}

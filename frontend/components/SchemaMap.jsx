'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { radar } from '@/lib/radar';

const TABLES = {
  sources: { title: 'sources', fields: ['id PK', 'key', 'name', 'kind', 'enabled', 'last_success_at'] },
  source_channels: { title: 'source_channels', fields: ['id PK', 'source_id FK', 'handle', 'enabled'] },
  source_runs: { title: 'source_runs', fields: ['id PK', 'source_id FK', 'outcome', 'discovered_count', 'inserted_count'] },
  branches: { title: 'branches', fields: ['id PK', 'name', 'keywords[]', 'max_budget_cents', 'is_active'] },
  orders: { title: 'orders', fields: ['id PK', 'source + external_id', 'branch_id FK', 'priority_score', 'freshness_score', 'status'] },
  applications: { title: 'applications', fields: ['id PK', 'order_id FK', 'order_title', 'result', 'applied_at'] },
  events: { title: 'events', fields: ['id PK', 'order_id FK', 'type', 'payload JSONB', 'created_at'] },
  search_profiles: { title: 'search_profiles', fields: ['id PK', 'name', 'is_active', 'criteria JSONB', 'created_at'] },
  search_runs: { title: 'search_runs', fields: ['id PK', 'profile_id FK', 'status', 'stats JSONB', 'started_at / finished_at'] },
  reports: { title: 'reports', fields: ['id PK', 'search_run_id FK', 'file_name', 'storage_key', 'orders_total', 'summary_json'] },
  schema_nodes: { title: 'schema_nodes · UI meta', fields: ['id PK', 'layout_id FK', 'table_key', 'x / y', 'width / height'] },
};

const EDGES = [
  ['sources', 'source_channels', 'relation'], ['sources', 'source_runs', 'relation'],
  ['source_runs', 'orders', 'ingest'], ['branches', 'orders', 'relation'],
  ['orders', 'applications', 'relation'], ['orders', 'events', 'relation'],
  ['search_profiles', 'search_runs', 'relation'], ['search_runs', 'reports', 'relation'],
];

const CANVAS_W = 1320;
const CANVAS_H = 740;
const MIN_K = 0.25;
const MAX_K = 3;

function clampK(k) {
  return Math.max(MIN_K, Math.min(MAX_K, k));
}

function statusColor(status) {
  if (status === 'success' || status === 'ok') return '#3fb950';
  if (status === 'warning') return '#d29922';
  return '#f85149';
}

function edgePath(from, to) {
  const x1 = Number(from.x) + Number(from.width);
  const y1 = Number(from.y) + 44;
  const x2 = Number(to.x);
  const y2 = Number(to.y) + 44;
  const bend = Math.max(30, Math.min(90, Math.abs(y2 - y1) / 2 + 24));
  return `M ${x1} ${y1} C ${x1} ${y1 + bend}, ${x2} ${y2 - bend}, ${x2} ${y2}`;
}

export default function SchemaMap() {
  const [nodes, setNodes] = useState([]);
  const [runs, setRuns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [panning, setPanning] = useState(false);
  const [cam, setCam] = useState({ x: 0, y: 0, k: 1 });

  const nodesRef = useRef([]);
  const dragRef = useRef(null);
  const camRef = useRef(cam);
  const panRef = useRef(null);
  const canvasRef = useRef(null);
  const saveTimerRef = useRef(null);

  useEffect(() => { nodesRef.current = nodes; }, [nodes]);

  const loadLayout = useCallback(async () => {
    try {
      const layout = await radar.layout();
      setNodes(layout.nodes || []);
      const vp = layout.viewport;
      if (vp && typeof vp === 'object') {
        setCam({ x: Number(vp.x) || 0, y: Number(vp.y) || 0, k: clampK(Number(vp.k) || 1) });
      }
      setError('');
    } catch (err) {
      setError(`Карта недоступна: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadRuns = useCallback(async () => {
    try { setRuns(await radar.runs(30)); } catch (err) { setError(`Pipeline недоступний: ${err.message}`); }
  }, []);

  useEffect(() => {
    loadLayout();
    loadRuns();
    const timer = setInterval(loadRuns, 15000);
    return () => clearInterval(timer);
  }, [loadLayout, loadRuns]);

  const saveNode = useCallback(async (node) => {
    try {
      await fetch(`/api/schema/nodes/${node.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ x: Number(node.x), y: Number(node.y), width: Number(node.width), height: Number(node.height) }),
      });
    } catch { /* позиція вже відображена локально; збереження не критичне */ }
  }, []);

  const saveViewport = useCallback((next) => {
    clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(() => {
      fetch('/api/schema/layout', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ viewport: { x: next.x, y: next.y, k: next.k } }),
      }).catch(() => {});
    }, 600);
  }, []);

  const applyCam = useCallback((next) => {
    camRef.current = next;
    setCam(next);
    saveViewport(next);
  }, [saveViewport]);

  const zoomAt = useCallback((factor, cx, cy) => {
    const { x, y, k } = camRef.current;
    const nk = clampK(k * factor);
    if (nk === k) return;
    applyCam({ k: nk, x: cx - ((cx - x) / k) * nk, y: cy - ((cy - y) / k) * nk });
  }, [applyCam]);

  useEffect(() => {
    const el = canvasRef.current;
    if (!el) return;
    const onWheel = (event) => {
      event.preventDefault();
      const rect = el.getBoundingClientRect();
      const factor = event.deltaY < 0 ? 1.12 : 1 / 1.12;
      zoomAt(factor, event.clientX - rect.left, event.clientY - rect.top);
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, [zoomAt]);

  useEffect(() => {
    function move(event) {
      const drag = dragRef.current;
      if (!drag) return;
      const k = camRef.current.k;
      const x = Math.max(18, Math.min(1240, drag.startX + (event.clientX - drag.clientX) / k));
      const y = Math.max(18, Math.min(620, drag.startY + (event.clientY - drag.clientY) / k));
      setNodes((prev) => prev.map((node) => node.id === drag.id ? { ...node, x, y } : node));
    }
    function up() {
      const drag = dragRef.current;
      if (!drag) return;
      dragRef.current = null;
      const node = nodesRef.current.find((item) => item.id === drag.id);
      if (node) saveNode(node);
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    return () => { window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up); };
  }, [saveNode]);

  function startPan(event) {
    if (event.button !== 0) return;
    if (event.target.closest('.schema__node') || event.target.closest('button')) return;
    panRef.current = { clientX: event.clientX, clientY: event.clientY, x: camRef.current.x, y: camRef.current.y };
    setPanning(true);
    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function onPanMove(event) {
    const pan = panRef.current;
    if (!pan) return;
    applyCam({ ...camRef.current, x: pan.x + (event.clientX - pan.clientX), y: pan.y + (event.clientY - pan.clientY) });
  }

  function endPan() {
    panRef.current = null;
    setPanning(false);
  }

  function startDrag(event, node) {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    dragRef.current = { id: node.id, clientX: event.clientX, clientY: event.clientY, startX: Number(node.x), startY: Number(node.y) };
  }

  function zoomStep(factor) {
    const rect = canvasRef.current && canvasRef.current.getBoundingClientRect();
    zoomAt(factor, ((rect && rect.width) || CANVAS_W) / 2, ((rect && rect.height) || CANVAS_H) / 2);
  }

  function fitView() {
    const list = nodesRef.current;
    if (!list.length) { applyCam({ x: 0, y: 0, k: 1 }); return; }
    const minX = Math.min(...list.map((n) => Number(n.x)));
    const minY = Math.min(...list.map((n) => Number(n.y)));
    const maxX = Math.max(...list.map((n) => Number(n.x) + Number(n.width)));
    const maxY = Math.max(...list.map((n) => Number(n.y) + Number(n.height)));
    const rect = canvasRef.current && canvasRef.current.getBoundingClientRect();
    const w = ((rect && rect.width) || CANVAS_W) - 32;
    const h = ((rect && rect.height) || CANVAS_H) - 32;
    const k = clampK(Math.min(w / Math.max(maxX - minX, 1), h / Math.max(maxY - minY, 1)));
    applyCam({ k, x: 16 - minX * k, y: 16 - minY * k });
  }

  async function resetLayout() {
    const spaced = nodesRef.current.map((node, index) => ({
      ...node,
      x: 40 + (index % 3) * 430,
      y: 30 + Math.floor(index / 3) * 178,
    }));
    setNodes(spaced);
    applyCam({ x: 0, y: 0, k: 1 });
    await Promise.all(spaced.map(saveNode));
  }

  const byKey = {};
  nodes.forEach((node) => { byKey[node.table_key] = node; });
  const activeRun = runs.find((run) => run.status === 'running') || null;

  return (
    <div className="schema">
      <div className="schema__head">
        <div><h1 className="page__title">Карта бази даних</h1><p className="page__subtitle">Схема збору, класифікації та роботи із замовленнями.</p></div>
        <div className="schema__actions">
          <span className="schema__legend"><i className="schema__legend-dot schema__legend-dot--ok" /> success <i className="schema__legend-dot schema__legend-dot--warn" /> warning <i className="schema__legend-dot schema__legend-dot--bad" /> failure</span>
          <button className="button" type="button" onClick={() => zoomStep(1 / 1.25)}>−</button>
          <span className="schema__zoom-label">{Math.round(cam.k * 100)}%</span>
          <button className="button" type="button" onClick={() => zoomStep(1.25)}>+</button>
          <button className="button" type="button" onClick={fitView}>Fit</button>
          <button className="button" type="button" onClick={resetLayout}>Вирівняти карту</button>
        </div>
      </div>
      {error && <p className="schema__error">{error}</p>}
      <div
        ref={canvasRef}
        className="schema__canvas"
        style={{ position: 'relative', overflow: 'hidden', touchAction: 'none', userSelect: 'none', cursor: panning ? 'grabbing' : 'grab' }}
        onPointerDown={startPan}
        onPointerMove={onPanMove}
        onPointerUp={endPan}
        onPointerCancel={endPan}
      >
        {loading ? (
          <div style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', color: '#8b95a8' }}>Завантаження карти…</div>
        ) : (
          <div style={{ position: 'absolute', top: 0, left: 0, width: CANVAS_W, height: CANVAS_H, transform: `translate(${cam.x}px, ${cam.y}px) scale(${cam.k})`, transformOrigin: '0 0' }}>
            <svg className="schema__edges" viewBox={`0 0 ${CANVAS_W} ${CANVAS_H}`} style={{ position: 'absolute', top: 0, left: 0, width: CANVAS_W, height: CANVAS_H }}>
              <defs>
                <marker id="schema-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto"><path d="M0,0 L0,6 L9,3 z" fill="#546078" /></marker>
              </defs>
              {EDGES.map(([fromKey, toKey, kind]) => {
                const from = byKey[fromKey];
                const to = byKey[toKey];
                if (!from || !to) return null;
                const path = edgePath(from, to);
                const color = activeRun ? statusColor(activeRun.status) : '#61708d';
                const edgeColor = kind === 'ingest' ? color : '#61708d';
                return (
                  <g key={`${fromKey}-${toKey}`}>
                    <path className="schema__edge" d={path} markerEnd="url(#schema-arrow)" />
                    <path className="schema__edge-active" d={path} stroke={edgeColor} />
                    {kind === 'ingest' && activeRun && (
                      <circle r="5" fill={color} className="schema__particle"><animateMotion dur="2.4s" repeatCount="indefinite" path={path} /></circle>
                    )}
                  </g>
                );
              })}
            </svg>
            {nodes.map((node) => {
              const table = TABLES[node.table_key] || { title: node.table_key, fields: [] };
              return (
                <article
                  key={node.id}
                  className={`schema__node ${selected === node.table_key ? 'schema__node--selected' : ''}`}
                  style={{ left: Number(node.x), top: Number(node.y), width: Number(node.width), minHeight: Number(node.height), '--node-color': node.color, cursor: 'move' }}
                  onPointerDown={(event) => startDrag(event, node)}
                  onClick={() => setSelected(selected === node.table_key ? null : node.table_key)}
                >
                  <header className="schema__node-head"><span className="schema__node-grip">⠿</span><strong>{table.title}</strong><span className="schema__node-count">{table.fields.length}</span></header>
                  <ul className="schema__fields">{table.fields.map((field, index) => <li key={field} className={`schema__field ${index === 0 ? 'schema__field--key' : ''}`}>{field}</li>)}</ul>
                </article>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

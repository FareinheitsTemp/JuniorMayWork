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
  schema_nodes: { title: 'schema_nodes · UI meta', fields: ['id PK', 'layout_id FK', 'table_key', 'x / y', 'width / height'] },
};

const EDGES = [
  ['sources', 'source_channels', 'relation'], ['sources', 'source_runs', 'relation'],
  ['source_runs', 'orders', 'ingest'], ['branches', 'orders', 'relation'],
  ['orders', 'applications', 'relation'], ['orders', 'events', 'relation'],
];

const RESET = {
  sources: [52, 82], source_channels: [52, 390], source_runs: [385, 82], branches: [385, 390],
  orders: [735, 232], applications: [1080, 82], events: [1080, 400], schema_nodes: [735, 540],
};

function outcomeColor(run) {
  if (!run) return '#64748b';
  if (run.outcome === 'failure') return '#ef4444';
  if (run.outcome === 'warning' || run.outcome === 'running') return '#f59e0b';
  return '#3fcf8e';
}

function runLabel(run) {
  return ({ success: 'успішно', warning: 'попередження', failure: 'помилка', running: 'у процесі' })[run?.outcome] || 'очікує дані';
}

function port(node, target) {
  const x = Number(node.x), y = Number(node.y), w = Number(node.width), h = Number(node.height);
  const tx = Number(target.x) + Number(target.width) / 2, ty = Number(target.y) + Number(target.height) / 2;
  const cx = x + w / 2, cy = y + h / 2;
  if (Math.abs(tx - cx) >= Math.abs(ty - cy)) return tx >= cx ? [x + w, cy] : [x, cy];
  return ty >= cy ? [cx, y + h] : [cx, y];
}

function edgePath(from, to) {
  const [x1, y1] = port(from, to), [x2, y2] = port(to, from);
  const dx = Math.abs(x2 - x1), dy = Math.abs(y2 - y1);
  if (dx >= dy) {
    const bend = Math.max(55, dx * 0.42) * (x2 >= x1 ? 1 : -1);
    return `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`;
  }
  const bend = Math.max(55, dy * 0.42) * (y2 >= y1 ? 1 : -1);
  return `M ${x1} ${y1} C ${x1} ${y1 + bend}, ${x2} ${y2 - bend}, ${x2} ${y2}`;
}

export default function SchemaMap() {
  const [nodes, setNodes] = useState([]);
  const [runs, setRuns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const nodesRef = useRef([]);
  const dragRef = useRef(null);

  useEffect(() => { nodesRef.current = nodes; }, [nodes]);

  const loadLayout = useCallback(async () => {
    try {
      const layout = await radar.layout();
      setNodes(layout.nodes || []);
      setError('');
    } catch (err) { setError(`Карта недоступна: ${err.message}`); }
    finally { setLoading(false); }
  }, []);

  const loadRuns = useCallback(async () => {
    try { setRuns(await radar.runs(30)); } catch (err) { setError(`Pipeline недоступний: ${err.message}`); }
  }, []);

  useEffect(() => { loadLayout(); loadRuns(); const timer = setInterval(loadRuns, 15000); return () => clearInterval(timer); }, [loadLayout, loadRuns]);

  useEffect(() => {
    function move(event) {
      const drag = dragRef.current;
      if (!drag) return;
      const x = Math.max(18, Math.min(1240, drag.startX + event.clientX - drag.clientX));
      const y = Math.max(18, Math.min(620, drag.startY + event.clientY - drag.clientY));
      setNodes((prev) => prev.map((node) => node.id === drag.id ? { ...node, x, y } : node));
    }
    async function up() {
      const drag = dragRef.current;
      if (!drag) return;
      dragRef.current = null;
      const node = nodesRef.current.find((item) => item.id === drag.id);
      if (!node) return;
      try { await radar.moveNode(node.id, node); }
      catch (err) { setError(`Не зберегли позицію ${node.table_key}: ${err.message}`); }
    }
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    return () => { window.removeEventListener('pointermove', move); window.removeEventListener('pointerup', up); };
  }, []);

  function startDrag(event, node) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragRef.current = { id: node.id, clientX: event.clientX, clientY: event.clientY, startX: Number(node.x), startY: Number(node.y) };
    setSelected(node.table_key);
  }

  async function resetLayout() {
    const next = nodes.map((node) => {
      const [x, y] = RESET[node.table_key] || [80, 80];
      return { ...node, x, y };
    });
    setNodes(next);
    try { await Promise.all(next.map((node) => radar.moveNode(node.id, node))); setError(''); }
    catch (err) { setError(`Не вдалося скинути карту: ${err.message}`); }
  }

  const byKey = Object.fromEntries(nodes.map((node) => [node.table_key, node]));
  const activeRun = runs[0];
  const color = outcomeColor(activeRun);

  return (
    <section className="schema">
      <div className="schema__head">
        <div><h1 className="page__title">Карта бази даних</h1><p className="page__subtitle">Схема збору, класифікації та роботи із замовленнями.</p></div>
        <div className="schema__actions"><span className="schema__legend"><i className="schema__legend-dot schema__legend-dot--ok" /> success <i className="schema__legend-dot schema__legend-dot--warn" /> warning <i className="schema__legend-dot schema__legend-dot--bad" /> failure</span><button className="button" type="button" onClick={resetLayout}>Вирівняти карту</button></div>
      </div>
      {error && <div className="schema__error">{error}</div>}
      {loading && <div className="schema__loading">Завантажую схему…</div>}
      <div className="schema__viewport">
        <div className="schema__canvas">
          <svg className="schema__edges" viewBox="0 0 1320 740"><defs><marker id="schema-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto"><path d="M0,0 L0,6 L9,3 z" fill="#546078" /></marker></defs>
            {EDGES.map(([fromKey, toKey, kind]) => {
              const from = byKey[fromKey], to = byKey[toKey]; if (!from || !to) return null;
              const path = edgePath(from, to), edgeColor = kind === 'ingest' ? color : '#61708d';
              return <g key={`${fromKey}-${toKey}`}><path className="schema__edge" d={path} markerEnd="url(#schema-arrow)" /><path className="schema__edge-active" d={path} stroke={edgeColor} />{kind === 'ingest' && activeRun && <circle r="5" fill={color} className="schema__particle"><animateMotion dur="2.4s" repeatCount="indefinite" path={path} /></circle>}</g>;
            })}
          </svg>
          {nodes.map((node) => {
            const table = TABLES[node.table_key] || { title: node.table_key, fields: [] };
            return <article key={node.id} className={`schema__node ${selected === node.table_key ? 'schema__node--selected' : ''}`} style={{ left: Number(node.x), top: Number(node.y), width: Number(node.width), minHeight: Number(node.height), '--node-color': node.color }} onPointerDown={(event) => startDrag(event, node)}><header className="schema__node-head"><span className="schema__node-grip">⠿</span><strong>{table.title}</strong><span className="schema__node-count">{table.fields.length}</span></header><ul className="schema__fields">{table.fields.map((field, index) => <li key={field} className={`schema__field ${index === 0 ? 'schema__field--key' : ''}`}>{field}</li>)}</ul></article>;
          })}
        </div>
      </div>
      <aside className="schema__inspector card"><div className="schema__inspector-head"><h2 className="feed__title">Pipeline inspector</h2><span className="schema__status" style={{ color }}>{runLabel(activeRun)}</span></div>{runs.length === 0 ? <p className="page__hint">Ще не було запусків джерел.</p> : <ul className="schema__runs">{runs.slice(0, 8).map((run) => <li key={run.id} className="schema__run"><i className="schema__run-dot" style={{ background: outcomeColor(run) }} /><span>Джерело #{run.source_id}</span><span>{run.discovered_count} знайдено</span><span>{run.inserted_count} нових</span>{run.error_message && <span className="schema__run-error">{run.error_message}</span>}</li>)}</ul>}</aside>
    </section>
  );
}

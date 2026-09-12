'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { radar } from '@/lib/radar';

const TABLES = {
  sources: {
    title: 'sources',
    fields: ['id PK', 'key', 'name', 'kind', 'enabled', 'last_success_at'],
  },
  source_channels: {
    title: 'source_channels',
    fields: ['id PK', 'source_id FK', 'handle', 'enabled'],
  },
  source_runs: {
    title: 'source_runs',
    fields: ['id PK', 'source_id FK', 'outcome', 'discovered_count', 'inserted_count'],
  },
  branches: {
    title: 'branches',
    fields: ['id PK', 'name', 'keywords[]', 'max_budget_cents', 'is_active'],
  },
  orders: {
    title: 'orders',
    fields: ['id PK', 'source + external_id', 'branch_id FK', 'priority_score', 'freshness_score', 'status'],
  },
  applications: {
    title: 'applications',
    fields: ['id PK', 'order_id FK', 'order_title', 'result', 'applied_at'],
  },
  events: {
    title: 'events',
    fields: ['id PK', 'order_id FK', 'type', 'payload JSONB', 'created_at'],
  },
  schema_nodes: {
    title: 'schema_nodes',
    fields: ['id PK', 'layout_id FK', 'table_key', 'x / y', 'width / height'],
  },
};

const EDGES = [
  ['sources', 'source_channels', 'one-to-many'],
  ['sources', 'source_runs', 'one-to-many'],
  ['source_runs', 'orders', 'ingest'],
  ['branches', 'orders', 'classification'],
  ['orders', 'applications', 'one-to-many'],
  ['orders', 'events', 'one-to-many'],
  ['schema_nodes', 'orders', 'layout'],
];

function pipelineColor(runs) {
  const latest = runs[0];
  if (!latest) return '#64748b';
  if (latest.outcome === 'failure') return '#ef4444';
  if (latest.outcome === 'warning' || latest.outcome === 'running') return '#f59e0b';
  return '#3fcf8e';
}

function runLabel(run) {
  if (run.outcome === 'success') return 'успішно';
  if (run.outcome === 'warning') return 'попередження';
  if (run.outcome === 'failure') return 'помилка';
  return 'у процесі';
}

// Справжня ERD-мапа: ноди можна тягати, їхня позиція зберігається в schema_nodes.
export default function SchemaMap() {
  const [nodes, setNodes] = useState([]);
  const [runs, setRuns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const dragRef = useRef(null);

  const load = useCallback(async () => {
    try {
      const [layout, sourceRuns] = await Promise.all([radar.layout(), radar.runs(30)]);
      setNodes(layout.nodes || []);
      setRuns(sourceRuns || []);
      setError('');
    } catch (err) {
      setError(`Карта недоступна: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = setInterval(load, 15000);
    return () => clearInterval(timer);
  }, [load]);

  useEffect(() => {
    function move(event) {
      const drag = dragRef.current;
      if (!drag) return;
      const x = Math.max(0, drag.startX + event.clientX - drag.clientX);
      const y = Math.max(0, drag.startY + event.clientY - drag.clientY);
      setNodes((prev) => prev.map((n) => (n.id === drag.id ? { ...n, x, y } : n)));
    }

    async function up() {
      const drag = dragRef.current;
      if (!drag) return;
      dragRef.current = null;
      const node = nodes.find((n) => n.id === drag.id);
      if (!node) return;
      try {
        await radar.moveNode(node.id, node);
      } catch (err) {
        setError(`Не зберегли позицію ${node.table_key}: ${err.message}`);
      }
    }

    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    return () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
    };
  }, [nodes]);

  function startDrag(event, node) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragRef.current = {
      id: node.id,
      clientX: event.clientX,
      clientY: event.clientY,
      startX: Number(node.x),
      startY: Number(node.y),
    };
    setSelected(node.table_key);
  }

  const byKey = Object.fromEntries(nodes.map((n) => [n.table_key, n]));
  const color = pipelineColor(runs);
  const activeRun = runs[0];

  function edgePath(fromKey, toKey) {
    const from = byKey[fromKey];
    const to = byKey[toKey];
    if (!from || !to) return '';
    const x1 = Number(from.x) + Number(from.width);
    const y1 = Number(from.y) + Number(from.height) / 2;
    const x2 = Number(to.x);
    const y2 = Number(to.y) + Number(to.height) / 2;
    const middle = x1 + (x2 - x1) / 2;
    return `M ${x1} ${y1} C ${middle} ${y1}, ${middle} ${y2}, ${x2} ${y2}`;
  }

  return (
    <section className="schema">
      <div className="schema__head">
        <div>
          <h1 className="page__title">Карта бази даних</h1>
          <p className="page__subtitle">Перетягуй таблиці — позиції зберігаються у твоїй БД.</p>
        </div>
        <div className="schema__legend">
          <span><i className="schema__legend-dot schema__legend-dot--ok" /> успішний потік</span>
          <span><i className="schema__legend-dot schema__legend-dot--warn" /> попередження</span>
          <span><i className="schema__legend-dot schema__legend-dot--bad" /> помилка</span>
        </div>
      </div>

      {error && <div className="schema__error">{error}</div>}
      {loading && <div className="schema__loading">Завантажую схему…</div>}

      <div className="schema__canvas">
        <svg className="schema__edges" viewBox="0 0 1500 850" preserveAspectRatio="none">
          <defs>
            <marker id="schema-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto">
              <path d="M0,0 L0,6 L9,3 z" fill="#546078" />
            </marker>
          </defs>
          {EDGES.map(([from, to, kind]) => {
            const path = edgePath(from, to);
            if (!path) return null;
            const edgeColor = kind === 'ingest' ? color : '#546078';
            return (
              <g key={`${from}-${to}`}>
                <path className="schema__edge" d={path} markerEnd="url(#schema-arrow)" />
                <path className="schema__edge-active" d={path} stroke={edgeColor} />
                {kind === 'ingest' && activeRun && (
                  <circle r="6" fill={color} className="schema__particle">
                    <animateMotion dur="2.4s" repeatCount="indefinite" path={path} />
                  </circle>
                )}
              </g>
            );
          })}
        </svg>

        {nodes.map((node) => {
          const table = TABLES[node.table_key] || { title: node.table_key, fields: [] };
          const isSelected = selected === node.table_key;
          return (
            <article
              key={node.id}
              className={`schema__node ${isSelected ? 'schema__node--selected' : ''}`}
              style={{
                left: Number(node.x),
                top: Number(node.y),
                width: Number(node.width),
                minHeight: Number(node.height),
                '--node-color': node.color,
              }}
              onPointerDown={(event) => startDrag(event, node)}
            >
              <header className="schema__node-head">
                <span className="schema__node-grip">⠿</span>
                <strong>{table.title}</strong>
                <span className="schema__node-count">{table.fields.length}</span>
              </header>
              <ul className="schema__fields">
                {table.fields.map((field, index) => (
                  <li key={field} className={index === 0 ? 'schema__field schema__field--key' : 'schema__field'}>
                    {field}
                  </li>
                ))}
              </ul>
            </article>
          );
        })}
      </div>

      <aside className="schema__inspector card">
        <div className="schema__inspector-head">
          <h2 className="feed__title">Pipeline inspector</h2>
          <span className="schema__status" style={{ color }}>{activeRun ? runLabel(activeRun) : 'очікує дані'}</span>
        </div>
        {runs.length === 0 && <p className="page__hint">Ще не було запусків джерел.</p>}
        <ul className="schema__runs">
          {runs.slice(0, 8).map((run) => (
            <li key={run.id} className="schema__run">
              <i className="schema__run-dot" style={{ background: run.outcome === 'failure' ? '#ef4444' : run.outcome === 'warning' ? '#f59e0b' : '#3fcf8e' }} />
              <span>source #{run.source_id}</span>
              <span>{run.discovered_count} знайдено</span>
              <span>{run.inserted_count} нових</span>
              {run.error_message && <span className="schema__run-error">{run.error_message}</span>}
            </li>
          ))}
        </ul>
      </aside>
    </section>
  );
}

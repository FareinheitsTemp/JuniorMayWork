'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { radar } from '@/lib/radar';

// Claude Code syntax palette for Schema Map
// Primary entities: terracotta, Lookups: amber, Relations: slate blue, Meta/Telemetry: sage green
const TABLES = {
  sources: { title: 'sources', fields: ['id PK', 'key', 'name', 'kind', 'enabled', 'last_success_at'], color: '#788c5d' },
  source_channels: { title: 'source_channels', fields: ['id PK', 'source_id FK', 'handle', 'enabled'], color: '#788c5d' },
  source_runs: { title: 'source_runs', fields: ['id PK', 'source_id FK', 'outcome', 'discovered_count', 'inserted_count'], color: '#788c5d' },
  branches: { title: 'branches', fields: ['id PK', 'name', 'keywords[]', 'max_budget_cents', 'is_active'], color: '#d97757' },
  orders: { title: 'orders', fields: ['id PK', 'source + external_id', 'branch_id FK', 'status_id FK', 'priority_score', 'freshness_score'], color: '#d97757' },
  applications: { title: 'applications', fields: ['id PK', 'order_id FK', 'result', 'applied_at'], color: '#d97757' },
  events: { title: 'events', fields: ['id PK', 'order_id FK', 'type', 'payload JSONB', 'created_at'], color: '#b0aea5' },
  order_statuses: { title: 'order_statuses · lookup', fields: ['id PK', 'key', 'label', 'color', 'is_terminal'], color: '#d4a017' },
  order_status_history: { title: 'order_status_history', fields: ['id PK', 'order_id FK', 'from/to_status FK', 'actor', 'changed_at'], color: '#b0aea5' },
  order_notes: { title: 'order_notes', fields: ['id PK', 'order_id FK', 'body', 'created_at'], color: '#b0aea5' },
  order_skills: { title: 'order_skills · M:N', fields: ['order_id FK', 'skill_id FK'], color: '#6b8fa3' },
  skills: { title: 'skills · lookup', fields: ['id PK', 'name', 'slug'], color: '#d4a017' },
  application_results: { title: 'application_results · lookup', fields: ['id PK', 'key', 'label'], color: '#d4a017' },
  tags: { title: 'tags · lookup', fields: ['id PK', 'name'], color: '#d4a017' },
  search_profiles: { title: 'search_profiles', fields: ['id PK', 'name', 'is_active', 'criteria JSONB', 'interval / budget / age'], color: '#d97757' },
  search_profile_tags: { title: 'search_profile_tags', fields: ['id PK', 'profile_id FK', 'tag', 'tag_type'], color: '#6b8fa3' },
  search_profile_sources: { title: 'search_profile_sources', fields: ['profile_id FK', 'source_id FK', 'is_enabled'], color: '#6b8fa3' },
  search_runs: { title: 'search_runs', fields: ['id PK', 'profile_id FK', 'status', 'stats JSONB', 'started_at / finished_at'], color: '#d97757' },
  order_discoveries: { title: 'order_discoveries', fields: ['id PK', 'search_run_id FK', 'order_id FK', 'relevance_score', 'captured_at'], color: '#6b8fa3' },
  reports: { title: 'reports', fields: ['id PK', 'search_run_id FK', 'report_type', 'file_name', 'orders_total', 'summary_json'], color: '#d97757' },
  report_downloads: { title: 'report_downloads', fields: ['id PK', 'report_id FK', 'downloaded_at', 'user_agent'], color: '#788c5d' },
  search_run_daily_stats: { title: 'search_run_daily_stats', fields: ['run_date + source_id', 'orders_found', 'runs_count'], color: '#788c5d' },
  audit_log: { title: 'audit_log', fields: ['id PK', 'entity + entity_id', 'action', 'payload JSONB', 'created_at'], color: '#788c5d' },
  schema_nodes: { title: 'schema_nodes · UI meta', fields: ['id PK', 'layout_id FK', 'table_key', 'x / y', 'width / height'], color: '#6c6a64' },
};

const EDGES = [
  ['sources', 'source_channels', 'relation'], ['sources', 'source_runs', 'relation'],
  ['source_runs', 'orders', 'ingest'], ['branches', 'orders', 'relation'],
  ['order_statuses', 'orders', 'relation'],
  ['orders', 'applications', 'relation'], ['orders', 'events', 'relation'],
  ['orders', 'order_notes', 'relation'], ['orders', 'order_status_history', 'relation'],
  ['orders', 'order_skills', 'relation'], ['skills', 'order_skills', 'relation'],
  ['search_profiles', 'search_profile_tags', 'relation'],
  ['search_profiles', 'search_profile_sources', 'relation'], ['sources', 'search_profile_sources', 'relation'],
  ['search_profiles', 'search_runs', 'relation'], ['search_runs', 'order_discoveries', 'relation'],
  ['orders', 'order_discoveries', 'relation'],
  ['search_runs', 'reports', 'relation'], ['reports', 'report_downloads', 'relation'],
  ['sources', 'search_run_daily_stats', 'relation'],
];

const CANVAS_W = 1920;
const CANVAS_H = 1080;
const MIN_K = 0.25;
const MAX_K = 2.5;

function clampK(k) {
  return Math.max(MIN_K, Math.min(MAX_K, k));
}

function edgePath(from, to) {
  const x1 = Number(from.x) + Number(from.width);
  const y1 = Number(from.y) + 38;
  const x2 = Number(to.x);
  const y2 = Number(to.y) + 38;
  const bend = Math.max(30, Math.min(90, Math.abs(y2 - y1) / 2 + 24));
  return `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`;
}

export default function SchemaMap() {
  const [nodes, setNodes] = useState([]);
  const [runs, setRuns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [panning, setPanning] = useState(false);
  const [cam, setCam] = useState({ x: 0, y: 0, k: 0.85 });

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
        setCam({ x: Number(vp.x) || 0, y: Number(vp.y) || 0, k: clampK(Number(vp.k) || 0.85) });
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
    } catch { /* позиція збережена локально */ }
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
      const x = Math.max(18, Math.min(1860, drag.startX + (event.clientX - drag.clientX) / k));
      const y = Math.max(18, Math.min(1020, drag.startY + (event.clientY - drag.clientY) / k));
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
    zoomAt(factor, ((rect && rect.width) || 1200) / 2, ((rect && rect.height) || 700) / 2);
  }

  function fitView() {
    const list = nodesRef.current;
    if (!list.length) { applyCam({ x: 0, y: 0, k: 0.85 }); return; }
    const minX = Math.min(...list.map((n) => Number(n.x)));
    const minY = Math.min(...list.map((n) => Number(n.y)));
    const maxX = Math.max(...list.map((n) => Number(n.x) + Number(n.width)));
    const maxY = Math.max(...list.map((n) => Number(n.y) + Number(n.height)));
    const rect = canvasRef.current && canvasRef.current.getBoundingClientRect();
    const w = ((rect && rect.width) || 1200) - 40;
    const h = ((rect && rect.height) || 700) - 40;
    const k = clampK(Math.min(w / Math.max(maxX - minX, 1), h / Math.max(maxY - minY, 1)));
    applyCam({ k, x: 20 - minX * k, y: 20 - minY * k });
  }

  const byKey = {};
  nodes.forEach((node) => { byKey[node.table_key] = node; });
  const activeRun = runs.find((run) => run.status === 'running') || null;

  return (
    <section className="page" style={{ maxWidth: '100%', padding: '16px 20px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontSize: 13, fontWeight: 600 }}>›_</span>
            <h1 className="page__title">ERD Schema Visualizer</h1>
          </div>
          <p className="page__subtitle" style={{ margin: '2px 0 0' }}>
            24 таблиці v2-архітектури: зв'язки, PK/FK індекси, довідники, аудит та телеметрія.
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button className="button button--sm" type="button" onClick={() => zoomStep(1 / 1.25)}>−</button>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, minWidth: 44, textAlign: 'center', color: 'var(--text-dim)' }}>
            {Math.round(cam.k * 100)}%
          </span>
          <button className="button button--sm" type="button" onClick={() => zoomStep(1.25)}>+</button>
          <button className="button button--sm" type="button" onClick={fitView}>Fit View</button>
        </div>
      </div>

      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}

      <div
        ref={canvasRef}
        style={{
          position: 'relative',
          height: '76vh',
          minHeight: 560,
          background: 'var(--bg-soft)',
          backgroundImage: 'radial-gradient(var(--line) 1px, transparent 1px)',
          backgroundSize: '24px 24px',
          border: '1px solid var(--line)',
          borderRadius: 'var(--radius)',
          overflow: 'hidden',
          touchAction: 'none',
          userSelect: 'none',
          cursor: panning ? 'grabbing' : 'grab',
        }}
        onPointerDown={startPan}
        onPointerMove={onPanMove}
        onPointerUp={endPan}
        onPointerCancel={endPan}
      >
        {loading ? (
          <div style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>
            LOAD_SCHEMA_NODES...
          </div>
        ) : (
          <div style={{ position: 'absolute', top: 0, left: 0, width: CANVAS_W, height: CANVAS_H, transform: `translate(${cam.x}px, ${cam.y}px) scale(${cam.k})`, transformOrigin: '0 0' }}>
            <svg style={{ position: 'absolute', inset: 0, width: CANVAS_W, height: CANVAS_H, pointerEvents: 'none' }}>
              <defs>
                <marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
                  <path d="M0,0 L0,6 L7,3 z" fill="var(--line-light)" />
                </marker>
              </defs>
              {EDGES.map(([fromKey, toKey, kind]) => {
                const from = byKey[fromKey];
                const to = byKey[toKey];
                if (!from || !to) return null;
                const path = edgePath(from, to);
                const strokeColor = kind === 'ingest' ? 'var(--good)' : 'var(--line-light)';
                return (
                  <path
                    key={`${fromKey}-${toKey}`}
                    d={path}
                    fill="none"
                    stroke={strokeColor}
                    strokeWidth={kind === 'ingest' ? 2 : 1.25}
                    strokeDasharray={kind === 'ingest' ? '4 4' : 'none'}
                    markerEnd="url(#arrow)"
                    opacity={0.65}
                  />
                );
              })}
            </svg>

            {nodes.map((node) => {
              const meta = TABLES[node.table_key] || { title: node.table_key, fields: [], color: 'var(--accent)' };
              const isSel = selected === node.table_key;
              return (
                <div
                  key={node.id}
                  style={{
                    position: 'absolute',
                    left: Number(node.x),
                    top: Number(node.y),
                    width: Number(node.width) || 220,
                    background: 'var(--card)',
                    border: `1px solid ${isSel ? 'var(--accent)' : 'var(--line)'}`,
                    borderRadius: 'var(--radius)',
                    boxShadow: isSel ? '0 0 0 2px var(--accent-soft)' : 'none',
                    cursor: 'move',
                    fontFamily: 'var(--font-mono)',
                  }}
                  onPointerDown={(e) => startDrag(e, node)}
                  onClick={() => setSelected(isSel ? null : node.table_key)}
                >
                  <div style={{
                    padding: '6px 10px',
                    borderBottom: '1px solid var(--line)',
                    background: 'var(--bg-soft)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, fontWeight: 600 }}>
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: meta.color }} />
                      <span style={{ color: 'var(--text)' }}>{meta.title}</span>
                    </div>
                    <span style={{ fontSize: 10, color: 'var(--text-muted)' }}>{meta.fields.length}</span>
                  </div>

                  <div style={{ padding: '6px 10px', display: 'grid', gap: 3 }}>
                    {meta.fields.map((f, i) => {
                      const isPk = f.includes('PK');
                      const isFk = f.includes('FK');
                      return (
                        <div key={f} style={{
                          fontSize: 11,
                          display: 'flex',
                          justifyContent: 'space-between',
                          color: isPk ? 'var(--warn)' : isFk ? 'var(--accent)' : 'var(--text-dim)',
                        }}>
                          <span>{f}</span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}

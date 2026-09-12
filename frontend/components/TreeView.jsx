'use client';

import { useState } from 'react';
import LeafCard from './LeafCard';

// Дерево: вітки (ніші) → листочки (замовлення).
// Замовлення без вітки потрапляють у групу «Без вітки».
export default function TreeView({ branches, orders, onAction }) {
  const [collapsed, setCollapsed] = useState({});

  const groups = branches.map((b) => ({
    key: b.id,
    name: b.name,
    maxBudget: b.max_budget_cents,
    leaves: orders.filter((o) => o.branch_id === b.id),
  }));
  const unsorted = orders.filter((o) => o.branch_id == null);
  if (unsorted.length > 0) {
    groups.push({ key: 'none', name: 'Без вітки', maxBudget: null, leaves: unsorted });
  }

  function toggle(key) {
    setCollapsed((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  return (
    <div className="tree">
      {groups.map((g) => {
        const isCollapsed = collapsed[g.key] && g.leaves.length > 0;
        return (
          <div key={g.key} className="tree__branch">
            <button type="button" className="tree__branch-head" onClick={() => toggle(g.key)}>
              <span className={`tree__toggle ${isCollapsed ? 'tree__toggle--closed' : ''}`}>▾</span>
              <span className="tree__branch-name">{g.name}</span>
              <span className="tree__branch-count">{g.leaves.length}</span>
              {g.maxBudget != null && (
                <span className="tree__branch-limit">до ${(g.maxBudget / 100).toFixed(0)}</span>
              )}
            </button>
            {!isCollapsed && (
              <div className="tree__leaves">
                {g.leaves.length === 0 && <div className="tree__empty">листочків поки немає</div>}
                {g.leaves.map((o) => (
                  <LeafCard key={o.id} order={o} onAction={onAction} />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

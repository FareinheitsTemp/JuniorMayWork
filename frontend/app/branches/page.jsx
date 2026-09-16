'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { TerminalIcon, PlusIcon, EditIcon, TrashIcon, CloseIcon } from '@/components/Icons';
import '@/styles/blocks/grid-extra.scss';

export default function BranchesPage() {
  const [branches, setBranches] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [flash, setFlash] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [busy, setBusy] = useState(false);

  const [form, setForm] = useState({
    name: '',
    keywords: '',
    max_budget: '',
    is_active: true,
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.branches();
      setBranches(data || []);
      setError('');
    } catch (err) {
      setError(`Вітки недоступні: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function startCreate() {
    setEditingId(null);
    setForm({ name: '', keywords: '', max_budget: '', is_active: true });
    setShowModal(true);
  }

  function startEdit(b) {
    setEditingId(b.id);
    setForm({
      name: b.name || '',
      keywords: (b.keywords || []).join(', '),
      max_budget: b.max_budget_cents ? (b.max_budget_cents / 100).toString() : '',
      is_active: b.is_active ?? true,
    });
    setShowModal(true);
  }

  function closeModal() {
    setShowModal(false);
    setEditingId(null);
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.name.trim()) return;

    setBusy(true);
    const data = {
      name: form.name.trim(),
      keywords: form.keywords
        .split(',')
        .map((k) => k.trim())
        .filter(Boolean),
      max_budget_cents: form.max_budget ? Math.round(Number(form.max_budget) * 100) : null,
      is_active: form.is_active,
    };

    try {
      if (editingId == null) {
        await api.createBranch(data);
        setFlash(`Вітку «${form.name}» створено`);
      } else {
        await api.updateBranch(editingId, data);
        setFlash(`Вітку «${form.name}» оновлено`);
      }
      closeModal();
      await load();
    } catch (err) {
      setError(`Не вдалося зберегти: ${err.message}`);
    } finally {
      setBusy(false);
    }
  }

  async function removeBranch(b) {
    if (!window.confirm(`Видалити вітку «${b.name}»?`)) return;
    try {
      await api.deleteBranch(b.id);
      setFlash(`Вітку «${b.name}» видалено`);
      await load();
    } catch (err) {
      setError(`Не вдалося видалити: ${err.message}`);
    }
  }

  return (
    <section className="page">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <TerminalIcon size={18} style={{ color: 'var(--accent)' }} />
            <h1 className="page__title">Вітки (Branches Registry)</h1>
          </div>
          <p className="page__subtitle">Ніші пошуку, ключові слова для класифікації та бюджети вакансій.</p>
        </div>
        <button className="button button--primary" type="button" onClick={startCreate}>
          <PlusIcon size={14} />
          <span>Створити вітку</span>
        </button>
      </div>

      {error && <div className="toast" onClick={() => setError('')}>{error}</div>}
      {flash && (
        <div className="report-banner" style={{ marginBottom: 16 }}>
          <strong style={{ color: 'var(--accent)' }}>SUCCESS:</strong>
          <span>{flash}</span>
          <button className="button button--sm" type="button" style={{ marginLeft: 'auto' }} onClick={() => setFlash('')}>✕</button>
        </div>
      )}

      <div className="grid__scroll">
        {loading ? (
          <div className="skeleton-list" style={{ padding: 16 }}>
            <div className="skeleton" />
            <div className="skeleton" />
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: 60 }}>id</th>
                <th>Назва вітки</th>
                <th>Ключові слова (Keywords)</th>
                <th>Ліміт бюджету</th>
                <th>Статус</th>
                <th style={{ textAlign: 'right' }}>Дії</th>
              </tr>
            </thead>
            <tbody>
              {branches.map((b) => (
                <tr key={b.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-muted)' }}>#{b.id}</td>
                  <td>
                    <strong style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{b.name}</strong>
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                      {(b.keywords || []).map((kw) => (
                        <span key={kw} style={{ fontFamily: 'var(--font-mono)', fontSize: 11, padding: '2px 6px', background: 'var(--bg-soft)', borderRadius: 'var(--radius-sm)', color: 'var(--text-dim)' }}>
                          {kw}
                        </span>
                      ))}
                      {(!b.keywords || b.keywords.length === 0) && <span style={{ color: 'var(--text-muted)' }}>—</span>}
                    </div>
                  </td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>
                    {b.max_budget_cents ? `$${(b.max_budget_cents / 100).toFixed(0)}` : '—'}
                  </td>
                  <td>
                    <span className={`status-dot status-dot--${b.is_active ? 'ok' : 'idle'}`} />
                    <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>
                      {b.is_active ? 'ACTIVE' : 'DISABLED'}
                    </span>
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <div className="row-actions">
                      <button className="button button--sm" type="button" onClick={() => startEdit(b)}>
                        <EditIcon size={12} />
                        <span>Редагувати</span>
                      </button>
                      <button className="button button--danger button--sm" type="button" onClick={() => removeBranch(b)}>
                        <TrashIcon size={12} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {branches.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty-hint">Віток ще немає — створіть першу нішу.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal-card" onClick={(e) => e.stopPropagation()}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--line)', paddingBottom: 10, marginBottom: 14 }}>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--accent)' }}>
                {editingId == null ? 'BRANCH_CREATE' : `BRANCH_UPDATE #${editingId}`}
              </span>
              <button className="button button--sm" type="button" onClick={closeModal}>
                <CloseIcon size={12} />
              </button>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="field">
                <label className="field__label">Назва ніші</label>
                <input
                  className="input"
                  required
                  placeholder="наприклад: Go / Backend"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>

              <div className="field">
                <label className="field__label">Ключові слова (через кому)</label>
                <input
                  className="input"
                  placeholder="golang, postgres, microservices"
                  value={form.keywords}
                  onChange={(e) => setForm({ ...form, keywords: e.target.value })}
                />
              </div>

              <div className="field">
                <label className="field__label">Макс. бюджет ($)</label>
                <input
                  className="input"
                  type="number"
                  placeholder="наприклад: 1500"
                  value={form.max_budget}
                  onChange={(e) => setForm({ ...form, max_budget: e.target.value })}
                />
              </div>

              <label className="field field--checkbox">
                <input
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                />
                <span>Активна вітка (використовується для збору)</span>
              </label>

              <div className="modal-card__actions">
                <button className="button" type="button" onClick={closeModal}>Скасувати</button>
                <button className="button button--primary" type="submit" disabled={busy}>
                  {editingId == null ? 'Створити' : 'Зберегти'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </section>
  );
}

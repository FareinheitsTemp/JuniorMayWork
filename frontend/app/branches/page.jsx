'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';

const EMPTY_FORM = { name: '', keywords: '', max_budget: 100, is_active: true };

// Вітки: повний CRUD (name, ключові слова, ліміт бюджету, активність).
export default function BranchesPage() {
  const [branches, setBranches] = useState([]);
  const [form, setForm] = useState(EMPTY_FORM);
  const [editingId, setEditingId] = useState(null);
  const [flash, setFlash] = useState('');

  const load = useCallback(async () => {
    try {
      setBranches((await api.branches()) || []);
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function startEdit(b) {
    setEditingId(b.id);
    setForm({
      name: b.name,
      keywords: (b.keywords || []).join(', '),
      max_budget: b.max_budget_cents / 100,
      is_active: b.is_active,
    });
  }

  function reset() {
    setEditingId(null);
    setForm(EMPTY_FORM);
  }

  async function submit(e) {
    e.preventDefault();
    const data = {
      name: form.name,
      keywords: form.keywords.split(',').map((s) => s.trim()).filter(Boolean),
      max_budget_cents: Math.round(Number(form.max_budget) * 100),
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
      reset();
      await load();
    } catch (err) {
      setFlash(`Помилка: ${err.message}`);
    }
  }

  async function remove(b) {
    if (!window.confirm(`Видалити вітку «${b.name}»? Замовлення залишаться, але втратять вітку.`)) return;
    try {
      await api.deleteBranch(b.id);
      await load();
    } catch (e) {
      setFlash(`Помилка: ${e.message}`);
    }
  }

  return (
    <section className="page">
      <h1 className="page__title">Вітки</h1>
      {flash && <div className="page__hint">{flash}</div>}

      <form className="card" onSubmit={submit}>
        <h2 className="feed__title">{editingId == null ? 'Нова вітка' : `Редагування вітки #${editingId}`}</h2>
        <div className="page__row">
          <div className="field">
            <label className="field__label" htmlFor="b-name">Назва</label>
            <input
              id="b-name"
              className="input"
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="React-гігі"
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="b-keywords">Ключові слова (через кому)</label>
            <input
              id="b-keywords"
              className="input"
              value={form.keywords}
              onChange={(e) => setForm({ ...form, keywords: e.target.value })}
              placeholder="react, next.js, javascript"
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="b-budget">Ліміт, $</label>
            <input
              id="b-budget"
              className="input"
              type="number"
              min="1"
              value={form.max_budget}
              onChange={(e) => setForm({ ...form, max_budget: e.target.value })}
            />
          </div>
          <label className="field" style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
            <input
              type="checkbox"
              checked={form.is_active}
              onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
            />
            <span className="field__label" style={{ textTransform: 'none' }}>активна</span>
          </label>
          <button type="submit" className="button button--primary">
            {editingId == null ? 'Створити' : 'Зберегти'}
          </button>
          {editingId != null && (
            <button type="button" className="button" onClick={reset}>Скасувати</button>
          )}
        </div>
      </form>

      <div className="card">
        <table className="table">
          <thead>
            <tr>
              <th>Назва</th>
              <th>Ключові слова</th>
              <th>Ліміт</th>
              <th>Активна</th>
              <th>Дії</th>
            </tr>
          </thead>
          <tbody>
            {branches.length === 0 && (
              <tr>
                <td colSpan={5} className="page__hint">Віток поки немає — створи першу.</td>
              </tr>
            )}
            {branches.map((b) => (
              <tr key={b.id}>
                <td>{b.name}</td>
                <td>{(b.keywords || []).join(', ')}</td>
                <td>${(b.max_budget_cents / 100).toFixed(0)}</td>
                <td>{b.is_active ? 'так' : 'ні'}</td>
                <td>
                  <div className="leaf__actions">
                    <button type="button" className="button" onClick={() => startEdit(b)}>
                      Редагувати
                    </button>
                    <button type="button" className="button button--danger" onClick={() => remove(b)}>×</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

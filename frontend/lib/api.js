// API-клієнт: усі запити йдуть відносними /api/* (next.config.mjs проксує на :8080).

async function request(path, options = {}) {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body && body.error) message = body.error;
    } catch {
      // без тіла відповіді
    }
    throw new Error(message);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  branches: () => request('/branches'),
  createBranch: (data) => request('/branches', { method: 'POST', body: JSON.stringify(data) }),
  updateBranch: (id, data) => request(`/branches/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteBranch: (id) => request(`/branches/${id}`, { method: 'DELETE' }),

  orders: (params = {}) => {
    const qs = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') qs.set(key, value);
    });
    const query = qs.toString();
    return request(`/orders${query ? `?${query}` : ''}`);
  },
  updateOrder: (id, patch) => request(`/orders/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  deleteOrder: (id) => request(`/orders/${id}`, { method: 'DELETE' }),
  apply: (id, note = '') => request(`/orders/${id}/apply`, { method: 'POST', body: JSON.stringify({ note }) }),

  events: (limit = 100) => request(`/events?limit=${limit}`),
  applications: (limit = 100) => request(`/applications?limit=${limit}`),
  setApplicationResult: (id, result) => request(`/applications/${id}`, { method: 'PATCH', body: JSON.stringify({ result }) }),

  stats: () => request('/stats'),
  report: (from, to) => request(`/reports/summary?from=${from}&to=${to}`),

  scraperStatus: () => request('/scraper/status'),
  scraperRun: () => request('/scraper/run', { method: 'POST' }),
};

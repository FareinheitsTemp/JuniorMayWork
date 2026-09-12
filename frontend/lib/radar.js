// Radar API: fresh queue, pipeline telemetry та збережена ERD-мапа.

async function request(path, options = {}) {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      message = body.error || message;
    } catch {
      // відповідь без JSON
    }
    throw new Error(message);
  }
  return res.json();
}

export const radar = {
  fresh: (limit = 8) => request(`/radar/fresh-orders?limit=${limit}`),
  runs: (limit = 30) => request(`/radar/runs?limit=${limit}`),
  layout: () => request('/schema/layout'),
  moveNode: (id, node) => request(`/schema/nodes/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(node),
  }),
  markSeen: (id) => request(`/orders/${id}/seen`, { method: 'POST' }),
  dismiss: (id) => request(`/orders/${id}/dismiss`, { method: 'POST' }),
};

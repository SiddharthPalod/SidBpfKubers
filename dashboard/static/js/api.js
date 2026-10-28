// SidKubers API Client & Server Action Bridge

const API = {
  async getStatus() {
    const res = await fetch('/api/status');
    if (!res.ok) throw new Error(`HTTP ${res.status}: status fetch failed`);
    return await res.json();
  },

  async runAction(action, params = {}) {
    const res = await fetch('/api/action', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, params })
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}: action failed`);
    return await res.json();
  },

  async getLogs() {
    const res = await fetch('/api/logs');
    if (!res.ok) throw new Error(`HTTP ${res.status}: logs fetch failed`);
    return await res.json();
  },

  async getEbpfMetrics() {
    const res = await fetch('/api/ebpf');
    if (!res.ok) throw new Error(`HTTP ${res.status}: ebpf fetch failed`);
    return await res.text();
  },

  async getEbpfEvents() {
    try {
      const res = await fetch('/api/ebpf/events');
      if (!res.ok) return [];
      return await res.json();
    } catch {
      return [];
    }
  }
};

// Authority API - uses authority_token (separate from citizen auth_token). Base URL same as api.js.
const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

function getAuthHeader() {
  const token = localStorage.getItem('authority_token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function authorityRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...getAuthHeader(), ...options.headers },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.message || data.error || `HTTP ${res.status}`);
  return data;
}

export const authorityApi = {
  login: (email, password) =>
    authorityRequest('/authority/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  me: () => authorityRequest('/authority/me'),
  complaints: (params = {}) => {
    const sp = new URLSearchParams();
    if (params.status) sp.set('status', params.status);
    sp.set('page', String(params.page ?? 1));
    sp.set('page_size', String(params.page_size ?? 20));
    return authorityRequest(`/authority/complaints?${sp}`);
  },
  updateStatus: (complaintId, newStatus, reason) =>
    authorityRequest(`/authority/complaints/${complaintId}/status`, {
      method: 'POST',
      body: JSON.stringify({ new_status: newStatus, reason }),
    }),
};

export function setAuthorityToken(token) {
  if (token) localStorage.setItem('authority_token', token);
  else localStorage.removeItem('authority_token');
}

export function getAuthorityToken() {
  return localStorage.getItem('authority_token');
}

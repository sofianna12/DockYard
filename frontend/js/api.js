const API_BASE = 'http://localhost:8081'

async function apiFetch(path, options = {}) {
  const token = localStorage.getItem('token')
  const res = await fetch(API_BASE + path, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...options,
  })

  if (res.status === 401) {
    localStorage.removeItem('token')
    window.location.href = '/pages/login.html'
    return
  }

  if (res.status === 204) return null

  const data = res.ok ? await res.json().catch(() => ({})) : await res.json()
  if (!res.ok) throw new Error(data.error || 'Request failed')
  return data
}

const api = {
  register: (email, password) =>
    apiFetch('/api/auth/register', { method: 'POST', body: JSON.stringify({ email, password }) }),

  login: (email, password) =>
    apiFetch('/api/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),

  getProjects: (page = 1, limit = 20, q = '') => apiFetch(`/api/projects?page=${page}&limit=${limit}${q ? '&q=' + encodeURIComponent(q) : ''}`),

  getProject: (id) => apiFetch(`/api/projects/${id}`),

  createProject: (data) =>
    apiFetch('/api/projects', { method: 'POST', body: JSON.stringify(data) }),

  updateProject: (id, data) =>
    apiFetch(`/api/projects/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteProject: (id) =>
    apiFetch(`/api/projects/${id}`, { method: 'DELETE' }),

  launchContainer: (id) =>
    apiFetch(`/api/projects/${id}/launch`, { method: 'POST' }),

  stopContainer: (id) =>
    apiFetch(`/api/projects/${id}/stop`, { method: 'POST' }),

  getStatus: (id) =>
    apiFetch(`/api/projects/${id}/status`),

  listFiles: (id) =>
    apiFetch(`/api/projects/${id}/files`),

  uploadFile: async (id, formData) => {
    const token = localStorage.getItem('token')
    const res = await fetch(API_BASE + `/api/projects/${id}/files`, {
      method: 'POST',
      headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: formData,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || 'Upload failed')
    return data
  },

  deleteFile: (id, filename) =>
    apiFetch(`/api/projects/${id}/files/${encodeURIComponent(filename)}`, { method: 'DELETE' }),

  getLogs: (id, tail = 100) =>
    apiFetch(`/api/projects/${id}/logs?tail=${tail}`),
}

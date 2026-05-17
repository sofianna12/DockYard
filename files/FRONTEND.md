# FRONTEND.md — Plain HTML/CSS/JS Frontend

## Overview

No framework. No build step. No npm.
Pure HTML + CSS + Vanilla JavaScript.
Communicates with the Go backend via fetch() API.

## File Structure

```
frontend/
├── Dockerfile
├── nginx.conf
├── index.html          → redirect to /login
├── css/
│   └── style.css       → global styles (shared across all pages)
├── pages/
│   ├── login.html
│   ├── register.html
│   ├── dashboard.html
│   ├── project-details.html
│   ├── add-project.html
│   └── edit-project.html
└── js/
    ├── api.js          → all fetch() calls to backend
    ├── auth.js         → login, register, logout, token helpers
    ├── dashboard.js    → load and render project cards
    ├── project-details.js → details page logic, launch, stop, poll
    ├── add-project.js  → form submit for new project
    └── edit-project.js → load existing project, form submit
```

## Dockerfile

```dockerfile
FROM nginx:alpine
COPY . /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

## nginx.conf

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri.html $uri/ /index.html;
    }
}
```

## index.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="refresh" content="0; url=/pages/login.html">
  <title>DockYard</title>
</head>
<body></body>
</html>
```

---

## css/style.css

Single stylesheet shared by all pages.

Sections to implement:

```css
/* === RESET & BASE === */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Inter', sans-serif; background: #0f1117; color: #e2e8f0; min-height: 100vh; }

/* === NAVBAR === */
.navbar { ... }
.navbar-title { ... }
.navbar-logout { ... }

/* === AUTH PAGES (login, register) === */
.auth-container { ... }   /* centered card */
.auth-card { ... }
.auth-title { ... }
.auth-link { ... }        /* "Don't have an account? Register" */

/* === FORM ELEMENTS === */
.form-group { ... }
.form-label { ... }
.form-input { ... }
.form-select { ... }
.form-textarea { ... }
.form-error { color: #fc8181; font-size: 0.85rem; }

/* === BUTTONS === */
.btn { ... }
.btn-primary { background: #3b82f6; color: white; }
.btn-secondary { background: transparent; border: 1px solid #4a5568; }
.btn-danger { background: #e53e3e; color: white; }
.btn-sm { padding: 0.25rem 0.75rem; font-size: 0.85rem; }
.btn:disabled { opacity: 0.6; cursor: not-allowed; }

/* === DASHBOARD === */
.dashboard-container { ... }
.dashboard-header { ... }   /* title + add button */
.projects-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1.5rem; }
.empty-state { ... }        /* shown when no projects */

/* === PROJECT CARD === */
.project-card { background: #1a1f2e; border-radius: 12px; padding: 1.5rem; cursor: pointer; transition: transform 0.2s; }
.project-card:hover { transform: translateY(-2px); }
.project-card-title { font-size: 1.2rem; font-weight: 600; }
.project-card-description { color: #94a3b8; font-size: 0.9rem; margin-top: 0.5rem; }
.project-card-image { font-family: monospace; font-size: 0.8rem; color: #64748b; margin-top: 1rem; }

/* === STATUS BADGE === */
.badge { display: inline-block; padding: 0.2rem 0.75rem; border-radius: 999px; font-size: 0.8rem; font-weight: 600; }
.badge-stopped  { background: #2d3748; color: #94a3b8; }
.badge-pulling  { background: #744210; color: #fbd38d; }
.badge-running  { background: #1c4532; color: #68d391; }
.badge-error    { background: #742a2a; color: #fc8181; }

/* === PROJECT DETAILS PAGE === */
.details-container { max-width: 700px; margin: 2rem auto; padding: 0 1rem; }
.details-card { background: #1a1f2e; border-radius: 12px; padding: 2rem; }
.details-title { font-size: 2rem; font-weight: 700; }
.details-row { display: flex; gap: 1rem; margin-top: 1rem; border-bottom: 1px solid #2d3748; padding-bottom: 1rem; }
.details-label { color: #64748b; width: 140px; flex-shrink: 0; }
.details-value { color: #e2e8f0; }
.details-actions { display: flex; gap: 1rem; margin-top: 2rem; }
.countdown { color: #f59e0b; font-size: 0.9rem; margin-top: 0.5rem; }
.open-app-link { color: #3b82f6; text-decoration: underline; font-size: 0.9rem; }

/* === SPINNER === */
.spinner { display: inline-block; width: 16px; height: 16px;
  border: 2px solid #ffffff40; border-top-color: #fff;
  border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* === ALERTS === */
.alert { padding: 0.75rem 1rem; border-radius: 8px; margin-bottom: 1rem; font-size: 0.9rem; }
.alert-error   { background: #742a2a; color: #fc8181; }
.alert-success { background: #1c4532; color: #68d391; }

/* === RESPONSIVE === */
@media (max-width: 640px) {
  .projects-grid { grid-template-columns: 1fr; }
  .details-row { flex-direction: column; }
}
```

---

## js/api.js

```javascript
const API_BASE = 'http://localhost:8080'

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

  const data = res.ok ? await res.json().catch(() => ({})) : await res.json()
  if (!res.ok) throw new Error(data.error || 'Request failed')
  return data
}

// Auth
const api = {
  register: (email, password) =>
    apiFetch('/api/auth/register', { method: 'POST', body: JSON.stringify({ email, password }) }),

  login: (email, password) =>
    apiFetch('/api/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),

  // Projects
  getProjects: () => apiFetch('/api/projects'),

  getProject: (id) => apiFetch(`/api/projects/${id}`),

  createProject: (data) =>
    apiFetch('/api/projects', { method: 'POST', body: JSON.stringify(data) }),

  updateProject: (id, data) =>
    apiFetch(`/api/projects/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteProject: (id) =>
    apiFetch(`/api/projects/${id}`, { method: 'DELETE' }),

  // Docker
  launchContainer: (id) =>
    apiFetch(`/api/projects/${id}/launch`, { method: 'POST' }),

  stopContainer: (id) =>
    apiFetch(`/api/projects/${id}/stop`, { method: 'POST' }),

  getStatus: (id) =>
    apiFetch(`/api/projects/${id}/status`),
}
```

---

## js/auth.js

```javascript
// Guard: redirect to login if no token
function requireAuth() {
  if (!localStorage.getItem('token')) {
    window.location.href = '/pages/login.html'
  }
}

// Logout
function logout() {
  localStorage.removeItem('token')
  window.location.href = '/pages/login.html'
}
```

---

## pages/login.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>DockYard — Login</title>
  <link rel="stylesheet" href="../css/style.css">
</head>
<body>
  <div class="auth-container">
    <div class="auth-card">
      <h1 class="auth-title">⚓ DockYard</h1>
      <p style="color:#64748b; margin-bottom:2rem;">Sign in to your account</p>

      <div id="alert" class="alert alert-error" style="display:none"></div>

      <div class="form-group">
        <label class="form-label">Email</label>
        <input id="email" type="email" class="form-input" placeholder="you@example.com">
      </div>
      <div class="form-group">
        <label class="form-label">Password</label>
        <input id="password" type="password" class="form-input" placeholder="••••••••">
      </div>

      <button id="submit-btn" class="btn btn-primary" style="width:100%; margin-top:1rem">
        Sign In
      </button>

      <p class="auth-link">Don't have an account? <a href="register.html">Register</a></p>
    </div>
  </div>

  <script src="../js/api.js"></script>
  <script>
    // Redirect if already logged in
    if (localStorage.getItem('token')) window.location.href = 'dashboard.html'

    document.getElementById('submit-btn').addEventListener('click', async () => {
      const email    = document.getElementById('email').value.trim()
      const password = document.getElementById('password').value
      const alert    = document.getElementById('alert')
      const btn      = document.getElementById('submit-btn')

      if (!email || !password) {
        alert.textContent = 'Please fill in all fields'
        alert.style.display = 'block'
        return
      }

      btn.disabled = true
      btn.innerHTML = '<span class="spinner"></span>'

      try {
        const res = await api.login(email, password)
        localStorage.setItem('token', res.token)
        window.location.href = 'dashboard.html'
      } catch (err) {
        alert.textContent = err.message
        alert.style.display = 'block'
        btn.disabled = false
        btn.textContent = 'Sign In'
      }
    })
  </script>
</body>
</html>
```

---

## pages/register.html

Same structure as login.html but:
- Title: "Create Account"
- Calls api.register() then redirects to login.html on success
- Link: "Already have an account? Sign in"

---

## pages/dashboard.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>DockYard — Dashboard</title>
  <link rel="stylesheet" href="../css/style.css">
</head>
<body>

  <nav class="navbar">
    <span class="navbar-title">⚓ DockYard</span>
    <button class="btn btn-secondary btn-sm" onclick="logout()">Logout</button>
  </nav>

  <div class="dashboard-container">
    <div class="dashboard-header">
      <h2>My Projects</h2>
      <a href="add-project.html" class="btn btn-primary">+ Add Project</a>
    </div>
    <div id="projects-grid" class="projects-grid">
      <!-- cards injected by dashboard.js -->
    </div>
  </div>

  <script src="../js/api.js"></script>
  <script src="../js/auth.js"></script>
  <script src="../js/dashboard.js"></script>
</body>
</html>
```

---

## js/dashboard.js

```javascript
requireAuth()

async function loadProjects() {
  const grid = document.getElementById('projects-grid')
  try {
    const projects = await api.getProjects()

    if (projects.length === 0) {
      grid.innerHTML = `
        <div class="empty-state">
          <p>No projects yet.</p>
          <a href="add-project.html" class="btn btn-primary" style="margin-top:1rem">+ Add your first project</a>
        </div>`
      return
    }

    grid.innerHTML = projects.map(p => `
      <div class="project-card" onclick="window.location.href='project-details.html?id=${p.id}'">
        <div style="display:flex; justify-content:space-between; align-items:center">
          <span class="project-card-title">${p.title}</span>
          <span class="badge badge-${p.status}">${p.status}</span>
        </div>
        <p class="project-card-description">${p.description || 'No description'}</p>
        <p class="project-card-image">🐳 ${p.docker_image}</p>
      </div>
    `).join('')
  } catch (err) {
    grid.innerHTML = `<div class="alert alert-error">${err.message}</div>`
  }
}

loadProjects()
```

---

## pages/project-details.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>DockYard — Project Details</title>
  <link rel="stylesheet" href="../css/style.css">
</head>
<body>

  <nav class="navbar">
    <span class="navbar-title">⚓ DockYard</span>
    <button class="btn btn-secondary btn-sm" onclick="logout()">Logout</button>
  </nav>

  <div class="details-container">
    <div class="details-card">

      <div id="alert" class="alert alert-error" style="display:none"></div>

      <div style="display:flex; justify-content:space-between; align-items:flex-start">
        <h1 id="title" class="details-title">Loading...</h1>
        <span id="status-badge" class="badge badge-stopped">stopped</span>
      </div>

      <div id="countdown" class="countdown" style="display:none"></div>
      <div id="open-app" style="display:none; margin-top:0.5rem">
        <a id="open-app-link" href="#" target="_blank" class="open-app-link">🔗 Open App</a>
      </div>

      <div class="details-row">
        <span class="details-label">Description</span>
        <span id="description" class="details-value">—</span>
      </div>
      <div class="details-row">
        <span class="details-label">Repository</span>
        <span id="repository" class="details-value">—</span>
      </div>
      <div class="details-row">
        <span class="details-label">Docker Image</span>
        <span id="docker-image" class="details-value" style="font-family:monospace">—</span>
      </div>
      <div class="details-row">
        <span class="details-label">Auto-stop</span>
        <span id="auto-stop" class="details-value">—</span>
      </div>
      <div class="details-row">
        <span class="details-label">Created</span>
        <span id="created-at" class="details-value">—</span>
      </div>

      <div class="details-actions">
        <button class="btn btn-secondary" onclick="window.location.href='dashboard.html'">← Back</button>
        <button id="launch-btn" class="btn btn-primary" onclick="launchContainer()">▶ Run Docker</button>
        <button id="stop-btn"   class="btn btn-danger"  onclick="stopContainer()" style="display:none">■ Stop</button>
        <a href="#" onclick="editProject()" class="btn btn-secondary">✏ Edit</a>
      </div>

    </div>
  </div>

  <script src="../js/api.js"></script>
  <script src="../js/auth.js"></script>
  <script src="../js/project-details.js"></script>
</body>
</html>
```

---

## js/project-details.js

```javascript
requireAuth()

const params     = new URLSearchParams(window.location.search)
const projectId  = params.get('id')
if (!projectId) window.location.href = 'dashboard.html'

let project      = null
let pollInterval = null
let countdownInterval = null

function editProject() {
  window.location.href = `edit-project.html?id=${projectId}`
}

function renderProject(p) {
  project = p
  document.getElementById('title').textContent        = p.title
  document.getElementById('description').textContent  = p.description || '—'
  document.getElementById('docker-image').textContent = p.docker_image
  document.getElementById('auto-stop').textContent    = p.auto_stop_min + ' minutes'
  document.getElementById('created-at').textContent   = new Date(p.created_at).toLocaleDateString()

  // Repository link
  const repoEl = document.getElementById('repository')
  if (p.repository) {
    repoEl.innerHTML = `<a href="${p.repository}" target="_blank" class="open-app-link">${p.repository}</a>`
  }

  updateStatus(p.status, p.port, p.last_accessed_at)
}

function updateStatus(status, port, lastAccessed) {
  const badge     = document.getElementById('status-badge')
  const launchBtn = document.getElementById('launch-btn')
  const stopBtn   = document.getElementById('stop-btn')
  const openApp   = document.getElementById('open-app')
  const openLink  = document.getElementById('open-app-link')

  badge.className   = `badge badge-${status}`
  badge.textContent = status

  if (status === 'running') {
    launchBtn.style.display = 'none'
    stopBtn.style.display   = 'inline-block'
    openApp.style.display   = 'block'
    openLink.href           = `http://localhost:${port}`
    startCountdown(lastAccessed, project?.auto_stop_min)
    startPolling()
  } else if (status === 'pulling') {
    launchBtn.disabled      = true
    launchBtn.innerHTML     = '<span class="spinner"></span> Pulling...'
    stopBtn.style.display   = 'none'
    openApp.style.display   = 'none'
    startPolling()
  } else {
    launchBtn.disabled      = false
    launchBtn.textContent   = '▶ Run Docker'
    launchBtn.style.display = 'inline-block'
    stopBtn.style.display   = 'none'
    openApp.style.display   = 'none'
    stopPolling()
    stopCountdown()
  }
}

function startPolling() {
  if (pollInterval) return
  pollInterval = setInterval(async () => {
    try {
      const res = await api.getStatus(projectId)
      updateStatus(res.status, res.port, project?.last_accessed_at)
    } catch (_) {}
  }, 5000)
}

function stopPolling() {
  clearInterval(pollInterval)
  pollInterval = null
}

function startCountdown(lastAccessed, autoStopMin) {
  stopCountdown()
  const el = document.getElementById('countdown')
  el.style.display = 'block'

  function tick() {
    const expires = new Date(new Date(lastAccessed).getTime() + autoStopMin * 60000)
    const diff    = expires - Date.now()
    if (diff <= 0) {
      el.textContent = 'Auto-stopping...'
      return
    }
    const m = Math.floor(diff / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    el.textContent = `⏱ Auto-stop in ${m}m ${s}s`
  }

  tick()
  countdownInterval = setInterval(tick, 1000)
}

function stopCountdown() {
  clearInterval(countdownInterval)
  countdownInterval = null
  document.getElementById('countdown').style.display = 'none'
}

async function launchContainer() {
  const btn   = document.getElementById('launch-btn')
  const alert = document.getElementById('alert')
  alert.style.display = 'none'

  btn.disabled    = true
  btn.innerHTML   = '<span class="spinner"></span> Launching...'

  try {
    const res = await api.launchContainer(projectId)
    window.open(`http://localhost:${res.port}`, '_blank')
    updateStatus('running', res.port, new Date().toISOString())
  } catch (err) {
    alert.textContent   = err.message
    alert.style.display = 'block'
    btn.disabled        = false
    btn.textContent     = '▶ Run Docker'
  }
}

async function stopContainer() {
  const btn = document.getElementById('stop-btn')
  btn.disabled    = true
  btn.textContent = 'Stopping...'
  try {
    await api.stopContainer(projectId)
    updateStatus('stopped', null, null)
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
    btn.disabled    = false
    btn.textContent = '■ Stop'
  }
}

// Initial load
async function init() {
  try {
    const p = await api.getProject(projectId)
    renderProject(p)
  } catch (err) {
    document.getElementById('alert').textContent   = err.message
    document.getElementById('alert').style.display = 'block'
  }
}

init()
```

---

## pages/add-project.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>DockYard — Add Project</title>
  <link rel="stylesheet" href="../css/style.css">
</head>
<body>

  <nav class="navbar">
    <span class="navbar-title">⚓ DockYard</span>
    <button class="btn btn-secondary btn-sm" onclick="logout()">Logout</button>
  </nav>

  <div class="details-container">
    <div class="details-card">
      <h2 style="margin-bottom:1.5rem">Add New Project</h2>

      <div id="alert" class="alert alert-error" style="display:none"></div>

      <div class="form-group">
        <label class="form-label">Title *</label>
        <input id="title" type="text" class="form-input" placeholder="Tetris">
      </div>
      <div class="form-group">
        <label class="form-label">Description</label>
        <textarea id="description" class="form-textarea" rows="3" placeholder="Arcade game..."></textarea>
      </div>
      <div class="form-group">
        <label class="form-label">Repository</label>
        <input id="repository" type="url" class="form-input" placeholder="https://github.com/user/tetris">
      </div>
      <div class="form-group">
        <label class="form-label">Docker Image *</label>
        <input id="docker-image" type="text" class="form-input" placeholder="bsord/tetris:latest">
      </div>
      <div class="form-group">
        <label class="form-label">Auto-stop after</label>
        <select id="auto-stop" class="form-select">
          <option value="30">30 minutes</option>
          <option value="60" selected>1 hour</option>
          <option value="240">4 hours</option>
          <option value="0">Never</option>
        </select>
      </div>

      <div class="details-actions">
        <button class="btn btn-secondary" onclick="window.location.href='dashboard.html'">← Cancel</button>
        <button id="submit-btn" class="btn btn-primary" onclick="submitForm()">Save Project</button>
      </div>
    </div>
  </div>

  <script src="../js/api.js"></script>
  <script src="../js/auth.js"></script>
  <script src="../js/add-project.js"></script>
</body>
</html>
```

---

## js/add-project.js

```javascript
requireAuth()

async function submitForm() {
  const btn   = document.getElementById('submit-btn')
  const alert = document.getElementById('alert')

  const data = {
    title:        document.getElementById('title').value.trim(),
    description:  document.getElementById('description').value.trim(),
    repository:   document.getElementById('repository').value.trim(),
    docker_image: document.getElementById('docker-image').value.trim(),
    auto_stop_min: parseInt(document.getElementById('auto-stop').value),
  }

  if (!data.title || !data.docker_image) {
    alert.textContent   = 'Title and Docker Image are required'
    alert.style.display = 'block'
    return
  }

  btn.disabled    = true
  btn.innerHTML   = '<span class="spinner"></span> Saving...'
  alert.style.display = 'none'

  try {
    await api.createProject(data)
    window.location.href = 'dashboard.html'
  } catch (err) {
    alert.textContent   = err.message
    alert.style.display = 'block'
    btn.disabled        = false
    btn.textContent     = 'Save Project'
  }
}
```

---

## pages/edit-project.html + js/edit-project.js

Same as add-project but:
- On load: fetch project by ?id= and prefill all fields
- On submit: call api.updateProject(id, data) instead of createProject
- On success: redirect to project-details.html?id=...
- Title: "Edit Project"
- Cancel button: goes back to project-details.html?id=...

---

## Navigation Flow

```
login.html
  └──→ dashboard.html
          └──→ add-project.html → dashboard.html
          └──→ project-details.html?id=XXX
                  └──→ edit-project.html?id=XXX → project-details.html?id=XXX
                  └──→ dashboard.html (Back button)
                  └──→ http://localhost:<port> (Open App — new tab)
```

## No Build Step Required

To run locally without Docker:
- Open login.html directly in a browser
- Or serve with any static server: `npx serve .` or `python3 -m http.server 3000`

With Docker: served by nginx on port 80 (mapped to 5173 in docker-compose).

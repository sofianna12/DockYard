# IMPLEMENTATION_ORDER.md — Step-by-Step Build Guide (Plain HTML Version)

No React. No npm. No build step.
Frontend = plain HTML + CSS + Vanilla JavaScript.
Backend = Go + Gin.
Database = PostgreSQL.

Read all .md files before starting. Implement phases in order.

---

## Phase 1 — Project Scaffold

### Step 1.1 — Root files
```
dockyard/
├── docker-compose.yml     (see DOCKER.md)
├── .env.example           (see DOCKER.md)
└── README.md
```

### Step 1.2 — Backend scaffold
```
backend/
├── Dockerfile
├── main.go
├── go.mod
└── .env
```

### Step 1.3 — Frontend scaffold
```
frontend/
├── Dockerfile
├── nginx.conf
├── index.html
├── css/
│   └── style.css
├── pages/
│   ├── login.html
│   ├── register.html
│   ├── dashboard.html
│   ├── project-details.html
│   ├── add-project.html
│   └── edit-project.html
└── js/
    ├── api.js
    ├── auth.js
    ├── dashboard.js
    ├── project-details.js
    ├── add-project.js
    └── edit-project.js
```

### Step 1.4 — Database scaffold
```
database/
├── Dockerfile
└── init/
    ├── 001_create_users.sql
    ├── 002_create_projects.sql
    └── 003_add_lifecycle.sql
```

---

## Phase 2 — Database

### Step 2.1
Create SQL migration files — see DATABASE.md for exact SQL.

### Step 2.2
Implement backend/models/user.go and backend/models/project.go — see BACKEND.md.

### Step 2.3
Implement backend/db/postgres.go — connect and AutoMigrate.

### Step 2.4
Implement backend/config/config.go — load env vars.

### Verification
```bash
docker-compose up database
# should show: database system is ready to accept connections
```

---

## Phase 3 — Backend Auth

### Step 3.1
Implement backend/handlers/auth.go:
- POST /api/auth/register
- POST /api/auth/login

### Step 3.2
Implement backend/middleware/auth.go — JWT validation.

### Step 3.3
Wire routes and CORS in main.go.

### Verification
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456"}'
# → 201

curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456"}'
# → { "token": "..." }
```

---

## Phase 4 — Backend Projects CRUD

### Step 4.1
Implement backend/handlers/projects.go — all 5 CRUD handlers.
Every query must filter by user_id from JWT.

### Verification
```bash
TOKEN="<from login>"
curl http://localhost:8080/api/projects -H "Authorization: Bearer $TOKEN"
# → []
```

---

## Phase 5 — Docker Integration (Backend)

### Step 5.1
Implement backend/services/docker/client.go — singleton Docker client.

### Step 5.2
Implement backend/services/docker/image.go — PullImage().

### Step 5.3
Implement backend/services/docker/container.go:
- RunContainer() → containerID, port
- StopAndRemoveContainer()

### Step 5.4
Implement backend/handlers/docker.go:
- POST /api/projects/:id/launch
- POST /api/projects/:id/stop
- GET  /api/projects/:id/status

### Step 5.5
Implement backend/services/cleanup/worker.go.
Start it in main.go with: go cleanup.StartWorker(db)

### Verification
```bash
# Create a project first, then:
curl -X POST http://localhost:8080/api/projects/$ID/launch \
  -H "Authorization: Bearer $TOKEN"
# → { "port": 49152, "url": "http://localhost:49152" }
```

---

## Phase 6 — Frontend: Shared Files

### Step 6.1 — css/style.css
Implement the full stylesheet as described in FRONTEND.md.
Sections: reset, navbar, auth, form elements, buttons,
dashboard, project card, status badge, details page,
spinner, alerts, responsive.

### Step 6.2 — js/api.js
Implement apiFetch() with JWT header injection and 401 redirect.
Implement the api object with all methods:
register, login, getProjects, getProject, createProject,
updateProject, deleteProject, launchContainer, stopContainer, getStatus.

### Step 6.3 — js/auth.js
Implement requireAuth() and logout().

### Step 6.4 — index.html
Meta refresh redirect to /pages/login.html.

---

## Phase 7 — Frontend: Auth Pages

### Step 7.1 — pages/login.html + inline script
- Email + password fields
- Calls api.login() on button click
- Shows spinner while loading
- Stores token in localStorage on success
- Redirects to dashboard.html
- Shows error alert on failure
- Link to register.html

### Step 7.2 — pages/register.html + inline script
- Email + password fields
- Calls api.register() on button click
- On success: redirect to login.html with success message
- On failure: show error alert
- Link to login.html

### Verification
Open http://localhost:5173/pages/login.html in browser.
Register → Login → should reach dashboard.html.

---

## Phase 8 — Frontend: Dashboard

### Step 8.1 — pages/dashboard.html
Navbar + projects grid container + Add Project button.

### Step 8.2 — js/dashboard.js
- Call requireAuth()
- Fetch projects via api.getProjects()
- Render project cards dynamically
- Each card: title, description, status badge, docker image
- Card click → navigate to project-details.html?id=XXX
- Empty state when no projects

### Verification
Add a project via API and confirm card appears on dashboard.

---

## Phase 9 — Frontend: Add Project

### Step 9.1 — pages/add-project.html
Form with: title, description, repository, docker image, auto-stop select.
Cancel → dashboard.html.

### Step 9.2 — js/add-project.js
- requireAuth()
- Validate title + docker_image not empty
- Call api.createProject()
- Show spinner while saving
- On success: redirect to dashboard.html
- On failure: show error alert

### Verification
Fill form → submit → card appears on dashboard.

---

## Phase 10 — Frontend: Project Details

### Step 10.1 — pages/project-details.html
Layout as described in FRONTEND.md:
title, status badge, countdown, open-app link,
info rows (description, repository, docker image, auto-stop, created),
action buttons (Back, Run Docker, Stop, Edit).

### Step 10.2 — js/project-details.js
Implement all functions as described in FRONTEND.md:
- init() — load project and call renderProject()
- renderProject(p) — fill all DOM elements
- updateStatus(status, port, lastAccessed) — show/hide correct buttons
- launchContainer() — call API, show spinner, open new tab on success
- stopContainer() — call API, reset UI
- startPolling() / stopPolling() — poll status every 5 seconds
- startCountdown() / stopCountdown() — live countdown timer

### Verification
Click a project card → see details page.
Click Run Docker → image pulls → status becomes running.
New tab opens with the running app.
Countdown shows.
Click Stop → status returns to stopped.

---

## Phase 11 — Frontend: Edit Project

### Step 11.1 — pages/edit-project.html
Same form as add-project.html but:
- Heading: "Edit Project"
- Cancel → project-details.html?id=XXX

### Step 11.2 — js/edit-project.js
- requireAuth()
- Read ?id= from URL
- Load project → prefill all form fields
- On submit: call api.updateProject(id, data)
- On success: redirect to project-details.html?id=XXX
- On failure: show error alert

---

## Phase 12 — Final Integration Test

### Step 12.1
```bash
docker-compose up --build
```
Open http://localhost:5173

### Step 12.2 — Manual checklist
- [ ] Register a new user
- [ ] Login → redirected to dashboard
- [ ] Dashboard shows empty state
- [ ] Click "+ Add Project"
- [ ] Fill form (docker image: bsord/tetris:latest) → Save
- [ ] Card appears on dashboard with "stopped" badge
- [ ] Click card → project details page
- [ ] All fields displayed correctly
- [ ] Repository shows as clickable link
- [ ] Click "← Back" → dashboard
- [ ] Click "▶ Run Docker" → button shows spinner + "Launching..."
- [ ] Status badge changes: stopped → pulling → running
- [ ] "Open App" link appears → click → opens game in new tab
- [ ] Countdown timer ticks down
- [ ] Click "■ Stop" → status returns to stopped
- [ ] Click "✏ Edit" → edit form prefilled
- [ ] Change title → save → details page shows new title
- [ ] Logout → redirected to login
- [ ] Visiting dashboard.html without login → redirected to login

### Step 12.3 — Cross-platform
Run docker-compose up --build on Linux, macOS, and Windows (WSL2).
All functionality should work identically.

---

## Notes for AI

- Do NOT use any JavaScript framework (no React, Vue, Angular, jQuery)
- Do NOT use npm or any build tool for the frontend
- Use fetch() for all HTTP calls — already wrapped in js/api.js
- JWT token is stored in localStorage with key 'token'
- Project ID is passed between pages via URL query string: ?id=XXX
- All pages load js/api.js and js/auth.js before their own script
- Inline <script> tags are acceptable for page-specific logic in html files,
  but keep complex logic in separate .js files
- Status polling and countdown timer must be cleared when leaving the page
  (use window.onbeforeunload or equivalent)
- The Docker socket mount in docker-compose is required — do not remove it
- Never hardcode the JWT secret
- All backend errors return { "error": "message" } — display this to the user

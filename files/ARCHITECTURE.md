# ARCHITECTURE.md — DockYard System Architecture

## Overview

DockYard is split into 3 Docker services that communicate over an internal Docker network:

```
┌─────────────────────────────────────────────────────────┐
│                     Docker Network                      │
│                                                         │
│  ┌──────────────┐    ┌──────────────┐  ┌────────────┐  │
│  │   frontend   │───▶│   backend    │─▶│  database  │  │
│  │ React + Vite │    │   Go + Gin   │  │ PostgreSQL │  │
│  │  port 5173   │    │  port 8080   │  │ port 5432  │  │
│  └──────────────┘    └──────┬───────┘  └────────────┘  │
│                             │                           │
└─────────────────────────────┼───────────────────────────┘
                              │ /var/run/docker.sock
                              ▼
                       Docker Daemon (Host)
                       (manages user containers)
```

## User Flow

```
1. User visits http://localhost:5173
2. Registers or logs in → receives JWT token
3. Dashboard shows their project cards
4. Clicks "+ Add Project" → fills form:
      Title:        Tetris
      Description:  Arcade game
      Repository:   https://github.com/user/tetris
      Docker Image: bsord/tetris:latest
      Auto-stop:    60 minutes
5. Saved to PostgreSQL
6. Clicks a project card → navigates to /projects/:id (Details Page)
7. Details page shows all project info + two buttons:
      [← Back]       → returns to Dashboard
      [▶ Run Docker] → launches the container
8. Backend receives POST /api/projects/:id/launch
9. Backend pulls image if not present locally
10. Backend runs container with docker SDK → gets dynamic port
11. Frontend polls GET /api/projects/:id/status every 5 seconds
12. When status = "running", frontend redirects to http://localhost:<port>
13. Background cleanup worker checks every 5 minutes:
        if last_accessed_at < NOW() - auto_stop_minutes → stop + remove container
```

## Component Responsibilities

### Frontend (React)
- Renders all UI pages
- Stores JWT in memory (AuthContext) or localStorage
- Calls backend REST API via Axios
- Polls container status every 5 seconds using TanStack Query
- Shows countdown timer to auto-stop
- Handles routing with React Router v6

### Backend (Go + Gin)
- Exposes REST API on port 8080
- Handles JWT auth middleware
- Performs all DB operations via GORM
- Controls Docker daemon via Docker SDK for Go
- Runs background cleanup goroutine on startup
- Returns container port after launch so frontend can redirect

### Database (PostgreSQL)
- Stores users and projects
- Projects table includes lifecycle fields: started_at, last_accessed_at, auto_stop_min
- Migrations run automatically on first startup via init scripts

## Security Notes

- Passwords are hashed with bcrypt (cost 12)
- JWT secret is set via environment variable
- JWT tokens expire after 24 hours
- All /api/projects/* routes require valid JWT
- Docker socket is mounted read/write — backend runs as a trusted internal service only
- Users can only see/edit their own projects (user_id check on every query)

## Auto-Stop Logic

```
Every 5 minutes, cleanup worker runs:
  SELECT * FROM projects
  WHERE status = 'running'
  AND last_accessed_at < NOW() - INTERVAL '<auto_stop_min> minutes'

For each result:
  docker stop <container_id>
  docker rm <container_id>
  UPDATE projects SET status='stopped', container_id=NULL, port=NULL
```

last_accessed_at is updated every time:
  - User opens the Details page (GET /api/projects/:id)
  - Frontend polls status (GET /api/projects/:id/status)

## Environment Variables

### Backend (.env)
```
PORT=8080
DB_URL=postgres://dockyard:dockyard@database:5432/dockyard
JWT_SECRET=change_this_to_a_long_random_string
IDLE_TIMEOUT_MINUTES=60
CLEANUP_INTERVAL_MINUTES=5
```

### Frontend (.env)
```
VITE_API_URL=http://localhost:8080
```

### Docker Compose (.env.example)
```
POSTGRES_USER=dockyard
POSTGRES_PASSWORD=dockyard
POSTGRES_DB=dockyard
JWT_SECRET=change_this_to_a_long_random_string
```

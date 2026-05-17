# DockYard 🚀⚓

DockYard is a self-hosted web platform that allows each registered user to manage and run their personal projects via Docker, directly from the browser.

## What it does

A user registers/logs in, adds their projects by providing a title, description, repository link, and Docker image. With a single click on a project card, they navigate to a details page where they can launch the Docker container. The container auto-stops after a configurable idle timeout.

## Key Features

- User registration and login with JWT authentication
- Full CRUD for projects (title, description, repository, docker image, auto-stop timeout)
- Project details page with Launch / Back buttons
- One-click Docker container launch with automatic image pull
- Real-time container status (Stopped / Pulling / Running)
- Auto-stop after configurable idle timeout
- Background cleanup worker (goroutine)
- Cross-platform: Linux, macOS, Windows (via Docker Desktop)
- Fully self-hosted, zero vendor lock-in

## Architecture

```
Browser
  |
  v
Frontend (React + TypeScript + Tailwind — port 5173)
  |  REST API (Axios)
  v
Backend (Go + Gin — port 8080)
  |                    |
  v                    v
PostgreSQL          Docker Daemon
(port 5432)         (/var/run/docker.sock)
```

## Services

| Service  | Technology         | Port |
|----------|--------------------|------|
| frontend | React + Vite       | 5173 |
| backend  | Go + Gin           | 8080 |
| database | PostgreSQL 16      | 5432 |

## Quick Start

```bash
git clone https://github.com/youruser/dockyard
cd dockyard
cp .env.example .env
docker-compose up --build

Κάνεις docker-compose down → τα δεδομένα μένουν εκεί, ασφαλή
Κάνεις docker-compose up ξανά → η PostgreSQL τα βρίσκει και συνεχίζει κανονικά
Μόνο docker-compose down -v ή docker volume rm dockyard_pgdata τα διαγράφει

# Μπες στη βάση
docker exec -it dockyard_database_1 psql -U dockyard -d dockyard

# Εντολές μέσα:
\dt              → δες τους πίνακες
SELECT * FROM users;
SELECT * FROM projects;
\q               → έξοδος

```

Then open: http://localhost:5173

## Repository Structure

```
dockyard/
├── README.md
├── docker-compose.yml
├── .env.example
├── backend/
├── frontend/
└── database/
```

## Documentation Index

| File | Description |
|------|-------------|
| README.md | This file — project overview |
| ARCHITECTURE.md | Full system architecture and data flow |
| BACKEND.md | Go backend implementation guide |
| FRONTEND.md | React frontend implementation guide |
| DATABASE.md | PostgreSQL schema and migrations |
| API.md | All REST API endpoints with request/response examples |
| DOCKER.md | Docker setup, compose config, and deployment |
| IMPLEMENTATION_ORDER.md | Step-by-step build order for AI or developer |



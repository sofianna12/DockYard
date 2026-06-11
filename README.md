# DockYard 

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
Frontend (HTML/CSS/JS + Nginx — port 5173)
  |  REST API (fetch)
  v
Backend (Go + Gin — port 8081)
  |                    |
  v                    v
PostgreSQL          Docker Daemon
(port 5432)         (/var/run/docker.sock)
```

## Services

| Service  | Technology          | Port |
|----------|-------------------- |------|
| frontend | HTML/CSS/JS + Nginx | 5173 |
| backend  | Go + Gin            | 8081 |
| database | PostgreSQL 16       | 5432 |

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/sofianna12/DockYard
cd DockYard
cp .env.example .env
```

Edit `.env` and fill in the two secrets:

```bash
# JWT_SECRET — any random string:
openssl rand -base64 32

# ENCRYPTION_KEY — must be exactly 64 hex characters:
openssl rand -hex 32
```

Copy the output of each command and paste it into `.env`.

### 2. Start

Use `start.sh` / `start.ps1` instead of `docker-compose` directly.
They automatically detect the Docker socket permissions so the backend can launch containers on any OS.

**Linux / macOS:**
```bash
./start.sh up --build
```

**Windows (PowerShell):**
```powershell
./start.ps1 up --build
```

Then open: http://localhost:5173

> After the first run, the correct settings are saved in `.env`, so plain `docker-compose up` works too.

> **Rootless Docker?** Set `DOCKER_SOCK` and `DOCKER_GID` in `.env` (see `.env.example`).
> A standard Docker install needs nothing — it works out of the box.

### 3. Stop / restart

```bash 
# Linux:
./start.sh down

# Windows:
./start.ps1 down
```

Data is preserved in the PostgreSQL volume and reloaded on the next start.

### Danger zone

```bash
# Permanently delete all data (cannot be undone):
docker-compose down -v
```

### Database access

```bash
docker exec -it dockyard_database_1 psql -U dockyard -d dockyard
```

```sql
\dt                  -- list all tables
SELECT * FROM users;
SELECT * FROM projects;
\q                   -- exit
```

## Building project images

The Docker daemon only runs images it has **locally**, so build your project image on
the same daemon DockYard uses — otherwise the launch fails with `pull access denied`.
On a standard Docker install that is automatic:

```bash
docker build -t my-project:latest .
```

Then use `my-project:latest` as the project's Docker image in the UI.

> **Rootless Docker only:** if you run the daemon in rootless mode, point the build at
> its socket first, since it has a separate image store:
> ```bash
> export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
> ```

## Repository Structure

```
dockyard/
├── docker-compose.yml         # defines all services
├── .env.example               # template for environment variables
├── backend/                   # Go REST API
├── frontend/                  # HTML/CSS/JS + Nginx
└── database/                  # PostgreSQL init scripts
```





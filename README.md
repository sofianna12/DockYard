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

```bash
git clone https://github.com/sofianna12/DockYard
cd dockyard
cp .env.example .env
#You can generate a value for JWT_SECRET and ENCRYPTION_KEY by running either of these commands and
#pasting the output into your .env file:
openssl rand -base64 32
#or
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
docker-compose up --build
#Then open: http://localhost:5173


#data remains intact, safely stored in the volume
docker-compose down     
#PostgreSQL finds the existing data and continues normally    
docker-compose up          
#permanently deletes all data (use with caution)
docker-compose down -v     
#same as above, removes the volume directly
docker volume rm dockyard_pgdata 

# Connect to the database
docker exec -it dockyard_database_1 psql -U dockyard -d dockyard

# Commands inside psql:
\dt              # list all tables
SELECT * FROM users;
SELECT * FROM projects;
\q               # exit
```

## Repository Structure

```
dockyard/
├── docker-compose.yml         # defines all services
├── .env.example               # template for environment variables
├── backend/                   # Go REST API
├── frontend/                  # HTML/CSS/JS + Nginx
└── database/                  # PostgreSQL init scripts
```





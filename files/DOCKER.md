# DOCKER.md — Docker Setup & Deployment

## docker-compose.yml

```yaml
version: '3.9'

services:

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "5173:5173"
    depends_on:
      - backend
    environment:
      - VITE_API_URL=http://localhost:8080
    networks:
      - dockyard-network

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    depends_on:
      database:
        condition: service_healthy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - PORT=8080
      - DB_URL=postgres://dockyard:dockyard@database:5432/dockyard
      - JWT_SECRET=${JWT_SECRET}
      - IDLE_TIMEOUT_MINUTES=60
      - CLEANUP_INTERVAL_MINUTES=5
    networks:
      - dockyard-network

  database:
    image: postgres:16
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./database/init:/docker-entrypoint-initdb.d
    environment:
      - POSTGRES_USER=dockyard
      - POSTGRES_PASSWORD=dockyard
      - POSTGRES_DB=dockyard
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dockyard"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - dockyard-network

volumes:
  pgdata:

networks:
  dockyard-network:
    driver: bridge
```

## .env.example

```
JWT_SECRET=change_this_to_a_long_random_secret_string
POSTGRES_USER=dockyard
POSTGRES_PASSWORD=dockyard
POSTGRES_DB=dockyard
```

## How to Run

### First time
```bash
git clone https://github.com/youruser/dockyard
cd dockyard
cp .env.example .env
# Edit .env and set a real JWT_SECRET
docker-compose up --build
```

### Subsequent runs
```bash
docker-compose up
```

### Stop
```bash
docker-compose down
```

### Stop and delete database
```bash
docker-compose down -v
```

## Important: Docker Socket

The backend needs access to the host Docker daemon to manage user containers:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

### Linux / macOS
The path /var/run/docker.sock works natively.

### Windows (WSL2 + Docker Desktop)
Docker Desktop for Windows automatically exposes the socket.
The same path /var/run/docker.sock works inside WSL2.

## Port Map

| Service  | Container Port | Host Port | Access |
|----------|---------------|-----------|--------|
| frontend | 5173          | 5173      | http://localhost:5173 |
| backend  | 8080          | 8080      | http://localhost:8080 |
| database | 5432          | 5432      | localhost:5432 |
| user containers | dynamic | dynamic | http://localhost:<port> |

## User Container Ports

When a user launches a project, Docker assigns a random available port on the host (e.g. 32768–60999).
The backend returns this port to the frontend, which opens it in a new browser tab.

Example: user launches Tetris → Docker runs on port 49152 → frontend opens http://localhost:49152

## Network Architecture

```
Host Machine
├── Docker Engine (daemon)
├── dockyard-network (internal bridge)
│   ├── frontend  (5173)
│   ├── backend   (8080) ←──── also connected to Docker socket
│   └── database  (5432)
└── User containers (separate, published to host ports)
```

User containers are NOT on the dockyard-network.
They are launched directly on the host Docker and accessible via localhost:<port>.

## Troubleshooting

### Database not ready
The backend waits for the database healthcheck before starting.
If it still fails, increase the retry count in the healthcheck.

### Docker socket permission denied (Linux)
```bash
sudo chmod 666 /var/run/docker.sock
# or add your user to the docker group:
sudo usermod -aG docker $USER
```

### Port already in use
Change the host port in docker-compose.yml:
```yaml
ports:
  - "8081:8080"  # use 8081 instead
```

# API.md — REST API Reference

## Base URL
```
http://localhost:8080
```

## Authentication
All routes except /api/auth/* require a JWT token in the Authorization header:
```
Authorization: Bearer <token>
```

---

## Auth Endpoints

### POST /api/auth/register

Register a new user.

Request:
```json
{
  "email": "user@example.com",
  "password": "secretpassword"
}
```

Response 201:
```json
{
  "message": "registered successfully"
}
```

Response 409 (email taken):
```json
{
  "error": "email already in use"
}
```

---

### POST /api/auth/login

Login and receive JWT token.

Request:
```json
{
  "email": "user@example.com",
  "password": "secretpassword"
}
```

Response 200:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Response 401:
```json
{
  "error": "invalid credentials"
}
```

---

## Project Endpoints

All require: `Authorization: Bearer <token>`

---

### GET /api/projects

Get all projects for the authenticated user.

Response 200:
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "...",
    "title": "Tetris",
    "description": "Arcade game",
    "repository": "https://github.com/user/tetris",
    "docker_image": "bsord/tetris:latest",
    "auto_stop_min": 60,
    "status": "stopped",
    "container_id": null,
    "port": null,
    "started_at": null,
    "last_accessed_at": null,
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
]
```

---

### POST /api/projects

Create a new project.

Request:
```json
{
  "title": "Tetris",
  "description": "Arcade game",
  "repository": "https://github.com/user/tetris",
  "docker_image": "bsord/tetris:latest",
  "auto_stop_min": 60
}
```

Response 201:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Tetris",
  "description": "Arcade game",
  "repository": "https://github.com/user/tetris",
  "docker_image": "bsord/tetris:latest",
  "auto_stop_min": 60,
  "status": "stopped",
  ...
}
```

---

### GET /api/projects/:id

Get a single project by ID. Also updates last_accessed_at.

Response 200: (same shape as single project above)

Response 404:
```json
{ "error": "project not found" }
```

---

### PUT /api/projects/:id

Update a project. Only updates provided fields.

Request:
```json
{
  "title": "Tetris Updated",
  "auto_stop_min": 120
}
```

Response 200: (updated project)

---

### DELETE /api/projects/:id

Delete a project. If container is running, stops and removes it first.

Response 204: (no body)

---

## Docker Endpoints

### POST /api/projects/:id/launch

Launch the Docker container for a project.

Flow:
1. If already running, returns existing port immediately
2. Sets status to "pulling"
3. Pulls image if not present
4. Creates and starts container with -P (random port)
5. Updates status to "running", saves container_id and port

Response 200:
```json
{
  "port": 32768,
  "url": "http://localhost:32768"
}
```

Response 409 (already running):
```json
{
  "port": 32768,
  "url": "http://localhost:32768",
  "message": "already running"
}
```

Response 500 (docker error):
```json
{
  "error": "failed to pull image: ..."
}
```

---

### POST /api/projects/:id/stop

Stop and remove the running container.

Response 200:
```json
{
  "message": "container stopped"
}
```

---

### GET /api/projects/:id/status

Get current container status. Also updates last_accessed_at to prevent auto-stop while user is active.

Response 200:
```json
{
  "status": "running",
  "port": 32768
}
```

Possible status values:
- "stopped"  — container is not running
- "pulling"  — image is being pulled
- "running"  — container is active
- "error"    — container failed to start

---

## Error Response Format

All errors follow this shape:
```json
{
  "error": "human readable message"
}
```

## HTTP Status Codes Used

| Code | Meaning |
|------|---------|
| 200 | OK |
| 201 | Created |
| 204 | No Content (delete) |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (missing or invalid JWT) |
| 403 | Forbidden (not your resource) |
| 404 | Not Found |
| 409 | Conflict (e.g. email taken, already running) |
| 500 | Internal Server Error |

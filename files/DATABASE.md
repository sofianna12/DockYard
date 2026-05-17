# DATABASE.md — PostgreSQL Schema & Migrations

## Overview

Database: PostgreSQL 16
ORM: GORM (used in backend)
Migration strategy: SQL files in database/init/ — executed automatically by PostgreSQL on first container start

## File Structure

```
database/
├── Dockerfile
└── init/
    ├── 001_create_users.sql
    ├── 002_create_projects.sql
    └── 003_add_lifecycle.sql
```

## Dockerfile

```dockerfile
FROM postgres:16
COPY init/ /docker-entrypoint-initdb.d/
```

## Migration 001 — Users

```sql
-- database/init/001_create_users.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

## Migration 002 — Projects

```sql
-- database/init/002_create_projects.sql

CREATE TABLE projects (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    repository    VARCHAR(500),
    docker_image  VARCHAR(500) NOT NULL,
    auto_stop_min INTEGER NOT NULL DEFAULT 60,
    status        VARCHAR(50) NOT NULL DEFAULT 'stopped',
    container_id  VARCHAR(255),
    port          INTEGER,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_status  ON projects(status);
```

## Migration 003 — Lifecycle fields

```sql
-- database/init/003_add_lifecycle.sql

ALTER TABLE projects
    ADD COLUMN started_at      TIMESTAMP WITH TIME ZONE,
    ADD COLUMN last_accessed_at TIMESTAMP WITH TIME ZONE;
```

## GORM Models (Go structs — for reference)

```go
// models/user.go
type User struct {
    ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
    Email        string    `gorm:"uniqueIndex;not null"`
    PasswordHash string    `gorm:"not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// models/project.go
type Project struct {
    ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
    UserID        uuid.UUID  `gorm:"type:uuid;not null"`
    Title         string     `gorm:"not null"`
    Description   string
    Repository    string
    DockerImage   string     `gorm:"not null"`
    AutoStopMin   int        `gorm:"default:60"`
    Status        string     `gorm:"default:'stopped'"`  // stopped | pulling | running | error
    ContainerID   *string
    Port          *int
    StartedAt     *time.Time
    LastAccessedAt *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

## Status Values

| Value   | Meaning                              |
|---------|--------------------------------------|
| stopped | Container is not running             |
| pulling | Docker image is being pulled         |
| running | Container is active and accessible   |
| error   | Container failed to start            |

## Data Relationships

```
users (1) ──── (many) projects
```

Each project belongs to exactly one user.
Users can only access their own projects (enforced at API layer).
Deleting a user cascades to delete all their projects.

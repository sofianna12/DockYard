# BACKEND.md — Go Backend Implementation Guide

## Tech Stack

| Package | Purpose |
|---------|---------|
| github.com/gin-gonic/gin | HTTP router and middleware |
| gorm.io/gorm | ORM for PostgreSQL |
| gorm.io/driver/postgres | GORM PostgreSQL driver |
| github.com/golang-jwt/jwt/v5 | JWT creation and validation |
| golang.org/x/crypto/bcrypt | Password hashing |
| github.com/docker/docker/client | Docker SDK for Go |
| github.com/google/uuid | UUID generation |
| github.com/joho/godotenv | .env file loading |

## File Structure

```
backend/
├── Dockerfile
├── main.go
├── go.mod
├── go.sum
├── .env
├── config/
│   └── config.go
├── db/
│   └── postgres.go
├── models/
│   ├── user.go
│   └── project.go
├── handlers/
│   ├── auth.go
│   ├── projects.go
│   └── docker.go
├── middleware/
│   └── auth.go
└── services/
    ├── docker/
    │   ├── client.go
    │   ├── container.go
    │   └── image.go
    └── cleanup/
        └── worker.go
```

## Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o dockyard .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/dockyard .
EXPOSE 8080
CMD ["./dockyard"]
```

## go.mod

```
module dockyard

go 1.22

require (
    github.com/gin-gonic/gin v1.10.0
    gorm.io/gorm v1.25.10
    gorm.io/driver/postgres v1.5.7
    github.com/golang-jwt/jwt/v5 v5.2.1
    golang.org/x/crypto v0.22.0
    github.com/docker/docker v26.1.0+incompatible
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
)
```

## main.go

```go
package main

import (
    "log"
    "dockyard/config"
    "dockyard/db"
    "dockyard/handlers"
    "dockyard/middleware"
    "dockyard/services/cleanup"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
)

func main() {
    godotenv.Load()

    cfg := config.Load()
    database := db.Connect(cfg.DBURL)

    // Start background cleanup worker
    go cleanup.StartWorker(database)

    r := gin.Default()

    // CORS middleware
    r.Use(func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    })

    // Auth routes (public)
    auth := r.Group("/api/auth")
    {
        auth.POST("/register", handlers.Register(database))
        auth.POST("/login",    handlers.Login(database, cfg))
    }

    // Project routes (protected)
    api := r.Group("/api", middleware.AuthRequired(cfg.JWTSecret))
    {
        api.GET("/projects",              handlers.GetProjects(database))
        api.POST("/projects",             handlers.CreateProject(database))
        api.GET("/projects/:id",          handlers.GetProject(database))
        api.PUT("/projects/:id",          handlers.UpdateProject(database))
        api.DELETE("/projects/:id",       handlers.DeleteProject(database))
        api.POST("/projects/:id/launch",  handlers.LaunchContainer(database))
        api.POST("/projects/:id/stop",    handlers.StopContainer(database))
        api.GET("/projects/:id/status",   handlers.GetStatus(database))
    }

    log.Printf("Server running on port %s", cfg.Port)
    r.Run(":" + cfg.Port)
}
```

## config/config.go

```go
package config

import "os"

type Config struct {
    Port                   string
    DBURL                  string
    JWTSecret              string
    IdleTimeoutMinutes     int
    CleanupIntervalMinutes int
}

func Load() *Config {
    return &Config{
        Port:                   getEnv("PORT", "8080"),
        DBURL:                  getEnv("DB_URL", "postgres://dockyard:dockyard@database:5432/dockyard"),
        JWTSecret:              getEnv("JWT_SECRET", "changeme"),
        IdleTimeoutMinutes:     getEnvInt("IDLE_TIMEOUT_MINUTES", 60),
        CleanupIntervalMinutes: getEnvInt("CLEANUP_INTERVAL_MINUTES", 5),
    }
}

func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}

func getEnvInt(key string, fallback int) int {
    // parse os.Getenv(key) as int, return fallback on error
    ...
}
```

## db/postgres.go

```go
package db

import (
    "log"
    "dockyard/models"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func Connect(dsn string) *gorm.DB {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Auto migrate structs (creates tables if not exist)
    db.AutoMigrate(&models.User{}, &models.Project{})

    log.Println("Database connected")
    return db
}
```

## handlers/auth.go

```go
package handlers

// Register
// POST /api/auth/register
// Body: { "email": "...", "password": "..." }
// - Validate email + password not empty
// - Check email not already taken
// - Hash password with bcrypt cost 12
// - Insert user into DB
// - Return 201 { "message": "registered" }

// Login
// POST /api/auth/login
// Body: { "email": "...", "password": "..." }
// - Find user by email
// - Compare password with bcrypt
// - Generate JWT with claims: { user_id, email, exp: now+24h }
// - Return 200 { "token": "..." }
```

## handlers/projects.go

```go
package handlers

// GetProjects
// GET /api/projects
// - Get user_id from JWT context
// - SELECT * FROM projects WHERE user_id = ?
// - Return 200 [ ...projects ]

// CreateProject
// POST /api/projects
// Body: { title, description, repository, docker_image, auto_stop_min }
// - Validate docker_image not empty
// - Insert project with user_id from JWT
// - Return 201 { project }

// GetProject
// GET /api/projects/:id
// - SELECT * FROM projects WHERE id = ? AND user_id = ?
// - Update last_accessed_at = NOW()
// - Return 200 { project }

// UpdateProject
// PUT /api/projects/:id
// Body: { title, description, repository, docker_image, auto_stop_min }
// - Verify ownership
// - Update fields
// - Return 200 { project }

// DeleteProject
// DELETE /api/projects/:id
// - Verify ownership
// - If status == "running": stop and remove container first
// - Delete from DB
// - Return 204
```

## handlers/docker.go

```go
package handlers

// LaunchContainer
// POST /api/projects/:id/launch
// - Verify ownership
// - If already running: return current port
// - Set status = "pulling"
// - Pull image if not present (services/docker/image.go)
// - Run container with -P (random port) (services/docker/container.go)
// - Get assigned host port
// - Update project: status=running, container_id, port, started_at, last_accessed_at
// - Return 200 { "port": 32768, "url": "http://localhost:32768" }

// StopContainer
// POST /api/projects/:id/stop
// - Verify ownership
// - docker stop + docker rm
// - Update project: status=stopped, container_id=null, port=null
// - Return 200

// GetStatus
// GET /api/projects/:id/status
// - Update last_accessed_at = NOW()
// - Return 200 { "status": "running"|"stopped"|"pulling"|"error", "port": 32768 }
```

## services/docker/client.go

```go
package docker

import (
    "github.com/docker/docker/client"
    "log"
    "sync"
)

var (
    instance *client.Client
    once     sync.Once
)

func GetClient() *client.Client {
    once.Do(func() {
        var err error
        instance, err = client.NewClientWithOpts(
            client.FromEnv,
            client.WithAPIVersionNegotiation(),
        )
        if err != nil {
            log.Fatal("Failed to create Docker client:", err)
        }
    })
    return instance
}
```

## services/docker/container.go

```go
package docker

import (
    "context"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/network"
)

// RunContainer starts a container from an image.
// Uses -P to assign random available host port.
// Returns containerID and host port.
func RunContainer(image string) (containerID string, port int, err error) {
    cli := GetClient()
    ctx := context.Background()

    resp, err := cli.ContainerCreate(ctx,
        &container.Config{Image: image},
        &container.HostConfig{PublishAllPorts: true},
        &network.NetworkingConfig{},
        nil,
        "",
    )
    if err != nil {
        return "", 0, err
    }

    err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
    if err != nil {
        return "", 0, err
    }

    port, err = GetContainerPort(resp.ID)
    return resp.ID, port, err
}

// GetContainerPort inspects a running container and returns the first mapped host port.
func GetContainerPort(containerID string) (int, error) {
    cli := GetClient()
    info, err := cli.ContainerInspect(context.Background(), containerID)
    if err != nil {
        return 0, err
    }
    for _, bindings := range info.NetworkSettings.Ports {
        for _, b := range bindings {
            // parse b.HostPort to int
            ...
        }
    }
    return 0, fmt.Errorf("no port found")
}

// StopAndRemoveContainer stops and removes a container.
func StopAndRemoveContainer(containerID string) error {
    cli := GetClient()
    ctx := context.Background()
    cli.ContainerStop(ctx, containerID, container.StopOptions{})
    return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
```

## services/docker/image.go

```go
package docker

import (
    "context"
    "io"
    "github.com/docker/docker/api/types/image"
)

// PullImage pulls a Docker image if not already present locally.
func PullImage(imageName string) error {
    cli := GetClient()
    ctx := context.Background()

    reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
    if err != nil {
        return err
    }
    defer reader.Close()
    io.Copy(io.Discard, reader) // consume output to wait for completion
    return nil
}
```

## services/cleanup/worker.go

```go
package cleanup

import (
    "log"
    "time"
    dockerSvc "dockyard/services/docker"
    "gorm.io/gorm"
)

func StartWorker(db *gorm.DB) {
    ticker := time.NewTicker(5 * time.Minute)
    log.Println("Cleanup worker started")

    for range ticker.C {
        runCleanup(db)
    }
}

func runCleanup(db *gorm.DB) {
    var projects []models.Project

    db.Raw(`
        SELECT * FROM projects
        WHERE status = 'running'
        AND last_accessed_at < NOW() - (auto_stop_min || ' minutes')::INTERVAL
    `).Scan(&projects)

    for _, p := range projects {
        if p.ContainerID == nil {
            continue
        }
        log.Printf("Stopping idle container for project: %s", p.Title)
        dockerSvc.StopAndRemoveContainer(*p.ContainerID)
        db.Model(&p).Updates(map[string]interface{}{
            "status":       "stopped",
            "container_id": nil,
            "port":         nil,
        })
    }
}
```

## middleware/auth.go

```go
package middleware

import (
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func AuthRequired(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })

        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["user_id"])
        c.Next()
    }
}
```

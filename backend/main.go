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

	go cleanup.StartWorker(database)

	r := gin.Default()

	// Health endpoint to verify DB connectivity quickly
	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := database.DB()
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "error": err.Error()})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{"status": "error", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

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

	auth := r.Group("/api/auth", middleware.RateLimit())
	{
		auth.POST("/register", handlers.Register(database))
		auth.POST("/login", handlers.Login(database, cfg))
	}

	api := r.Group("/api", middleware.AuthRequired(cfg.JWTSecret))
	{
		api.GET("/projects", handlers.GetProjects(database))
		api.POST("/projects", handlers.CreateProject(database, cfg))
		api.GET("/projects/:id", handlers.GetProject(database))
		api.PUT("/projects/:id", handlers.UpdateProject(database, cfg))
		api.DELETE("/projects/:id", handlers.DeleteProject(database))
		api.POST("/projects/:id/launch", handlers.LaunchContainer(database, cfg))
		api.POST("/projects/:id/stop", handlers.StopContainer(database))
		api.GET("/projects/:id/status", handlers.GetStatus(database))
		api.GET("/projects/:id/logs", handlers.GetLogs(database))
		api.GET("/projects/:id/files", handlers.ListFiles(database, cfg))
		api.POST("/projects/:id/files", handlers.UploadFile(database, cfg))
		api.DELETE("/projects/:id/files/:filename", handlers.DeleteFile(database, cfg))
	}

	// WebSocket terminal — auth via ?token= query param (no JWT middleware)
	r.GET("/ws/projects/:id/terminal", handlers.TerminalWS(database, cfg))

	log.Printf("Server running on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}

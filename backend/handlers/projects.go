package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"dockyard/config"
	"dockyard/models"
	"dockyard/services/crypto"
	dockerSvc "dockyard/services/docker"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getUserID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("user_id")
	if !exists {
		return uuid.UUID{}, fmt.Errorf("user_id not in context")
	}
	str, ok := raw.(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("user_id has wrong type")
	}
	return uuid.Parse(str)
}

func GetProjects(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit
		q := c.Query("q")

		var projects []models.Project
		var total int64
		query := db.Model(&models.Project{}).Where("user_id = ?", userID)
		if q != "" {
			like := "%" + q + "%"
			query = query.Where("title ILIKE ? OR description ILIKE ?", like, like)
		}
		query.Count(&total)
		query.Offset(offset).Limit(limit).Find(&projects)

		c.JSON(http.StatusOK, gin.H{
			"data":  projects,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

func CreateProject(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var input struct {
			Title            string `json:"title" binding:"required"`
			Description      string `json:"description"`
			Repository       string `json:"repository"`
			DockerImage      string `json:"docker_image" binding:"required"`
			RegistryUser     string `json:"registry_user"`
			RegistryPassword string `json:"registry_password"`
			EnvVars          string `json:"env_vars"`
			Mounts           string `json:"mounts"`
			ContainerPort    int    `json:"container_port"`
			TerminalMode     bool   `json:"terminal_mode"`
			WebTerminal      bool   `json:"web_terminal"`
			AutoStopMin      int    `json:"auto_stop_min"`
			Scheme           string `json:"scheme"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		autoStop := input.AutoStopMin
		if autoStop == 0 {
			autoStop = 60
		}

		encPassword, err := crypto.Encrypt(cfg.EncryptionKey, input.RegistryPassword)
		if err != nil {
			log.Printf("failed to encrypt registry password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		scheme := input.Scheme
		if scheme != "https" {
			scheme = "http"
		}

		project := models.Project{
			ID:               uuid.New(),
			UserID:           userID,
			Title:            input.Title,
			Description:      input.Description,
			Repository:       input.Repository,
			DockerImage:      input.DockerImage,
			RegistryUser:     input.RegistryUser,
			RegistryPassword: encPassword,
			EnvVars:          input.EnvVars,
			Mounts:           input.Mounts,
			ContainerPort:    input.ContainerPort,
			TerminalMode:     input.TerminalMode,
			WebTerminal:      input.WebTerminal,
			AutoStopMin:      autoStop,
			Status:           "stopped",
			Scheme:           scheme,
		}

		if err := db.Create(&project).Error; err != nil {
			log.Printf("failed to create project: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
			return
		}

		c.JSON(http.StatusCreated, project)
	}
}

func GetProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		now := time.Now()
		db.Model(&project).Update("last_accessed_at", now)
		project.LastAccessedAt = &now

		c.JSON(http.StatusOK, project)
	}
}

func UpdateProject(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		var input struct {
			Title            string `json:"title"`
			Description      string `json:"description"`
			Repository       string `json:"repository"`
			DockerImage      string `json:"docker_image"`
			RegistryUser     string `json:"registry_user"`
			RegistryPassword string `json:"registry_password"`
			EnvVars          string `json:"env_vars"`
			Mounts           string `json:"mounts"`
			ContainerPort    int    `json:"container_port"`
			TerminalMode     *bool  `json:"terminal_mode"`
			WebTerminal      *bool  `json:"web_terminal"`
			AutoStopMin      int    `json:"auto_stop_min"`
			Scheme           string `json:"scheme"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]interface{}{}
		if input.Title != "" {
			updates["title"] = input.Title
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}
		if input.Repository != "" {
			updates["repository"] = input.Repository
		}
		if input.DockerImage != "" {
			updates["docker_image"] = input.DockerImage
		}
		if input.RegistryUser != "" {
			updates["registry_user"] = input.RegistryUser
		}
		if input.RegistryPassword != "" {
			encPassword, err := crypto.Encrypt(cfg.EncryptionKey, input.RegistryPassword)
			if err != nil {
				log.Printf("failed to encrypt registry password: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
			updates["registry_password"] = encPassword
		}
		updates["env_vars"] = input.EnvVars
		updates["mounts"] = input.Mounts
		if input.ContainerPort >= 0 {
			updates["container_port"] = input.ContainerPort
		}
		if input.TerminalMode != nil {
			updates["terminal_mode"] = *input.TerminalMode
		}
		if input.WebTerminal != nil {
			updates["web_terminal"] = *input.WebTerminal
		}
		if input.AutoStopMin > 0 {
			updates["auto_stop_min"] = input.AutoStopMin
		}
		switch input.Scheme {
		case "https", "http":
			updates["scheme"] = input.Scheme
		}

		if err := db.Model(&project).Updates(updates).Error; err != nil {
			log.Printf("failed to update project: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update project"})
			return
		}

		// Re-fetch to return fresh data
		db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project)
		c.JSON(http.StatusOK, project)
	}
}

func DeleteProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		if project.Status == "running" && project.ContainerID != nil {
			if err := dockerSvc.StopAndRemoveContainer(*project.ContainerID); err != nil {
				log.Printf("failed to stop container on delete: %v", err)
			}
		}

		if err := db.Delete(&project).Error; err != nil {
			log.Printf("failed to delete project: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
